package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/project-draco/naming"
	"github.com/project-draco/pkg/entity"
)

type config struct {
	dotfiles []string
	staticmdg,
	cochangemdg,
	errorsfile,
	inheritancefile,
	fieldtypesfile string
	supplementalRefactorings,
	smells []string
}

var before = map[string]float64{
	"reusability": 1, "flexibility": 1, "understandability": -0.99,
	"reusability2": 1, "flexibility2": 1, "understandability2": -0.99,
	"mpc2": -1, "cbo2": -1, "pc2": -1,
}

func main() {
	output := flag.String("output", "",
		"one of: smells (default), suggestions, metric, count, csv, metapost")
	dotfile := flag.String("dot-file", "", "")
	dotdir := flag.String("dot-dir", "", "")
	minimumSupportCount := flag.Int(
		"minimum-support-count", 2, "minimum support count",
	)
	minimumConfidence := flag.Float64(
		"minimum-confidence", 0.5, "minimum confidence",
	)
	allowToDependOnCurrentClass := flag.Bool(
		"allow-to-depend-on-current-class",
		false,
		"allow method to depend on current class",
	)
	supplementalRefactorings := flag.String("supplemental-refactorings", "", "")
	smells := flag.String("smells", "", "use these smells instead of compute them")
	configfile := flag.String("config", "", "")
	flag.Parse()
	if flag.NArg() < 3 && *configfile == "" {
		fmt.Printf("usage: recommender <static mdg file> <co-change mdg file> <errors file> [<inheritance> <field types>]\n")
		return
	}
	var configs []config
	if *configfile == "" {
		var dotfiles []string
		if *dotdir == "" {
			dotfiles = []string{*dotfile}
		} else {
			f, err := os.Open(*dotdir)
			check(err, "could not open dotdir")
			fis, err := f.Readdir(0)
			check(err, "could not read dotdir")
			for _, fi := range fis {
				dotfiles = append(dotfiles, filepath.Join(*dotdir, fi.Name()))
			}
		}
		configs = append(configs, config{
			dotfiles,
			flag.Arg(0),
			flag.Arg(1),
			flag.Arg(2),
			"",
			"",
			nil,
			splitIfNotEmpty(*smells, "|"),
		})
		if flag.NArg() >= 4 {
			configs[0].inheritancefile = flag.Arg(3)
		}
		if flag.NArg() >= 5 {
			configs[0].fieldtypesfile = flag.Arg(4)
		}
	} else {
		cf, err := os.Open(*configfile)
		check(err, "could not open config")
		defer cf.Close()
		s := bufio.NewScanner(cf)
		for s.Scan() {
			configfields := strings.Split(s.Text(), ";")
			for i := len(configfields); i < 8; i++ {
				configfields = append(configfields, "")
			}
			configs = append(configs, config{
				[]string{configfields[0]},
				configfields[1],
				configfields[2],
				configfields[3],
				configfields[4],
				configfields[5],
				splitIfNotEmpty(configfields[6], "|"),
				splitIfNotEmpty(configfields[7], "|"),
			})
		}
		check(s.Err(), "could not read config")
	}
	var improvements [][]map[string]float64
	if *output == "csv" {
		fmt.Println("subject;sc;ec;sdc;ccdc;cd;cboo;mpco;pco;ro;fo;uo;cbow;mpcw;pcw;rw;fw;uw")
	}
	for i, cfg := range configs {
		computeMetrics := *output == "metric" || *output == "metapost" || *output == "csv"
		allsmells, configImprovements, attributes := doAnalysis(
			cfg,
			*output == "suggestions",
			computeMetrics,
			splitIfNotEmpty(*supplementalRefactorings, "|"),
			*minimumSupportCount,
			*minimumConfidence,
			*allowToDependOnCurrentClass,
		)
		if *output == "metapost" {
			improvements = append(improvements, configImprovements)
		} else if *output == "metric" {
			for _, imp := range configImprovements {
				fmt.Println(imp)
			}
		} else if *output == "csv" {
			fmt.Printf("%v;%v;%v;%v;%v;%v;",
				cfg.dotfiles,
				len(allsmells[0]),
				attributes["entities-count"],
				attributes["static-dependencies-count"],
				attributes["co-change-dependencies-count"],
				attributes["clusters-density"],
			)
			metrics := []string{"cbo2", "mpc2", "pc2",
				"reusability2", "flexibility2", "understandability2"}
			for i := 0; i < 2; i++ {
				for j, metric := range metrics {
					sep := ""
					if i != 1 || j < len(metrics)-1 {
						sep = ";"
					}
					change := ""
					if len(configImprovements) > i {
						change = fmt.Sprintf("%v", configImprovements[i][metric]-before[metric])
					}
					fmt.Printf("%v%v", change, sep)
				}
			}
			fmt.Println()
		} else {
			if *dotdir == "" {
				fmt.Print(cfg.dotfiles[0])
			} else {
				fmt.Print(*dotdir)
			}
			if *output == "count" {
				fmt.Printf(": %v\n", len(allsmells[0]))
			} else {
				fmt.Println()
				for _, s := range allsmells[0] {
					fmt.Println(s)
				}
				if i < len(configs)-1 {
					fmt.Println()
				}
			}
		}
	}
	if *output == "metapost" {
		printMetapost(improvements)
	}
}

