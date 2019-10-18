package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/project-draco/naming"
)

// MoveMethod represents a move method refactoring
type MoveMethod struct {
	From, To string
}

// Parse extracts the name and the destination class of the moved method
// from the detail string
func (mm *MoveMethod) Parse(detail string) {
	methods := strings.Split(detail, " to ")
	methods[0] = strings.TrimPrefix(methods[0], "Move Method ")
	methods[0] = strings.TrimSpace(methods[0])
	mm.From, mm.To = mm.qualifiedName(methods[0]), mm.className(methods[1])
}

func (mm *MoveMethod) qualifiedName(method string) string {
	segments := strings.Split(method, " from class ")
	return segments[1] + "." + mm.methodSignature(segments[0])
}

func (mm *MoveMethod) className(method string) string {
	segments := strings.Split(method, " from class ")
	return segments[1]
}

func (mm *MoveMethod) methodSignature(method string) string {
	segments := strings.Split(method, " : ")
	segments[0] = naming.RemoveGenerics(segments[0])
	parensidx := strings.Index(segments[0], "(")
	params := strings.Split(
		segments[0][parensidx+1:len(segments[0])-1],
		",",
	)
	for i, p := range params {
		p = strings.TrimSpace(p)
		params[i] = p[strings.Index(p, " ")+1:]
	}
	methodName := segments[0][strings.Index(segments[0], " ")+1 : parensidx]
	return methodName + "(" + strings.Join(params, ",") + ")"
}

func (mm *MoveMethod) String() string {
	return fmt.Sprint(*mm)
}

// Unknown represents a refactoring of unknown type
type Unknown struct{ detail string }

// Parse copies the detail string into the Unknow refactoring
func (u *Unknown) Parse(detail string) {
	u.detail = detail
}

func (u *Unknown) String() string {
	return fmt.Sprint(*u)
}

type parser interface{ Parse(string) }

var (
	factoryByType = map[string]func() parser{
		"Move Method": func() parser { return &MoveMethod{} },
		"Unknown":     func() parser { return &Unknown{} },
	}
)

// Parse returns the refactorings contained in reader r
func Parse(r io.Reader) (refactorings []interface{}, err error) {
	csvr := csv.NewReader(r)
	csvr.Comma = ';'
	_, err = csvr.Read() // discard header
	if err != nil {
		return nil, fmt.Errorf("could not read header: %v", err)
	}
	for {
		record, err := csvr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not read record: %v", err)
		}
		var ref parser
		if newRefactoring, ok := factoryByType[record[1]]; ok {
			ref = newRefactoring()
		} else {
			ref = factoryByType["Unknown"]()
		}
		ref.Parse(record[2])
		refactorings = append(refactorings, ref)
	}
	return refactorings, nil
}
