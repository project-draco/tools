package main

import (
	"strings"
	"testing"

	"github.com/awalterschulze/gographviz"
)

func TestFindEvolutionarySmells(t *testing.T) {
	cg := gographviz.NewGraph()
	ast, err := gographviz.ParseString(`
		digraph {
			subgraph cluster1 {
				"p_C1.java/[CN]/C1/[MT]/m1()"
				"p_C2.java/[CN]/C2/[MT]/m2()";
			}
		}`)
	checkT(t, err)
	err = gographviz.Analyse(ast, cg)
	checkT(t, err)
	sdreader := strings.NewReader(`
		p_C1.java/[CN]/C1/[MT]/m11() p_C1.java/[CN]/C1/[MT]/m12()
		p_C3.java/[CN]/C3/[MT]/m3() p_C1.java/[CN]/C1/[MT]/m1()
	`)
	sdfinder, err := newFinder(sdreader, nil)
	checkT(t, err)
	ccdreader := strings.NewReader(`
		p_C1.java/[CN]/C1/[MT]/m1() p_C2.java/[CN]/C2/[MT]/m2()
	`)
	ccfinder, err := newFinder(ccdreader, nil)
	checkT(t, err)
	smells, err := findEvolutionarySmellsUsingClusters(sdreader, cg, sdfinder, ccfinder, nil)
	smells = searchCandidates(smells, sdreader, sdfinder, ccfinder)
	checkT(t, err)
	if len(smells) != 1 {
		t.Errorf("Expected %v but was %v", 1, len(smells))
	} else {
		if smells[0].entity != "p_C1.java/[CN]/C1/[MT]/m1()" {
			t.Errorf("Expected %v but was %v", "p_C1.java/[CN]/C1/[MT]/m1()", smells[0].entity)
		}
		if smells[0].target != "C2" {
			t.Errorf("Expected %v but was %v", "C2", smells[0].target)
		}
		if smells[0].depcount != 2 {
			t.Errorf("Expected %v but was %v", 2, smells[0].depcount)
		}
	}

	ccdreader = strings.NewReader(`
		p_C1.java/[CN]/C1/[MT]/m1() p_C2.java/[CN]/ 2
	`)
	ccfinder, err = newFinder(ccdreader, nil)
	checkT(t, err)
	smells, err = findEvolutionarySmellsUsingDependencies(
		sdreader, ccdreader, sdfinder, ccfinder, nil, nil, 0, 0,
	)
	checkT(t, err)
	if len(smells) != 1 {
		t.Errorf("Expected %v but was %v", 1, len(smells))
	} else {
		if smells[0].entity != "p_C1.java/[CN]/C1/[MT]/m1()" {
			t.Errorf("Expected %v but was %v", "p_C1.java/[CN]/C1/[MT]/m1()", smells[0].entity)
		}
	}
}

func checkT(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
