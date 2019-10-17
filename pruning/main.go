package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	pminsupport := flag.Int("minsupport", 1, "Minimum support count")
	pminconfidence := flag.Float64("minconfidence", 0.0, "Minimum confidence")
	pjoin := flag.Bool("join", false, "Join body and parameters files")
	pignoreparameters := flag.Bool("ignoreparameters", false, "Ignore parameters files")
	pcountfile := flag.String("countfile", "", "Path to count file")
	pstats := flag.Bool("stats", false, "Print stats and exit")
	flag.Parse()
	if pcountfile == nil || *pcountfile == "" {
		log.Fatal("Count file must be informed")
	}

	joinre1 := regexp.MustCompile("/body$")
	joinre2 := regexp.MustCompile("/parameters$")
	joinre3 := regexp.MustCompile("/package$")
	counts := map[string]int{}
	cf, err := os.Open(*pcountfile)
	if err != nil {
		log.Fatal(err)
	}
	cs := bufio.NewScanner(cf)
	for cs.Scan() {
		arr := strings.Split(cs.Text(), "\t")
		c, err := strconv.Atoi(arr[1])
		if err != nil {
			log.Fatal(err)
		}
		if *pignoreparameters && joinre2.MatchString(arr[0]) {
			continue
		}
		if *pjoin {
			if joinre3.MatchString(arr[0]) {
				continue
			}
			arr[0] = joinre1.ReplaceAllLiteralString(arr[0], "")
			arr[0] = joinre2.ReplaceAllLiteralString(arr[0], "")
		}
		counts[arr[0]] += c
	}
	if cs.Err() != nil {
		log.Fatal(cs.Err())
	}
	cf.Close()

	vertices := map[string]string{}
	edgescount := 0
	filterAndPrint := func(arr []string) {
		support, err := strconv.Atoi(arr[2])
		if err != nil {
			log.Fatal(err)
		}
		confidence := float64(support) / float64(counts[arr[0]])
		if support >= *pminsupport && confidence >= *pminconfidence {
			if *pstats {
				vertices[arr[0]] = arr[0]
				vertices[arr[1]] = arr[1]
				edgescount++
			} else {
				fmt.Printf("%v\t%v\t%v\n", arr[0], arr[1], arr[2])
			}
		}
	}
	if *pjoin {
		type edge struct{ source, destination string }
		graph := map[edge]int{}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			arr := strings.Split(scanner.Text(), "\t")
			if joinre3.MatchString(arr[0]) || joinre3.MatchString(arr[1]) {
				continue
			}
			for i := 0; i < 2; i++ {
				arr[i] = joinre1.ReplaceAllLiteralString(arr[i], "")
				arr[i] = joinre2.ReplaceAllLiteralString(arr[i], "")
			}
			c, err := strconv.Atoi(arr[2])
			if err != nil {
				log.Fatal(err)
			}
			graph[edge{arr[0], arr[1]}] += c
		}
		if scanner.Err() != nil {
			log.Fatal(scanner.Err())
		}
		for k, v := range graph {
			filterAndPrint([]string{k.source, k.destination, strconv.Itoa(v)})
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			arr := strings.Split(scanner.Text(), "\t")
			if *pignoreparameters && (joinre2.MatchString(arr[0]) || joinre2.MatchString(arr[1])) {
				continue
			}
			filterAndPrint(arr)
		}
		if scanner.Err() != nil {
			log.Fatal(scanner.Err())
		}
	}
	if *pstats {
		fmt.Printf("vertices: %v, edges: %v\n", len(vertices), edgescount)
	}
}
