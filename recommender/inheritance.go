package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/project-draco/pkg/entity"
)

type inheritance struct {
	superclasses [][]int
	index        map[string]int
	rindex       map[int]string
	sindex       map[string]bool
}

func newInheritance(r io.Reader) (*inheritance, error) {
	result := &inheritance{nil, make(map[string]int), make(map[int]string), make(map[string]bool)}
	s := bufio.NewScanner(r)
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
		arr := strings.Split(strings.TrimSpace(s.Text()), "\t")
		if len(arr) < 2 {
			arr = strings.Split(strings.TrimSpace(s.Text()), " ")
		}
		if len(arr) < 2 {
			return nil, fmt.Errorf("Invalid argument: malformed line: %v", s.Text())
		}
		for i := range arr {
			if _, ok := result.index[arr[i]]; !ok {
				result.index[arr[i]] = len(result.index)
				result.rindex[len(result.rindex)] = arr[i]
				result.superclasses = append(result.superclasses, nil)
			}
		}
		subidx := result.index[arr[0]]
		superidx := result.index[arr[1]]
		result.superclasses[subidx] = append(result.superclasses[subidx], superidx)
		result.sindex[entity.Entity(arr[1]).Filename()] = true
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return result, nil
}

func (inh *inheritance) OutboundList() [][]int {
	if inh == nil || inh.superclasses == nil {
		return nil
	}
	var result [][]int
	for i, sup := range inh.superclasses {
		if len(sup) == 0 {
			continue
		}
		result = append(result, nil)
		result[len(result)-1] = append(result[len(result)-1], i)
		for _, c := range sup {
			result[len(result)-1] = append(result[len(result)-1], c)
		}
	}
	return result
}

func (inh *inheritance) InboundList() [][]int {
	var result [][]int
	outbound := inh.OutboundList()
	for i, sub := range outbound {
		if len(sub) == 0 {
			continue
		}
		var inbound []int
	next:
		for j := 1; j < len(sub); j++ {
			parent := sub[j]
			for _, cc := range result {
				if parent == cc[0] {
					continue next
				}
			}
			inbound = append(inbound, parent)
			inbound = append(inbound, sub[0])
			for k, sub2 := range outbound {
				if k == i {
					continue
				}
				for l := 1; l < len(sub2); l++ {
					if sub2[l] == parent {
						inbound = append(inbound, sub2[0])
						break
					}
				}
			}
		}
		if len(inbound) > 0 {
			result = append(result, inbound)
		}
	}
	return result
}

func (inh *inheritance) File(i int) string {
	return entity.Entity(inh.rindex[i]).Filename()
}

func (inh *inheritance) IsSuperclass(file string) bool {
	if inh == nil || inh.sindex == nil {
		return false
	}
	return inh.sindex[file]
}
