package main

import (
	"io"
	"strings"

	"github.com/awalterschulze/gographviz"
)

type smell struct {
	entity, target string
	depcount       int
	candidates     []struct {
		name     string
		depcount int
	}
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
			filename := entity(v).filename()
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
					sdfinder, entity(e), filename, inh, []string{},
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
	precondition func(e entity, fromfilename, tofilename string, ignore []string) bool,
	inh *inheritance,
	searchCandidates bool,
	minimumSupportCount int,
) ([]smell, error) {
	// entitiesWithSmell maps entities to filenames it relates to
	entitiesWithSmell := map[string][]string{}
	ccdReader.Seek(0, 0)
	s := newDependencyScanner(ccdReader)
next:
	for s.Scan() {
		d := s.Dependency()
		if d.SupportCount < minimumSupportCount {
			continue
		}
		for _, from := range d.From {
			if entity(from).filename() == entity(d.To).filename() {
				continue next
			}
			if inh.IsSuperclass(entity(from).filename()) {
				continue next
			}
			if precondition != nil &&
				!precondition(
					entity(from),
					entity(from).filename(),
					entity(d.To).filename(),
					d.From,
				) {
				continue next
			}
		}
		k := strings.Join(d.From, "\t")
		entitiesWithSmell[k] = append(
			entitiesWithSmell[k],
			entity(d.To).filename(),
		)
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	smells := make([]smell, 0, len(entitiesWithSmell))
	for e, files := range entitiesWithSmell {
		smells = addSmell(smells, e, sdReader, entity(e).filename(), files,
			sdfinder, ccdfinder, searchCandidates)
	}

	return smells, nil
}

func haveAtLeastOneStaticDependencyButNoneWithinTheSameFileOrTheSuperclass(
	f *finder,
	e entity,
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
		if entity(dependency).filename() == filename {
			return false
		}
		if inh.IsSuperclass(entity(dependency).filename()) {
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
	ffnn []string,
	sdfinder, ccdfinder *finder,
	searchCandidates bool,
) []smell {
	var cs []struct {
		name     string
		depcount int
	}
	for _, fn := range ffnn {
		cs = append(cs, struct {
			name     string
			depcount int
		}{fn, 0})
	}
	smell := smell{entity: strings.TrimSpace(e), candidates: cs}
	if searchCandidates {
		bestCandidate, maxDependenciesToBeRemoved := findBestCandidate(
			sdReader, entity(e).queryString(), filename, ffnn,
			sdfinder, ccdfinder, &smell)
		if bestCandidate != "" && maxDependenciesToBeRemoved >= 0 {
			smell.target = bestCandidate
			smell.depcount = maxDependenciesToBeRemoved
			return append(smells, smell)
		}
	} else {
		return append(smells, smell)
	}
	return smells
}

func findBestCandidate(
	sdReader io.ReadSeeker,
	querystring, filename string,
	candidatesFileNames []string,
	sdfinder, ccdfinder *finder,
	smell *smell,
) (string, int) {
	dependenciesBefore := -1
	if sdReader != nil {
		sdReader.Seek(0, 0)
		sdGraph, err := newGraph(nil, sdReader)
		check(err, "could not create graph from static dependencies")
		dependenciesBefore = sdGraph.edgesCount()
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
			ccdfinder.dependenciesBetweenFiles(filename, candidatefilename)...)
		foundAnotherDependencyNotInvolvingCurrentEntity := false
		dependenciesInvolvingCurrentEntity := 0
		for _, d := range dbf {
			if entity(d[0]).queryString() != querystring &&
				entity(d[1]).queryString() != querystring {
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
				continue
			}
		}
		if smell != nil {
			smell.candidates = append(smell.candidates, struct {
				name     string
				depcount int
			}{
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
