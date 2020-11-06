package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/project-draco/pkg/entity"
)

type config struct {
	dotfile,
	staticmdg,
	cochangemdg,
	errorsfile,
	inheritancefile,
	fieldtypesfile,
	supplementalRefactorings string
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
	configfile := flag.String("config", "", "")
	flag.Parse()
	if flag.NArg() < 4 && *configfile == "" {
		fmt.Printf("usage: recommender <static mdg file> <co-change mdg file> <errors file> [<inheritance> <field types>]\n")
		return
	}
	var configs []config
	if *configfile == "" {
		configs = append(configs, config{*dotfile, flag.Arg(0), flag.Arg(1), flag.Arg(2), "", "", ""})
		if flag.NArg() >= 4 {
			configs[0].inheritancefile = flag.Arg(3)
		}
		if flag.NArg() >= 5 {
			configs[0].fieldtypesfile = flag.Arg(4)
		}
	} else {
		fc, err := os.Open(*configfile)
		check(err, "could not open config")
		defer fc.Close()
		s := bufio.NewScanner(fc)
		for s.Scan() {
			configfields := strings.Split(s.Text(), ";")
			for i := len(configfields); i < 7; i++ {
				configfields = append(configfields, "")
			}
			configs = append(configs, config{
				configfields[0],
				configfields[1],
				configfields[2],
				configfields[3],
				configfields[4],
				configfields[5],
				configfields[6],
			})
		}
		check(s.Err(), "could not read config")
	}
	var ii [][]map[string]float64
	if *output == "csv" {
		fmt.Println("subject;sc;ec;sdc;ccdc;cd;cboo;mpco;pco;ro;fo;uo;cbow;mpcw;pcw;rw;fw;uw")
	}
	for i, cfg := range configs {
		computeMetrics := *output == "metric" || *output == "metapost" || *output == "csv"
		smells, improvements, attributes := doAnalysis(
			cfg,
			*output == "suggestions",
			computeMetrics,
			*supplementalRefactorings,
			*minimumSupportCount,
			*minimumConfidence,
			*allowToDependOnCurrentClass,
		)
		if *output == "metapost" {
			ii = append(ii, improvements)
		} else if *output == "metric" {
			for _, imp := range improvements {
				fmt.Println(imp)
			}
		} else if *output == "csv" {
			fmt.Printf("%v;%v;%v;%v;%v;%v;",
				cfg.dotfile,
				len(smells),
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
					if len(improvements) > i {
						change = fmt.Sprintf("%v", improvements[i][metric]-before[metric])
					}
					fmt.Printf("%v%v", change, sep)
				}
			}
			fmt.Println()
		} else {
			fmt.Print(cfg.dotfile)
			if *output == "count" {
				fmt.Printf(": %v\n", len(smells))
			} else {
				fmt.Println()
				for _, s := range smells {
					fmt.Println(s.entity, s.target, s.depcount, s.candidates)
				}
				if i < len(configs)-1 {
					fmt.Println()
				}
			}
		}
	}
	if *output == "metapost" {
		printMetapost(ii)
	}
}

func doAnalysis(
	args config,
	searchCandidates,
	metric bool,
	supplementalRefactorings string,
	minimumSupportCount int,
	minimumConfidence float64,
	allowToDependOnCurrentClass bool,
) ([]smell, []map[string]float64, map[string]float64) {
	var clusteredgraph *gographviz.Graph
	if args.cochangemdg != "" {
		var err error
		buf, err := ioutil.ReadFile(args.dotfile)
		check(err, "could not read dot file ")
		ast, err := gographviz.Parse(buf)
		check(err, "could not parse dot file")
		clusteredgraph = gographviz.NewGraph()
		err = gographviz.Analyse(ast, clusteredgraph)
		check(err, "could not analyse dot file")
	}
	f1, err := os.Open(args.staticmdg)
	check(err, "could not open static mdg file")
	defer f1.Close()
	f2, err := os.Open(args.cochangemdg)
	check(err, "could not open co-change mdg file")
	defer f2.Close()
	f3, err := os.Open(args.errorsfile)
	check(err, "could not open errors file")
	defer f3.Close()
	sdfinder, err := newFinder(f1, f3)
	check(err, "could not create static dependencies finder")
	ccdfinder, err := newFinder(f2, nil)
	check(err, "could not create co-change dependencies finder")
	var inh *inheritance
	if args.inheritancefile != "" {
		fi, err := os.Open(args.inheritancefile)
		check(err, "could not open inheritance file")
		defer fi.Close()
		inh, err = newInheritance(fi)
		check(err, "could not read inheritance file")
	}
	var smells []smell
	if clusteredgraph == nil {
		smells, err = findEvolutionarySmellsUsingDependencies(
			f1, f2, sdfinder, ccdfinder,
			func(e entity.Entity, fromfilename, tofilename string, ignore []string) bool {
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
			},
			inh, searchCandidates,
			minimumSupportCount,
			minimumConfidence,
		)
	} else {
		smells, err = findEvolutionarySmellsUsingClusters(
			f1, clusteredgraph, sdfinder, ccdfinder, inh, searchCandidates,
		)
	}
	check(err, "could not find smells")

	var ii []map[string]float64
	if metric {
		fieldTypesFileName := ""
		if args.inheritancefile != "" {
			fieldTypesFileName = args.inheritancefile
		}
		if args.supplementalRefactorings != "" && supplementalRefactorings == "" {
			supplementalRefactorings = args.supplementalRefactorings
		}
		ii = computeMetrics(sdfinder, ccdfinder, smells, inh,
			supplementalRefactorings, fieldTypesFileName, f1, f2)
	}

	attrs := map[string]float64{
		"entities-count":               float64(sdfinder.entitiesCount()),
		"static-dependencies-count":    float64(sdfinder.dependenciesCount()),
		"co-change-dependencies-count": float64(ccdfinder.dependenciesCount()),
		"clusters-density":             density(clusteredgraph, ccdfinder),
	}

	return smells, ii, attrs
}

