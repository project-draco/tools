package main

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type dependencyScanner struct {
	scanner *bufio.Scanner
}

func newDependencyScanner(r io.Reader) *dependencyScanner {
	return &dependencyScanner{bufio.NewScanner(r)}
}

func (ds *dependencyScanner) Scan() bool {
	for ds.scanner.Scan() {
		if strings.TrimSpace(ds.scanner.Text()) != "" {
			return true
		}
	}
	return false
}

func (ds *dependencyScanner) Err() error {
	return ds.scanner.Err()
}

func (ds *dependencyScanner) Dependency() struct {
	From         []string
	To           string
	SupportCount int
	Confidence   float64
	CommitsCount int
} {
	arr := strings.Split(strings.TrimSpace(ds.scanner.Text()), "\t")
	if len(arr) < 2 {
		arr = strings.Split(ds.scanner.Text(), " ")
	}
	var i int
	for i = len(arr) - 1; i > -1; i-- {
		_, err := strconv.ParseFloat(arr[i], 32)
		if err != nil {
			break
		}
	}
	entities := arr[0 : i+1]
	var numbers []string
	if i < len(arr)-1 {
		numbers = arr[i+1:]
	}
	supportCount := 0
	if len(numbers) > 0 {
		supportCount, _ = strconv.Atoi(numbers[0])
	}
	confidence := 0.0
	if len(numbers) > 1 {
		confidence, _ = strconv.ParseFloat(numbers[1], 32)
	}
	commitsCount := 0
	if len(numbers) > 3 {
		commitsCount, _ = strconv.Atoi(numbers[3])
	}
	return struct {
		From         []string
		To           string
		SupportCount int
		Confidence   float64
		CommitsCount int
	}{
		entities[:len(entities)-1], entities[len(entities)-1],
		supportCount,
		confidence,
		commitsCount,
	}
}
