package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/project-draco/pkg/entity"

	scanner "github.com/project-draco/pkg/dependency-scanner"
)

func main() {
	granularity := flag.String("granularity", "fine", "fine|coarse")
	minimumSupportCount := flag.Int(
		"minimum-support-count", 2, "minimum support count",
	)
	minimumConfidence := flag.Float64(
		"minimum-confidence", 0.5, "minimum confidence",
	)
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf(
			"usage: %v [options] <file1> <file2>\n", path.Base(os.Args[0]),
		)
		os.Exit(1)
	}

	keyfn := func(e entity.Entity) string {
		return e.QueryString()
	}
	if *granularity == "coarse" {
		keyfn = func(e entity.Entity) string {
			return e.Filename()
		}
	}

	var (
		readers      [2]io.Reader
		dependencies [2]map[[2]string]bool
		err          error
	)
	for i := range readers {
		readers[i], err = os.Open(flag.Arg(i))
		if err != nil {
			log.Fatalf("could not open dependency file: %v", err)
			os.Exit(1)
		}
		minSC := 0
		minConf := 0.0
		if i == 0 {
			minSC = *minimumSupportCount
			minConf = *minimumConfidence
		}
		dependencies[i], err = createMapFromDependencyFile(
			readers[i],
			keyfn,
			minSC,
			minConf,
		)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}
	if len(dependencies[0]) == 0 || len(dependencies[1]) == 0 {
		log.Fatalf(
			"empty dependencies file: %v, %v",
			len(dependencies[0]),
			len(dependencies[1]),
		)
	}
	onlyOnFirst, onlyOnSecond, onBoth := diff(dependencies)
	fmt.Println(
		float64(len(onlyOnFirst))/float64(len(dependencies[0])),
		float64(len(onBoth))/float64(len(dependencies[0])),
		float64(len(onlyOnSecond))/float64(len(dependencies[1])),
		float64(len(onBoth))/float64(len(dependencies[1])),
	)
}

func createMapFromDependencyFile(
	r io.Reader,
	keyfn func(entity.Entity) string,
	minimumSupportCount int,
	minimumConfidence float64,
) (
	map[[2]string]bool,
	error,
) {
	var result = make(map[[2]string]bool)
	scan := scanner.NewDependencyScannerWithFilter(
		r,
		minimumSupportCount,
		minimumConfidence,
	)
	for scan.Scan() {
		d := scan.Dependency()
		from := entity.Entity(d.From[0])
		to := entity.Entity(d.To)
		result[[2]string{keyfn(from), keyfn(to)}] = true
	}
	if scan.Err() != nil {
		return nil, fmt.Errorf(
			"could not read dependency file: %v", scan.Err(),
		)
	}
	return result, nil
}

func diff(
	dependencies [2]map[[2]string]bool,
) (
	onlyOnFirst [][2]string,
	onlyOnSecond [][2]string,
	onBoth [][2]string,
) {
	for dep := range dependencies[0] {
		if dependencies[1][dep] {
			onBoth = append(onBoth, dep)
		} else {
			onlyOnFirst = append(onlyOnFirst, dep)
		}
	}
	for dep := range dependencies[1] {
		if !dependencies[0][dep] {
			onlyOnSecond = append(onlyOnSecond, dep)
		}
	}
	return onlyOnFirst, onlyOnSecond, onBoth
}
