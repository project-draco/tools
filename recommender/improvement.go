package main

import (
	"io"
)

type direction int

const (
	biggerIsBetter direction = iota
	smallerIsBetter
)

func improvements(
	reassignments []map[string]string,
	inh *inheritance,
	ft *fieldTypes,
	sdfinder *finder,
	rr ...io.ReadSeeker,
) []map[string]float64 {
	var result []map[string]float64
	dependencyGraph, err := newGraph(nil, rr[0])
	check(err, "could not read static dependencies")
	dependencyGraph2, err := newGraph(nil, readers(rr)...)
	check(err, "could not read remaining dependencies")
	rr[0].Seek(0, 0)
	structure, err := newStructure(nil, rr[0])
	check(err, "could not create structure from static dependencies")
	before := measure(dependencyGraph, dependencyGraph2, structure, inh, ft, sdfinder)
	for i := range reassignments {
		m := make(map[string]float64)
		result = append(result, m)
		if len(reassignments[i]) == 0 {
			continue
		}
		rr[0].Seek(0, 0)
		refactoredDependenciesGraph, err := newGraph(reassignments[i], rr[0])
		check(err, "could not create graph from static dependencies")
		rr[0].Seek(0, 0)
		refactoredDependenciesGraph2, err := newGraph(reassignments[i], readers(rr)...)
		check(err, "could not create graph from remaining dependencies")
		rr[0].Seek(0, 0)
		refactoredStructure, err := newStructure(reassignments[i], rr[0])
		check(err, "could not create structure from static dependencies")
		after := measure(refactoredDependenciesGraph, refactoredDependenciesGraph2,
			refactoredStructure, inh, ft, sdfinder)
		for metric := range after {
			if before[metric] == 0 {
				m[metric] = 1
			} else {
				m[metric] = after[metric] / before[metric]
			}
		}
		m["reusability"] = -0.25*m["mpc"] + 0.25*m["cam"] + 0.5*m["cis"] + 0.5*m["dsc"]
		m["flexibility"] = 0.25*m["dam"] - 0.25*m["mpc"] + 0.5*m["moa"] + 0.5*m["nop"]
		m["understandability"] = -0.33*1 /*ANA*/ + 0.33*m["dam"] - 0.33*m["mpc"] + 0.33*m["cam"] -
			0.33*m["nop"] - 0.33*m["nom"] - 0.33*m["dsc"]
		m["reusability2"] = -0.25*m["mpc2"] + 0.25*m["cam"] + 0.5*m["cis"] + 0.5*m["dsc"]
		m["flexibility2"] = 0.25*m["dam"] - 0.25*m["mpc2"] + 0.5*m["moa"] + 0.5*m["nop"]
		m["understandability2"] = -0.33*1 /*ANA*/ + 0.33*m["dam"] - 0.33*m["mpc2"] + 0.33*m["cam"] -
			0.33*m["nop"] - 0.33*m["nom"] - 0.33*m["dsc"]
		m["mpc2"] = -1 * m["mpc2"]
		m["cbo2"] = -1 * m["cbo2"]
		m["pc2"] = -1 * m["pc2"]
	}
	return result
}

func readers(rr []io.ReadSeeker) []io.Reader {
	var result []io.Reader
	for _, r := range rr {
		r.Seek(0, 0)
		result = append(result, r)
	}
	return result
}

func measure(
	g *graph,
	g2 *graph,
	s *structure,
	inh *inheritance,
	ft *fieldTypes,
	f *finder,
) map[string]float64 {
	return map[string]float64{
		"pc":   float64(propagationCost(g.successors)),
		"pc2":  float64(propagationCost(g2.successors)),
		"cam":  cam(s),
		"cbo":  cbo(g.successors),
		"cbo2": cbo(g2.successors),
		"cis":  cis(s, f),
		"dam":  dam(s, f),
		"dsc":  dsc(s),
		"moa":  moa(s, ft),
		"mpc":  mpc(g.successors, g.weigths),
		"mpc2": mpc(g2.successors, g2.weigths),
		"nom":  nom(s),
		"nop":  nop(s, inh),
	}
}
