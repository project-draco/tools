package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/project-draco/naming"
)

type root struct {
	Package []struct {
		Name      string `xml:"name"`
		Confirmed string `xml:"confirmed,attr"`
		Class     []struct {
			Name      string       `xml:"name"`
			Confirmed string       `xml:"confirmed,attr"`
			Inbound   []dependency `xml:"inbound"`
			Outbound  []dependency `xml:"outbound"`
			Feature   []struct {
				Name      string       `xml:"name"`
				Type      string       `xml:"type,attr"`
				Confirmed string       `xml:"confirmed,attr"`
				Inbound   []dependency `xml:"inbound"`
				Outbound  []dependency `xml:"outbound"`
			} `xml:"feature"`
		} `xml:"class"`
	} `xml:"package"`
}

type dependency struct {
	Type      string `xml:"type,attr"`
	Confirmed string `xml:"confirmed,attr"`
	Name      string `xml:",chardata"`
}

func main() {
	inheritanceFlag := flag.Bool("inheritance", false, "")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: depfind-converter <xml file>")
		os.Exit(1)
	}
	input, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(fmt.Sprintf("could not open file '%v'\n", flag.Arg(0)))
	}
	r := &root{}
	err = xml.NewDecoder(input).Decode(r)
	if err != nil {
		log.Fatal(
			fmt.Sprintf("could not read file '%v': %v\n", flag.Arg(0), err),
		)
	}
	for _, pkg := range r.Package {
		if pkg.Confirmed == "no" {
			continue
		}
		for _, class := range pkg.Class {
			if class.Confirmed == "no" {
				continue
			}
			superClasses := []string{}
			for _, each := range class.Outbound {
				if each.Confirmed == "yes" && each.Type == "class" {
					superClasses = append(superClasses, each.Name)
				}
			}
			if *inheritanceFlag {
				for _, sc := range superClasses {
					from := naming.JavaClassToHR(class.Name)
					to := naming.JavaClassToHR(sc)
					if from == "" || to == "" {
						continue
					}
					fmt.Printf("%v\t%v\n", from, to)
				}
				continue
			}
			for _, feature := range class.Feature {
				for _, dep := range feature.Outbound {
					sameClass := strings.HasPrefix(dep.Name, class.Name)
					fromSuperClass := false
					for _, sc := range superClasses {
						if strings.HasPrefix(dep.Name, sc) {
							fromSuperClass = true
							break
						}
					}
					if (dep.Confirmed == "yes" || sameClass || fromSuperClass) &&
						dep.Type == "feature" {
						from := naming.JavaToHR(feature.Name)
						to := naming.JavaToHR(dep.Name)
						if from == "" || to == "" {
							continue
						}
						fmt.Printf("%v\t%v\n", from, to)
					}
				}
			}
		}
	}
}
