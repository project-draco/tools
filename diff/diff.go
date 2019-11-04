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
	var (
		minimumSupportCount [2]int
		minimumConfidence   [2]float64
	)
	name := flag.String("name", "", "name")
	granularity := flag.String("granularity", "fine", "fine|coarse")
	for i := 0; i < 2; i++ {
		flag.IntVar(
			&minimumSupportCount[i],
			fmt.Sprintf("minimum-support-count-%v", i),
			2,
			fmt.Sprintf("minimum support count for the file %v", i+1),
		)
		flag.Float64Var(
			&minimumConfidence[i],
			fmt.Sprintf("minimum-confidence-%v", i),
			0.5,
			fmt.Sprintf("minimum confidence for the file %v", i+1),
		)
	}
	object := flag.String("object", "dependency", "dependency|entity")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf(
			"usage: %v [options] <file1> <file2>\n", path.Base(os.Args[0]),
		)
		os.Exit(1)
	}

	if *name != "" {
		*name += ","
	}

	keyfn := func(e entity.Entity) string {
		return e.QueryString()
	}
	if *granularity == "coarse" {
		keyfn = func(e entity.Entity) string {
			return e.Filename()
		}
	}

	addfn := func(from, to string, result map[[2]string]bool) {
		result[[2]string{from, to}] = true
	}
	if *object == "entity" {
		addfn = func(from, to string, result map[[2]string]bool) {
			result[[2]string{from, from}] = true
			result[[2]string{to, to}] = true
		}
	}

	var (
		readers [2]io.Reader
		objects [2]map[[2]string]bool
		err     error
	)
	for i := range readers {
		readers[i], err = os.Open(flag.Arg(i))
		if err != nil {
			log.Fatalf("could not open dependency file: %v", err)
			os.Exit(1)
		}
		objects[i], err = createMapFromDependencyFile(
			readers[i],
			keyfn,
			addfn,
			minimumSupportCount[i],
			minimumConfidence[i],
		)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}
	if len(objects[0]) == 0 || len(objects[1]) == 0 {
		log.Fatalf(
			"empty dependencies file: %v, %v",
			len(objects[0]),
			len(objects[1]),
		)
	}
	onlyOnFirst, onlyOnSecond, onBoth := diff(objects)
	fmt.Printf(
		"%v%v,%v,%v,%v\n",
		*name,
		float64(len(onlyOnFirst))/float64(len(objects[0])),
		float64(len(onBoth))/float64(len(objects[0])),
		float64(len(onlyOnSecond))/float64(len(objects[1])),
		float64(len(onBoth))/float64(len(objects[1])),
	)
}

func createMapFromDependencyFile(
	r io.Reader,
	keyfn func(entity.Entity) string,
	addfn func(from, to string, result map[[2]string]bool),
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
		addfn(keyfn(from), keyfn(to), result)
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
		inv := [2]string{dep[1], dep[0]}
		if dependencies[1][dep] || dependencies[1][inv] {
			onBoth = append(onBoth, dep)
		} else {
			onlyOnFirst = append(onlyOnFirst, dep)
		}
	}
	for dep := range dependencies[1] {
		inv := [2]string{dep[1], dep[0]}
		if !dependencies[0][dep] && !dependencies[0][inv] {
			onlyOnSecond = append(onlyOnSecond, dep)
		}
	}
	return onlyOnFirst, onlyOnSecond, onBoth
}
