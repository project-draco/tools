package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	scanner "github.com/project-draco/pkg/dependency-scanner"
	"github.com/project-draco/pkg/entity"
)

func main() {
	minimumSupportCount := flag.Int(
		"min-support-count", 2, "minimum support count",
	)
	minimumConfidence := flag.Float64(
		"min-confidence", 0.5, "minimum confidence",
	)
	granularity := flag.String("granularity", "fine", "fine|coarse")
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

	var result = make(map[[2]string]struct{})
	scan := scanner.NewDependencyScannerWithFilter(
		reader,
		*minimumSupportCount,
		*minimumConfidence,
	)
	for scan.Scan() {
		d := scan.Dependency()
		for _, ent := range []entity.Entity{
			entity.Entity(d.From[0]),
			entity.Entity(d.To),
		} {
			if string(ent) == "" {
				continue
			}
			pkg, class := ent.Path(), ent.Name()+"_"+strings.Join(ent.Parameters(), "_")
			if len(*granularity) > 0 && (*granularity)[0] == 'c' {
				pkg = ent.Path()[:strings.LastIndex(ent.Path(), "_")]
				class = ent.Classname()
			}
			result[[2]string{pkg, class}] = struct{}{}
		}
	}
	if scan.Err() != nil {
		log.Fatalf("could not read dependency file: %v", scan.Err())
		os.Exit(1)
	}

	for ent := range result {
		fmt.Printf("%v %v\n", ent[0], ent[1])
	}
}
