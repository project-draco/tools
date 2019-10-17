package main

import (
	"io"
)

type graph struct {
	index      map[string]int
	successors [][]int
	weigths    [][]int
}

func newGraph(reassignments map[string]string, readers ...io.Reader) (*graph, error) {
	g := &graph{index: make(map[string]int)}
	for _, r := range readers {
		s := newDependencyScanner(r)
		for s.Scan() {
			d := s.Dependency()
			if len(d.From) != 1 {
				continue
			}
			for _, e := range []string{d.From[0], d.To} {
				if _, ok := g.index[entity(e).filename()]; !ok {
					g.index[entity(e).filename()] = len(g.successors)
					g.successors = append(g.successors, nil)
					g.weigths = append(g.weigths, nil)
				}
			}
			source := entity(d.From[0]).filename()
			destination := entity(d.To).filename()
			if fn, ok := reassignments[entity(d.From[0]).queryString()]; ok {
				source = fn
			}
			if fn, ok := reassignments[entity(d.To).queryString()]; ok {
				destination = fn
			}
			if source == destination {
				continue
			}
			found := -1
			for i, v := range g.successors[g.index[source]] {
				if v == g.index[destination] {
					found = i
					break
				}
			}
			if found == -1 {
				g.successors[g.index[source]] =
					append(g.successors[g.index[source]], g.index[destination])
				g.weigths[g.index[source]] = append(g.weigths[g.index[source]], 1)
			} else {
				g.weigths[g.index[source]][found]++
			}
		}
		if s.Err() != nil {
			return nil, s.Err()
		}
	}
	return g, nil
}

func (g *graph) copy() *graph {
	result := &graph{index: make(map[string]int), successors: make([][]int, len(g.successors))}
	for i := range g.successors {
		result.successors[i] = make([]int, len(g.successors[i]))
		copy(result.successors[i], g.successors[i])
	}
	for k, v := range g.index {
		result.index[k] = v
	}
	return result
}

func (g *graph) removeEdge(s, u string) {
	indexOfS := g.index[s]
	for i, index := range g.successors[indexOfS] {
		if index == g.index[u] {
			g.successors[indexOfS] = append(g.successors[indexOfS][0:i], g.successors[indexOfS][i+1:]...)
			break
		}
	}
}

func (g *graph) edgesCount() int {
	count := 0
	for _, ss := range g.successors {
		count += len(ss)
	}
	return count
}
