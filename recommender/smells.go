package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/awalterschulze/gographviz"
	scanner "github.com/project-draco/pkg/dependency-scanner"
	"github.com/project-draco/pkg/entity"
)

type (
	smell struct {
		entity, target string
		depcount       int
		candidates     []candidate
	}
	candidate struct {
		name     string
		depcount int
	}
)

func (s smell) String() string {
	return fmt.Sprintf("%v -> %v (depcount: %v, candidates: %v)", s.entity, s.target, s.depcount, s.candidates)
}

func findEvolutionarySmellsUsingClusters(
	sdReader io.ReadSeeker,
	clusteredgraph *gographviz.Graph,
	sdfinder, ccdfinder *finder,
	inh *inheritance,
	searchCandidates bool,
) ([]smell, error) {
	var smells []smell
	for clustername := range clusteredgraph.SubGraphs.SubGraphs {
		clusterEntitiesByFile := map[string][]string{}
		for _, v := range clusteredgraph.Relations.SortedChildren(clustername) {
			filename := entity.Entity(v).Filename()
			clusterEntitiesByFile[filename] =
				append(clusterEntitiesByFile[filename], strings.Trim(v, "\""))
		}
		if len(clusterEntitiesByFile) == 1 {
			continue
		}
		for filename, entities := range clusterEntitiesByFile {
			if filename == "" || inh.IsSuperclass(filename) {
				continue
			}
			for _, e := range entities {
				// discard entities without static dependencies to attempt to avoid dead code
				// discard entity that depends on another entity inside the same class
				if !haveAtLeastOneStaticDependencyButNoneWithinTheSameFileOrTheSuperclass(
					sdfinder, entity.Entity(e), filename, inh, []string{},
				) {
					continue
				}
				var ffnn []string
				for fn := range clusterEntitiesByFile {
					ffnn = append(ffnn, fn)
				}
				smells = addSmell(smells, e, sdReader, filename, ffnn,
					sdfinder, ccdfinder, searchCandidates)
			}
		}
	}
	return smells, nil
}

