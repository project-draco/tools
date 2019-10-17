package main

import (
	"fmt"
	"io"
)

type structure struct {
	files   []string
	methods map[string][]*method
	fields  map[string][]string
}

type method struct {
	Name       string
	Parameters []string
	entity     entity
}

func newStructure(reasssignments map[string]string, r io.Reader) (*structure, error) {
	result := &structure{nil, map[string][]*method{}, map[string][]string{}}
	s := newDependencyScanner(r)
	for s.Scan() {
		d := s.Dependency()
		if len(d.From) != 1 {
			continue
		}
		for _, e := range []string{d.From[0], d.To} {
			fn := entity(e).filename()
			if rfn, ok := reasssignments[entity(e).queryString()]; ok {
				fn = rfn
			}
			lf, lm := len(result.fields), len(result.methods)
			params := entity(e).parameters()
			if params == nil {
				result.fields[fn] = append(result.fields[fn], entity(e).name())
			} else {
				result.methods[fn] = append(result.methods[fn],
					&method{entity(e).name(), entity(e).parameters(), entity(e)})
			}
			if len(result.fields) > lf || len(result.methods) > lm {
				result.files = append(result.files, fn)
				lf = len(result.fields)
				lm = len(result.methods)
			}
		}
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return result, nil
}

func (s *structure) Files() []string {
	return s.files
}

func (s *structure) Methods(file string) []*method {
	return s.methods[file]
}

func (s *structure) Fields(file string) []string {
	return s.fields[file]
}

func (s *structure) PublicMethodsCount(file string, f *finder) int {
	result := 0
	for _, m := range s.Methods(file) {
		dd := f.dependenciesOf(m.entity)
		for _, d := range dd.income {
			if entity(d).filename() != m.entity.filename() {
				result++
				break
			}
		}
	}
	return result
}

func (m *method) String() string {
	return fmt.Sprintf("%v%v", m.Name, m.Parameters)
}
