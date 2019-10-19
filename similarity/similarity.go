package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	parser "github.com/project-draco/pkg/refminer-parser"

	"github.com/project-draco/naming"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %v <refactorings> <suggestions>", path.Base(os.Args[0]))
	}
	f1, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("could not open refactorings file: %v", err)
	}
	f2, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatalf("could not open suggestions file: %v", err)
	}
	coefficients := make(map[string]float64)
	refactorings, err := parser.Parse(f1)
	scan := bufio.NewScanner(f2)
	for scan.Scan() {
		suggRec := strings.Split(scan.Text(), " ")
		suggFrom := suggRec[0]
		for _, r := range refactorings {
			if mm, ok := r.(*parser.MoveMethod); ok {
				suggFile := naming.FileFromHR(suggFrom)
				refFile := naming.FileFromHR(naming.JavaToHR(mm.From))
				if strings.Contains(suggFile, refFile) {
					coefficients[scan.Text()] += 1.0
				}
				suggPkg := suggFile[:strings.LastIndex(suggFile, "_")]
				refPkg := refFile[:strings.LastIndex(refFile, "_")]
				if strings.Contains(suggPkg, refPkg) {
					coefficients[scan.Text()] += 1.0
				}
			}
		}
	}
	if scan.Err() != nil {
		log.Fatalf("could not read suggestions: %v", scan.Err())
	}
	fmt.Println(coefficients)
}
