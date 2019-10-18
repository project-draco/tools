package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/project-draco/pkg/entity"
)

type fieldTypes struct {
	types map[string]string
}

func newFieldTypes(r io.Reader) (*fieldTypes, error) {
	result := &fieldTypes{map[string]string{}}
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
		result.types[entity.Entity(arr[0]).QueryString()] =
			entity.Entity(arr[1]).Filename()
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return result, nil
}

func (ft *fieldTypes) TypeOf(s string) string {
	if ft == nil || ft.types == nil {
		return ""
	}
	return ft.types[entity.Entity(s).QueryString()]
}
