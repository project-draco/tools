package main

import (
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/project-draco/naming"
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

type entity string

var classNameRegexp *regexp.Regexp

var fileNameRegexp *regexp.Regexp

var pathRegexp *regexp.Regexp

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
			qs := entity(e).queryString()
			fn := entity(e).filename()
			if _, ok := result.dependenciesByEntity[qs]; !ok {
				result.dependenciesByEntity[qs] = &dependencies{}
			}
			if _, ok := result.dependenciesByFile[fn]; !ok {
				result.dependenciesByFile[fn] = &dependencies{}
			}
		}
		dd := result.dependenciesByEntity[entity(d.To).queryString()]
		dd.income = append(dd.income, d.From[0])
		dd = result.dependenciesByEntity[entity(d.From[0]).queryString()]
		dd.outcome = append(dd.outcome, d.To)
		dd = result.dependenciesByFile[entity(d.To).filename()]
		dd.income = append(dd.income, d.From[0])
		dd = result.dependenciesByFile[entity(d.From[0]).filename()]
		dd.outcome = append(dd.outcome, d.To)
		fromTo := [2]string{entity(d.From[0]).filename(), entity(d.To).filename()}
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

func (f *finder) dependenciesOf(e entity) *dependencies {
	q := e.queryString()
	if q == "" {
		return nil
	}
	return f.dependenciesByEntity[q]
}

func (f *finder) find(e entity) bool {
	if f.dependenciesOf(e) != nil {
		return true
	}
	return f.onErrors(e)
}

func (f *finder) onErrors(e entity) bool {
	q := e.queryString()
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
		if querystringFilename(k) != filename1 && querystringFilename(k) != filename2 {
			continue
		}
		for _, d := range append(v.income, v.outcome...) {
			if entity(d).filename() != filename1 && entity(d).filename() != filename2 {
				continue
			}
			if querystringFilename(k) != entity(d).filename() {
				result = append(result, []string{k, entity(d).queryString()})
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

func (e entity) queryString() string {
	q := strings.TrimSpace(string(e))
	q = strings.Replace(q, "/body", "", -1)
	q = strings.Replace(q, "/parameters", "", -1)
	idx := strings.Index(q, ".java/")
	if idx == -1 {
		return ""
	}
	underscoreidx := strings.LastIndex(q[:idx], "_")
	if underscoreidx == -1 {
		underscoreidx = 0
	}
	q = q[underscoreidx:]
	arr := strings.Split(q, "/[CN]/")
	if len(arr) > 2 {
		q = arr[0] + "/[CN]/" + arr[len(arr)-1]
	}
	q = naming.RemoveGenerics(q)
	parenthesisidx := strings.Index(q, "(")
	if parenthesisidx != -1 {
		arr := strings.Split(q[parenthesisidx+1:len(q)-1], ",")
		for i := range arr {
			arr[i] = lastSubstring(arr[i], ".")
		}
		q = q[:parenthesisidx] + "(" + strings.Join(arr, ",") + ")"
	}
	return q
}

func (e entity) classname() string {
	if classNameRegexp == nil {
		var err error
		classNameRegexp, err = regexp.Compile(`.+\.java/\[CN\]/([^\[]+)/`)
		if err != nil {
			panic(err)
		}
	}
	return classNameRegexp.FindAllStringSubmatch(e.queryString(), -1)[0][1]
}

func (e entity) filename() string {
	return querystringFilename(e.queryString())
}

func querystringFilename(qs string) string {
	if fileNameRegexp == nil {
		fileNameRegexp = regexp.MustCompile(`\_([^\.]+)\.java/\[CN\]/`)
	}
	submatch := fileNameRegexp.FindAllStringSubmatch(qs, -1)
	if len(submatch) == 0 || len(submatch[0]) < 2 {
		return ""
	}
	return submatch[0][1]
}

func (e entity) path() string {
	if pathRegexp == nil {
		var err error
		pathRegexp, err = regexp.Compile(`([^\.]+)\.java/\[CN\]/`)
		if err != nil {
			panic(err)
		}
	}
	submatch := pathRegexp.FindAllStringSubmatch(string(e), -1)
	if len(submatch) == 0 || len(submatch[0]) < 2 {
		return ""
	}
	return submatch[0][1]
}

func (e entity) name() string {
	qs := e.queryString()
	qs = qs[strings.LastIndex(qs, "/")+1:]
	parenthesisidx := strings.Index(qs, "(")
	if parenthesisidx == -1 {
		return qs
	}
	return qs[:parenthesisidx]
}

func (e entity) parameters() []string {
	qs := e.queryString()
	parenthesisidx := strings.Index(qs, "(")
	if parenthesisidx == -1 {
		return nil
	}
	qs = qs[parenthesisidx+1 : len(qs)-1]
	if qs == "" {
		return []string{}
	}
	return strings.Split(qs, ",")
}

func lastSubstring(s, sep string) string {
	arr := strings.Split(strings.TrimSpace(s), sep)
	return arr[len(arr)-1]
}