func doAnalysis(
	cfg config,
	mustSearchCandidates,
	metric bool,
	supplementalRefactorings []string,
	minimumSupportCount int,
	minimumConfidence float64,
	allowToDependOnCurrentClass bool,
) ([][]smell, []map[string]float64, map[string]float64) {
	var clusteredgraphs []*gographviz.Graph
	for _, dotfile := range cfg.dotfiles {
		if dotfile == "" {
			continue
		}
		var err error
		buf, err := ioutil.ReadFile(dotfile)
		check(err, "could not read dot file")
		ast, err := gographviz.Parse(buf)
		check(err, "could not parse dot file")
		clusteredgraph := gographviz.NewGraph()
		err = gographviz.Analyse(ast, clusteredgraph)
		check(err, "could not analyse dot file")
		clusteredgraphs = append(clusteredgraphs, clusteredgraph)
	}
	f1, err := os.Open(cfg.staticmdg)
	check(err, "could not open static mdg file")
	defer f1.Close()
	f2, err := os.Open(cfg.cochangemdg)
	check(err, "could not open co-change mdg file")
	defer f2.Close()
	f3, err := os.Open(cfg.errorsfile)
	check(err, "could not open errors file")
	defer f3.Close()
	sdfinder, err := newFinder(f1, f3)
	check(err, "could not create static dependencies finder")
	ccdfinder, err := newFinder(f2, nil)
	check(err, "could not create co-change dependencies finder")
	var inh *inheritance
	if cfg.inheritancefile != "" {
		fi, err := os.Open(cfg.inheritancefile)
		check(err, "could not open inheritance file")
		defer fi.Close()
		inh, err = newInheritance(fi)
		check(err, "could not read inheritance file")
	}
	precondition := func(e entity.Entity, fromfilename, tofilename string, ignore []string) bool {
		// relaxes constraint of "evolutionary smell",
		// by allowing method to depend on current class if
		// static dependencies between the source and
		// destination classes of the co-change dependency
		// under analysis already exist
		if allowToDependOnCurrentClass &&
			fromfilename != tofilename &&
			sdfinder.hasDependenciesBetweenFiles(fromfilename, tofilename) {
			return true
		}
		return haveAtLeastOneStaticDependencyButNoneWithinTheSameFileOrTheSuperclass(
			sdfinder, e, fromfilename, inh, ignore,
		)
	}
	var allsmells [][]smell
	if len(cfg.smells) > 0 {
		for _, smellsFilename := range cfg.smells {
			sf, err := os.Open(smellsFilename)
			check(err, "could not open smells file")
			defer sf.Close()
			var smells []smell
			s := bufio.NewScanner(sf)
			for s.Scan() {
				if strings.TrimSpace(s.Text()) == "" {
					continue
				}
				fields := strings.Split(s.Text(), " -> ")
				smells = append(smells, smell{
					entity: fields[0],
					target: fields[1][:strings.Index(fields[1], " (")],
				})
			}
			check(s.Err(), "could not read smells file")
			allsmells = append(allsmells, smells)
		}
	} else if len(clusteredgraphs) == 0 {
		allsmells = [][]smell{{}}
		allsmells[0], err = findEvolutionarySmellsUsingDependencies(
			f1, f2, sdfinder, ccdfinder,
			precondition,
			inh,
			minimumSupportCount,
			minimumConfidence,
		)
		check(err, "could not find smells")
		if mustSearchCandidates {
			allsmells[0] = searchCandidates(allsmells[0], f1, sdfinder, ccdfinder)
		}
	} else {
		allsmells = [][]smell{{}}
		for _, clusteredgraph := range clusteredgraphs {
			ss, err := findEvolutionarySmellsUsingClusters(
				f1, clusteredgraph, sdfinder, ccdfinder, precondition, inh,
			)
			check(err, "could not find smells")
		next_smell:
			for _, s := range ss {
				for i, s_ := range allsmells[0] {
					if s.entity == s_.entity && s.target == s_.target {
						for _, c := range s.candidates {
							for _, c_ := range s.candidates {
								if c.depcount < c_.depcount {
									continue next_smell
								}
							}
						}
						allsmells[0][i] = s
						continue next_smell
					}
				}
				allsmells[0] = append(allsmells[0], s)
			}
		}
		if mustSearchCandidates {
			allsmells[0] = searchCandidates(allsmells[0], f1, sdfinder, ccdfinder)
		}
	}

	var improvements []map[string]float64
	if metric {
		fieldTypesFileName := ""
		if cfg.fieldtypesfile != "" {
			fieldTypesFileName = cfg.fieldtypesfile
		}
		if len(cfg.supplementalRefactorings) > 0 && len(supplementalRefactorings) == 0 {
			supplementalRefactorings = cfg.supplementalRefactorings
		}
		improvements = computeMetrics(sdfinder, ccdfinder, allsmells, inh,
			supplementalRefactorings, fieldTypesFileName, f1, f2)
	}

	var clustersdensitysum, avgclustersdensity float64
	for _, clusteredgraph := range clusteredgraphs {
		clustersdensitysum += density(clusteredgraph, ccdfinder)
	}
	if len(clusteredgraphs) > 0 {
		avgclustersdensity = clustersdensitysum / avgclustersdensity
	}
	attrs := map[string]float64{
		"entities-count":               float64(sdfinder.entitiesCount()),
		"static-dependencies-count":    float64(sdfinder.dependenciesCount()),
		"co-change-dependencies-count": float64(ccdfinder.dependenciesCount()),
		"clusters-density":             avgclustersdensity,
	}

	return allsmells, improvements, attrs
}

