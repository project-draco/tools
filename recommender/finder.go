package main

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/project-draco/pkg/entity"
)

type finder struct {
	dependenciesByEntity map[string]*dependencies
	dependenciesByFile   map[string]*dependencies
	fileByFile           map[[2]string]struct{}
	errors               string
}

type dependencies struct {
	income, outcome []string
}

func newFinder(dr, er io.Reader) (*finder, error) {
	result := &finder{
		make(map[string]*dependencies),
		make(map[string]*dependencies),
		make(map[[2]string]struct{}),
		"",
	}
	s := newDependencyScanner(dr)
	for s.Scan() {
		d := s.Dependency()
		if len(d.From) != 1 {
			continue
		}
		for _, e := range []string{d.From[0], d.To} {
			qs := entity.Entity(e).QueryString()
			fn := entity.Entity(e).Filename()
			if _, ok := result.dependenciesByEntity[qs]; !ok {
				result.dependenciesByEntity[qs] = &dependencies{}
			}
			if _, ok := result.dependenciesByFile[fn]; !ok {
				result.dependenciesByFile[fn] = &dependencies{}
			}
		}
		dd := result.dependenciesByEntity[entity.Entity(d.To).QueryString()]
		dd.income = append(dd.income, d.From[0])
		dd = result.dependenciesByEntity[entity.Entity(d.From[0]).QueryString()]
		dd.outcome = append(dd.outcome, d.To)
		dd = result.dependenciesByFile[entity.Entity(d.To).Filename()]
		dd.income = append(dd.income, d.From[0])
		dd = result.dependenciesByFile[entity.Entity(d.From[0]).Filename()]
		dd.outcome = append(dd.outcome, d.To)
		fromTo := [2]string{
			entity.Entity(d.From[0]).Filename(),
			entity.Entity(d.To).Filename(),
		}
		result.fileByFile[fromTo] = struct{}{}
		result.fileByFile[[2]string{fromTo[1], fromTo[0]}] = struct{}{}
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	if er != nil {
		buf, err := ioutil.ReadAll(er)
		if err != nil {
			return nil, err
		}
		result.errors = string(buf)
	}
	return result, nil
}

func (f *finder) dependenciesOf(e entity.Entity) *dependencies {
	q := e.QueryString()
	if q == "" {
		return nil
	}
	return f.dependenciesByEntity[q]
}

func (f *finder) find(e entity.Entity) bool {
	if f.dependenciesOf(e) != nil {
		return true
	}
	return f.onErrors(e)
}

func (f *finder) onErrors(e entity.Entity) bool {
	q := e.QueryString()
	if q == "" {
		return false
	}
	idx := strings.Index(q, "(")
	if idx > -1 {
		q = q[strings.LastIndex(q[:idx], "/")+1 : idx]
	}
	return strings.Contains(f.errors, q)
}

func (f *finder) hasDependenciesBetweenFiles(filename1, filename2 string) bool {
	_, ok1 := f.fileByFile[[2]string{filename1, filename2}]
	_, ok2 := f.fileByFile[[2]string{filename2, filename1}]
	return ok1 || ok2
}

func (f *finder) dependenciesBetweenFiles(filename1, filename2 string) [][]string {
	result := make([][]string, 0)
	for k, v := range f.dependenciesByEntity {
		if entity.QuerystringFilename(k) != filename1 &&
			entity.QuerystringFilename(k) != filename2 {
			continue
		}
		for _, d := range append(v.income, v.outcome...) {
			if entity.Entity(d).Filename() != filename1 &&
				entity.Entity(d).Filename() != filename2 {
				continue
			}
			if entity.QuerystringFilename(k) != entity.Entity(d).Filename() {
				result = append(
					result,
					[]string{k, entity.Entity(d).QueryString()},
				)
			}
		}
	}
	return result
}

func (f *finder) entitiesCount() int {
	return len(f.dependenciesByEntity)
}

func (f *finder) dependenciesCount() int {
	result := 0
	for _, dd := range f.dependenciesByEntity {
		result += len(dd.outcome)
	}
	return result
}
