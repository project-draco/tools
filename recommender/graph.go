package main

import (
	"fmt"
	"io"

	scanner "github.com/project-draco/pkg/dependency-scanner"
	"github.com/project-draco/pkg/entity"
)

type graph struct {
	index      map[string]int
	successors [][]int
	weigths    [][]int
}

func newGraph(reassignments map[string]string, readers ...io.Reader) (*graph, error) {
	g := &graph{index: make(map[string]int)}
	for _, r := range readers {
		s := scanner.NewDependencyScanner(r)
		for s.Scan() {
			d := s.Dependency()
			if len(d.From) != 1 {
				continue
			}
			source := entity.Entity(d.From[0]).Filename()
			destination := entity.Entity(d.To).Filename()
			if filename, ok := reassignments[entity.Entity(d.From[0]).QueryString()]; ok {
				//fmt.Printf("reassigned source %v to %v\n(%v)\n", source, filename, d)
				source = filename
			}
			if filename, ok := reassignments[entity.Entity(d.To).QueryString()]; ok {
				//fmt.Printf("reassigned destination %v to %v\n(%v)\n", destination, filename, d)
				destination = filename
			}
			if source == destination {
				continue
			}
			for _, filename := range []string{source, destination} {
				if _, ok := g.index[filename]; !ok {
					g.index[filename] = len(g.successors)
					g.successors = append(g.successors, nil)
					g.weigths = append(g.weigths, nil)
				}
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

func (g *graph) diff(other *graph) (result map[string]string) {
	if len(g.index) != len(other.index) {
		panic(fmt.Errorf("comparing graphs of different sizes: %v, %v", len(g.index), len(other.index)))
	}
	for key, idx := range g.index {
		otheridx, ok := other.index[key]
		if !ok {
			panic(fmt.Errorf("key not found on other graph: %v", key))
		}
		otherlen := len(other.successors[otheridx])
		glen := len(g.successors[idx])
		diff := otherlen - glen
		if diff != 0 {
			if result == nil {
				result = make(map[string]string)
			}
			result[key] = fmt.Sprintf("%v, %v, %v, %v", glen, otherlen, diff, other.successors[otheridx])
		}
	}
	return result
}