func computeMetrics(
	sdfinder, ccdfinder *finder,
	allsmells [][]smell,
	inh *inheritance,
	supplementalRefactorings []string,
	fieldTypesFileName string,
	f1, f2 *os.File,
) []map[string]float64 {
	var fldTypes *fieldTypes
	if fieldTypesFileName != "" {
		fft, err := os.Open(fieldTypesFileName)
		check(err, "could not open field types file")
		defer fft.Close()
		fldTypes, err = newFieldTypes(fft)
		check(err, "could not read field types file")
	}
	var reassignments []map[string]string
	joinedReassignments := map[string]string{}
	for _, smells := range allsmells {
		evolutionaryReassignments := map[string]string{}
		for _, s := range smells {
			if s.target != "" {
				evolutionaryReassignments[entity.Entity(s.entity).QueryString()] = s.target
				joinedReassignments[entity.Entity(s.entity).QueryString()] = s.target
			}
		}
		reassignments = append(reassignments, evolutionaryReassignments)
	}
	for _, sr := range supplementalRefactorings {
		srf, err := os.Open(sr)
		check(err, "could not open supplemental refactorigs file")
		defer srf.Close()
		supplementalReassignments := map[string]string{}
		s := bufio.NewScanner(srf)
		for s.Scan() {
			if strings.TrimSpace(s.Text()) == "" {
				continue
			}
			fields := strings.Split(s.Text(), ";")
			if len(fields) < 2 {
				check(fmt.Errorf("invalid refactoring: %v, %v", s.Text(), sr), "")
			}
			ent := entity.Entity(naming.JavaToHR(fields[0]))
			//TODO: the code bellow checks if the supplemental refactoring will not result in
			// an improvement because another dependency remains after move. We must check if
			// this code is necessary
			bestCandidate, _ := findBestCandidate(
				nil,
				ent.QueryString(),
				ent.Filename(),
				[]string{fields[1]},
				sdfinder,
				ccdfinder,
				nil,
			)
			if bestCandidate != "" {
				supplementalReassignments[ent.QueryString()] = bestCandidate
				joinedReassignments[ent.QueryString()] = bestCandidate
			}
		}
		check(s.Err(), "could not read supplemental refactorings file")
		reassignments = append(reassignments, supplementalReassignments)
	}
	if len(supplementalRefactorings) > 0 {
		reassignments = append(reassignments, joinedReassignments)
	}
	return improvements(reassignments, inh, fldTypes, sdfinder, f1, f2)
}

