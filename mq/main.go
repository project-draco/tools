package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/awalterschulze/gographviz"
	scanner "github.com/project-draco/pkg/dependency-scanner"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	check(err)
	gast, err := gographviz.Parse(data)
	check(err)
	g := gographviz.NewGraph()
	err = gographviz.Analyse(gast, g)
	check(err)
	clusterOfNode := map[string]string{}
	for cluster, nodes := range g.Relations.ParentToChildren {
		if !strings.HasPrefix(cluster, "cluster") {
			continue
		}
		for node := range nodes {
			clusterOfNode[strings.ReplaceAll(node, `"`, "")] = cluster
		}
	}
	f, err := os.Open(os.Args[1])
	check(err)
	α := make(map[string]float64)
	β := make(map[string]float64)
	scanner := scanner.NewDependencyScanner(f)
	for scanner.Scan() {
		dep := scanner.Dependency()
		i := clusterOfNode[dep.From[0]]
		j := clusterOfNode[dep.To]
		if i == "" || j == "" {
			continue
		}
		fmt.Println(i, j)
		if i == j {
			α[i] += float64(dep.SupportCount)
		} else {
			β[i] += float64(dep.SupportCount)
			β[j] += float64(dep.SupportCount)
		}
	}
	check(scanner.Err())
	mq := 0.0
	for cluster := range g.Relations.ParentToChildren {
		if !strings.HasPrefix(cluster, "cluster") {
			continue
		}
		if α[cluster] == 0 && β[cluster] == 0 {
			continue
		}
		mq += 2 * α[cluster] / (2*α[cluster] + β[cluster])
	}
	fmt.Println(mq)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