func computeMetrics(
	sdfinder, ccdfinder *finder,
	smells []smell,
	inh *inheritance,
	supplementalRefactorings string,
	fieldTypesFileName string,
	f1, f2 *os.File,
) []map[string]float64 {
	var ii []map[string]float64
	var fldTypes *fieldTypes
	if fieldTypesFileName != "" {
		fft, err := os.Open(fieldTypesFileName)
		check(err, "could not open field types file")
		defer fft.Close()
		fldTypes, err = newFieldTypes(fft)
		check(err, "could not read field types file")
	}
	var reassignments []map[string]string
	evolutionaryReassignments := map[string]string{}
	baselineReassignments := map[string]string{}
	joinReassignments := map[string]string{}
	for _, s := range smells {
		if s.target != "" {
			evolutionaryReassignments[entity.Entity(s.entity).QueryString()] = s.target
			joinReassignments[entity.Entity(s.entity).QueryString()] = s.target
		}
	}
	reassignments = append(reassignments, evolutionaryReassignments)
	if supplementalRefactorings != "" {
		srf, err := os.Open(supplementalRefactorings)
		check(err, "could not opend supplemental refactorigs file")
		defer srf.Close()
		s := bufio.NewScanner(srf)
		for s.Scan() {
			arr := strings.Split(s.Text(), ";")
			ent := entity.Entity(arr[0])
			//TODO: the code bellow checks if the supplemental refactoring will not result in
			// an improvement because another dependency remains after move. We must check if
			// this code is necessary
			bestCandidate, _ := findBestCandidate(
				nil,
				ent.QueryString(),
				ent.Filename(),
				[]string{arr[1]},
				sdfinder,
				ccdfinder,
				nil,
			)
			if bestCandidate != "" {
				baselineReassignments[ent.QueryString()] = bestCandidate
				joinReassignments[ent.QueryString()] = bestCandidate
			}
		}
		check(s.Err(), "could not read supplemental refactorings file")
		reassignments = append(reassignments, baselineReassignments)
		reassignments = append(reassignments, joinReassignments)
	}
	ii = improvements(reassignments, inh, fldTypes, sdfinder, f1, f2)
	return ii
}

func printMetapost(ii [][]map[string]float64) {
	symbol := []string{"bullet", "star", "diamond"}
	fmt.Println(`verbatimtex
		%&latex
		\documentclass[20pt]{article}
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
		for _, improvements := range ii {
			imin, imax := improvementsBounds(improvements, metric, before[metric])
			min = math.Min(imin, min)
			max = math.Max(imax, max)
		}
		coef := 5.0 / max
		fmt.Printf("draw (0,0)--(%[1]vu,0)--(%[1]vu,5u)--(0,5u)--cycle;\n", len(ii)+1)
		if min < 0 {
			fmt.Printf("draw (0,0)--(%[1]vu,0)--(%[1]vu,%[2]vu)--(0,%[2]vu)--cycle;\n",
				len(ii)+1, min*coef)
		}
		fmt.Printf("label.lft(btex \\LARGE{$0$} etex,(0,0));\n")
		fmt.Printf("label.lft(btex \\LARGE{$%.6f$} etex,(0,5u));\n", max)
		if min < 0 {
			fmt.Printf("label.lft(btex \\LARGE{$%.6f$} etex,(0,%vu));\n", min, min*coef)
		}
		for j, improvements := range ii {
			imin, imax := improvementsBounds(improvements, metric, before[metric])
			imax = math.Max(imax, 0)
			imin = math.Min(imin, 0)
			fmt.Printf("draw (%[1]vu,%[2]vu)--(%[1]vu,%[3]vu);\n", j+1, imin*coef, imax*coef)
			for k := 0; k < len(improvements); k++ {
				fmt.Printf(`label(btex \Huge{$\%[3]v$} etex,(%[1]vu,%[2]vu));
				`, j+1, (improvements[k][metric]-before[metric])*coef, symbol[k])
			}
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
		val := improvements[k][metric] - before
		min = math.Min(val, min)
		max = math.Max(val, max)
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