func printMetapost(allImprovements [][]map[string]float64) {
	symbol := []string{"bullet", "maltese", "blacktriangleright", "blackbowtie", "star", "blacklozenge"}
	color := []string{"blue", "blue", "blue", "orange", "red", "OliveGreen"}
	fmt.Println(`verbatimtex
		%&latex
		\documentclass[60pt]{article}
        \usepackage[dvipsnames]{xcolor}
        \usepackage{amsfonts,amssymb}
        \usepackage{boisik}
		\begin{document}
		etex
		`)
	for i, metric := range []string{
		"reusability", "flexibility", "understandability",
		"reusability2", "flexibility2", "understandability2",
		"mpc2", "cbo2", "pc2"} {
		fmt.Printf("beginfig(%v)\n", i+1)
		if i == 0 {
			fmt.Println("u = 1cm;")
		}
		min, max := math.MaxFloat64, -1.0
		for _, improvements := range allImprovements {
			imin, imax := improvementsBounds(improvements, metric, before[metric])
			min = math.Min(imin, min)
			max = math.Max(imax, max)
		}
		coef := 5.0 / max
		fmt.Printf("draw (0,0)--(%[1]vu,0)--(%[1]vu,5u)--(0,5u)--cycle;\n", len(allImprovements)+1)
		if min < 0 {
			fmt.Printf("draw (0,0)--(%[1]vu,0)--(%[1]vu,%.6[2]fu)--(0,%.6[2]fu)--cycle;\n",
				len(allImprovements)+1, min*coef)
		}
		fmt.Printf("label.lft(btex \\LARGE{$0$} etex,(0,0));\n")
		fmt.Printf("label.lft(btex \\LARGE{$%.6f$} etex scaled 1.5,(0,5u));\n", max*100)
		if min < 0 {
			fmt.Printf("label.lft(btex \\LARGE{$%.6f$} etex scaled 1.5,(0,%.6fu));\n", min*100, min*coef)
		}
		fmt.Printf("label.bot(btex System index etex scaled 2.5, (%.6fu,%.6fu));\n", float64(len(allImprovements)+1)/2, min*coef-1.5)
		for j, improvements := range allImprovements {
			imin, imax := improvementsBounds(improvements, metric, before[metric])
			imax = math.Max(imax, 0)
			imin = math.Min(imin, 0)
			fmt.Printf("draw (%[1]vu,%.6[2]fu)--(%[1]vu,%.6[3]fu);\n", j+1, imin*coef, imax*coef)
			for k := 0; k < len(improvements); k++ {
				value, ok := improvements[k][metric]
				if ok {
					fmt.Printf(
						`label(btex \Huge{$\color{%[4]v}\%[3]v$} etex,(%[1]vu,%.6[2]fu));
    `,
						j+1,
						(value-before[metric])*coef,
						symbol[k],
						color[k],
					)
				}
			}
			fmt.Printf("label.bot(btex %[1]v etex scaled 2.5, (%[1]vu,%.6fu));\n", j+1, min*coef-0.5)
		}
		fmt.Println("endfig;")
	}
	fmt.Println("end;")
	fmt.Println(`
		verbatimtex
		\end{document}
		etex`)
}

func improvementsBounds(improvements []map[string]float64, metric string, before float64) (float64, float64) {
	min, max := math.MaxFloat64, -1.0
	for k := 0; k < len(improvements); k++ {
		value, ok := improvements[k][metric]
		if !ok {
			continue
		}
		delta := value - before
		min = math.Min(delta, min)
		max = math.Max(delta, max)
	}
	return min, max
}

func density(clusteredgraph *gographviz.Graph, ccdfinder *finder) float64 {
	if clusteredgraph == nil || len(clusteredgraph.SubGraphs.SubGraphs) == 0 {
		return 0
	}
	sum := 0.0
	for clustername := range clusteredgraph.SubGraphs.SubGraphs {
		clusterEntities := map[string]string{}
		for _, v := range clusteredgraph.Relations.SortedChildren(clustername) {
			nv := strings.Trim(v, "\"")
			clusterEntities[entity.Entity(nv).QueryString()] = nv
		}
		if len(clusterEntities) <= 1 {
			continue
		}
		count := 0
		for _, v := range clusterEntities {
			deps := ccdfinder.dependenciesOf(entity.Entity(v))
			if deps == nil {
				continue
			}
			for _, d := range deps.outcome {
				if _, ok := clusterEntities[entity.Entity(d).QueryString()]; ok {
					count++
				}
			}
		}
		sum += float64(count) / float64(len(clusterEntities)*(len(clusterEntities)-1))
	}
	return sum / float64(len(clusteredgraph.SubGraphs.SubGraphs))
}

func check(err error, info string) {
	if err != nil {
		log.Fatalf("%v: %v", info, err)
	}
}

func splitIfNotEmpty(str, sep string) []string {
	if str == "" {
		return nil
	}
	return strings.Split(str, sep)
}
