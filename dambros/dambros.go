package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	scanner "github.com/project-draco/pkg/dependency-scanner"
)

func main() {
	minimumSupportCount := flag.Int(
		"min-support-count", 2, "minimum support count",
	)
	minimumConfidence := flag.Float64(
		"min-confidence", 0.5, "minimum confidence",
	)
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Printf(
			"usage: %v [options] <file>\n", path.Base(os.Args[0]),
		)
		os.Exit(1)
	}

	reader, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("could not open dependency file: %v", err)
		os.Exit(1)
	}

	var result = make(map[string][]int)
	scan := scanner.NewDependencyScannerWithFilter(
		reader,
		*minimumSupportCount,
		*minimumConfidence,
	)
	for scan.Scan() {
		d := scan.Dependency()
		from := d.From[0]
		result[from] = append(result[from], d.SupportCount)
	}
	if scan.Err() != nil {
		log.Fatalf("could not read dependency file: %v", scan.Err())
		os.Exit(1)
	}

	fmt.Println("entity;nocc;soc")
	for ent, supps := range result {
		if ent == "" {
			continue
		}
		nocc := len(supps)
		soc := 0
		for _, s := range supps {
			soc += s
		}
		fmt.Printf("%v;%v;%v\n", ent, nocc, soc)
	}
}