func findEvolutionarySmellsUsingDependencies(
	sdReader, ccdReader io.ReadSeeker,
	sdfinder, ccdfinder *finder,
	precondition func(e entity.Entity, fromfilename, tofilename string, ignore []string) bool,
	inh *inheritance,
	searchCandidates bool,
	minimumSupportCount int,
	minimumConfidence float64,
) ([]smell, error) {
	// entitiesWithSmell maps entities to filenames it relates to
	entitiesWithSmell := map[string][]string{}
	ccdReader.Seek(0, 0)
	s := scanner.NewDependencyScanner(ccdReader)
next:
	for s.Scan() {
		d := s.Dependency()
		if d.SupportCount < minimumSupportCount {
			continue
		}
		if d.Confidence < minimumConfidence {
			continue
		}
		for _, from := range d.From {
			if entity.Entity(from).Filename() == entity.Entity(d.To).Filename() {
				continue next
			}
			if inh.IsSuperclass(entity.Entity(from).Filename()) {
				continue next
			}
			if precondition != nil &&
				!precondition(
					entity.Entity(from),
					entity.Entity(from).Filename(),
					entity.Entity(d.To).Filename(),
					d.From,
				) {
				continue next
			}
		}
		k := strings.Join(d.From, "\t")
		entitiesWithSmell[k] = append(
			entitiesWithSmell[k],
			entity.Entity(d.To).Filename(),
		)
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	smells := make([]smell, 0, len(entitiesWithSmell))
	for e, files := range entitiesWithSmell {
		smells = addSmell(
			smells,
			e,
			sdReader,
			entity.Entity(e).Filename(),
			files,
			sdfinder,
			ccdfinder,
			searchCandidates,
		)
	}

	return smells, nil
}

func haveAtLeastOneStaticDependencyButNoneWithinTheSameFileOrTheSuperclass(
	f *finder,
	e entity.Entity,
	filename string,
	inh *inheritance,
	ignore []string,
) bool {
	if f.onErrors(e) {
		return false
	}
	staticdependencies := f.dependenciesOf(e)
	var dd []string
	if staticdependencies != nil {
		dd = append(dd, staticdependencies.outcome...)
		dd = append(dd, staticdependencies.income...)
	}
	if len(dd) == 0 {
		return false
	}
next:
	for _, dependency := range dd {
		for _, ig := range ignore {
			if dependency == ig {
				continue next
			}
		}
		if entity.Entity(dependency).Filename() == filename {
			return false
		}
		if inh.IsSuperclass(entity.Entity(dependency).Filename()) {
			return false
		}
	}
	return true
}

func addSmell(
	smells []smell,
	e string,
	sdReader io.ReadSeeker,
	filename string,
	filenames []string,
	sdfinder, ccdfinder *finder,
	searchCandidates bool,
) []smell {
	var cs []candidate
	for _, fn := range filenames {
		cs = append(cs, candidate{fn, 0})
	}
	smell := smell{entity: strings.TrimSpace(e), candidates: cs}
	if searchCandidates {
		bestCandidate, maxDependenciesToBeRemoved := findBestCandidate(
			sdReader,
			entity.Entity(e).QueryString(),
			filename,
			filenames,
			sdfinder,
			ccdfinder,
			&smell,
		)
		if bestCandidate == "" || maxDependenciesToBeRemoved == 0 {
			return smells
		}
		smell.target = bestCandidate
		smell.depcount = maxDependenciesToBeRemoved
	}
	return append(smells, smell)
}

func findBestCandidate(
	sdReader io.ReadSeeker,
	querystring,
	filename string,
	candidatesFileNames []string,
	sdfinder,
	ccdfinder *finder,
	smell *smell,
) (string, int) {
	dependenciesBefore := -1
	var prevGraph *graph
	if sdReader != nil {
		sdReader.Seek(0, 0)
		sdGraph, err := newGraph(nil, sdReader)
		check(err, "could not create graph from static dependencies")
		dependenciesBefore = sdGraph.edgesCount()
		prevGraph = sdGraph
	}
	var bestCandidate string
	maxDependenciesToBeRemoved := -1
	for _, candidatefilename := range candidatesFileNames {
		if candidatefilename == filename {
			continue
		}
		// TODO: we have to reason if we must check only static dependencies or
		// to check co-change dependencies too
		dbf := append(
			sdfinder.dependenciesBetweenFiles(filename, candidatefilename),
			ccdfinder.dependenciesBetweenFiles(filename, candidatefilename)...,
		)
		foundAnotherDependencyNotInvolvingCurrentEntity := false
		dependenciesInvolvingCurrentEntity := 0
		for _, d := range dbf {
			if entity.Entity(d[0]).QueryString() != querystring &&
				entity.Entity(d[1]).QueryString() != querystring {
				//foundAnotherDependencyNotInvolvingCurrentEntity = true
				//break
			} else {
				dependenciesInvolvingCurrentEntity++
			}
		}
		if sdReader != nil {
			sdReader.Seek(0, 0)
			sdGraph, err := newGraph(map[string]string{querystring: candidatefilename}, sdReader)
			check(err, "could not create graph from static dependencies and refactoring")
			after := sdGraph.edgesCount()
			if dependenciesBefore < after {
				_ = prevGraph
				//fmt.Println(querystring, candidatefilename, prevGraph.diff(sdGraph))
				continue
			}
		}
		if smell != nil {
			smell.candidates = append(smell.candidates, candidate{
				candidatefilename,
				dependenciesInvolvingCurrentEntity,
			})
		}
		if maxDependenciesToBeRemoved < dependenciesInvolvingCurrentEntity {
			if !foundAnotherDependencyNotInvolvingCurrentEntity {
				bestCandidate = candidatefilename
			}
			maxDependenciesToBeRemoved = dependenciesInvolvingCurrentEntity
		}
	}
	return bestCandidate, maxDependenciesToBeRemoved
}
