package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/project-draco/naming"
)

type rule struct {
	Antecedent []string
	Consequent string
}

type ruleAsString string

type rules map[ruleAsString]int

type ruleWithCount struct {
	r rule
	c int
}

type set map[string]struct{}

var (
	out             io.Writer = os.Stdout
	executorFunc              = execCmd
	maxCommitLength           = flag.Int("max", 50, "Max commit length")
	minCommits                = flag.Int("min-commits", 0, "Min commits count")
	ignore                    = flag.String("ignore", "", "A string to ignore")
	output                    = flag.String(
		"output", "rules", "One of: rules|rules-and-commits|transactions|count")
	granularity = flag.String(
		"granularity", "fine", "Granularity of consequent. One of: fine|coarse")
	aggregationLevel   = flag.Int("aggregation-level", 1, "Aggregation level")
	freeAggregatesOnly = flag.Bool("free-aggregates-only", false,
		"Only allow free aggregates, i.e., aggregates that "+
			"there is no rule with one of its antecedents and "+
			"one of its non-antecedents in the same file")
	limit             = flag.Int("limit", math.MaxInt64, "limit of number of commits")
	minSupport        = flag.Float64("min-support", 0, "Minimum support")
	minSupportCount   = flag.Int("min-support-count", 0, "Minimum support count")
	minConfidence     = flag.Float64("min-confidence", 0, "Minimum confidence")
	cpuprofile        = flag.Bool("cpuprofile", false, "cpu profile")
	memprofile        = flag.Bool("memprofile", false, "memory profile")
	commitsMaximumAge = flag.String("max-age", "", "[Y][M][D]")
	commitsRange      = flag.String("range", "", "commits range")
	filter            = flag.String("filter", "", "regex used to filter file names")
	mu                sync.Mutex
)

func main() {
	flag.Parse()
	if *cpuprofile {
		pprof.StartCPUProfile(os.Stdout)
		defer pprof.StopCPUProfile()
	}
	if *memprofile {
		defer pprof.WriteHeapProfile(os.Stdout)
	}
	collect()
}

func collect() {
	commits := gitLog()
	if len(commits) < *minCommits {
		return
	}
	fineGrainedRules := rules{}
	coarseGrainedRules := rules{}
	conjunctiveFineGrainedRules := rules{}
	conjunctiveCoarseGrainedRules := rules{}
	commitsCountByAntecedents := map[string]int{}
	commitsByFineGrainedRule := map[ruleAsString]set{}
	commitsByCoarseGrainedRule := map[ruleAsString]set{}
	adjacencyList := map[string]set{}
	commitsCount := 0
	regexpToReplace := regexp.MustCompile(`\/body$|\/parameters$`)
	regexpToIgnore := regexp.MustCompile(*ignore)
	regexpToFilter := regexp.MustCompile(*filter)
	if *ignore == "" {
		regexpToIgnore = nil
	}
	if *filter == "" {
		regexpToFilter = nil
	}
	for i, c := range commits {
		if i >= *limit {
			break
		}
		modified, deleted := gitDiffTree(c, regexpToReplace, regexpToIgnore, regexpToFilter)
		if len(modified) <= *maxCommitLength && len(modified) > 1 {
			commitsCount++
			for m := range modified {
				commitsCountByAntecedents[m]++
			}
			if *output == "rules" || *output == "rules-and-commits" {
				fgr, cgr := addRules(
					fineGrainedRules,
					coarseGrainedRules,
					adjacencyList,
					modified,
				)
				if *aggregationLevel > 1 {
					ch := make(chan ruleWithCount)
					go aggregate(ch, fgr, rules{}, rules{},
						map[string]int{}, map[string]set{}, 1, 0, 0, 0)
					for rc := range ch {
						conjunctiveFineGrainedRules[rc.r.asString()]++
					}
					ch = make(chan ruleWithCount)
					go aggregate(ch, cgr, rules{}, rules{},
						map[string]int{}, map[string]set{}, 1, 0, 0, 0)
					for rc := range ch {
						conjunctiveCoarseGrainedRules[rc.r.asString()]++
					}
				}
				for r := range fgr {
					commitsByFineGrainedRule[r] = commitsByFineGrainedRule[r].add(c)
				}
				for r := range cgr {
					commitsByCoarseGrainedRule[r] = commitsByCoarseGrainedRule[r].add(c)
				}
			}
		}
		for _, d := range deleted {
			delete(commitsCountByAntecedents, d)
		}
		deleteRules(fineGrainedRules, deleted)
		deleteRules(coarseGrainedRules, deleted)
	}
	var (
		rr, cr        rules
		commitsByRule map[ruleAsString]set
	)
	if *granularity == "fine" {
		rr = fineGrainedRules
		cr = conjunctiveFineGrainedRules
		commitsByRule = commitsByFineGrainedRule
	} else {
		rr = coarseGrainedRules
		cr = conjunctiveCoarseGrainedRules
		commitsByRule = commitsByCoarseGrainedRule
	}
	var ch chan ruleWithCount
	if *aggregationLevel > 1 {
		ch = make(chan ruleWithCount)
		go aggregate(ch, rr, cr, fineGrainedRules, commitsCountByAntecedents,
			adjacencyList, commitsCount,
			*minSupport, *minConfidence, *minSupportCount)
	}
	if !*cpuprofile && !*memprofile {
		printOutput(
			rr,
			ch,
			commitsCountByAntecedents,
			commitsCount,
			commitsByRule,
		)
	} else if ch != nil {
		// drain the channel
		for range ch {
		}
	}
}

func gitLog() (commits []string) {
	since := ""
	if *commitsMaximumAge != "" {
		years, months, days := 0, 0, 0
		re := regexp.MustCompile(`((\d)+Y)?((\d)+M)?((\d)+D)?`)
		submatch := re.FindStringSubmatch(*commitsMaximumAge)
		pointer := []*int{&years, &months, &days}
		for i := 1; i < len(submatch); i += 2 {
			if submatch[i+1] != "" {
				var err error
				*pointer[(i-1)/2], err = strconv.Atoi(submatch[i+1])
				if err != nil {
					panic(err)
				}
			}
		}
		lastCommitDateAsString := string(executorFunc(
			[]string{"git", "log", "-n 1", "--pretty=format:%aI"},
		))
		iso8601 := "2006-01-02T15:04:05-07:00"
		lastCommitDate, err := time.Parse(iso8601, lastCommitDateAsString)
		if err != nil {
			panic(err)
		}
		lastCommitDate = lastCommitDate.AddDate(-years, -months, -days)
		since = fmt.Sprintf("--since=%v", lastCommitDate.Format(iso8601))
	}
	args := []string{
		"git", "log", "--date=iso", "--reverse", "--pretty=format:%H",
	}
	if since != "" {
		args = append(args, since)
	}
	if *commitsRange != "" {
		args = append(args, *commitsRange)
	}
	for line := range scanOutput(executorFunc(args)) {
		commits = append(commits, line)
	}
	return
}

func gitDiffTree(
	commit string,
	regexpToReplace, regexpToIgnore, regexpToFilter *regexp.Regexp,
) (
	modified set,
	deleted []string,
) {
	args := []string{
		"git", "diff-tree", "--no-commit-id", "--name-status", "-r", commit,
	}
	for line := range scanOutput(executorFunc(args)) {
		if len(line) < 2 ||
			strings.HasSuffix(line, "/package") ||
			strings.HasSuffix(line, "/extend") {
			continue
		}
		entity := regexpToReplace.ReplaceAllString(line[2:], "")
		if (regexpToIgnore == nil || !regexpToIgnore.MatchString(entity)) &&
			(regexpToFilter == nil || regexpToFilter.MatchString(entity)) {
			if strings.HasPrefix(line, "D	") {
				deleted = append(deleted, entity)
			} else {
				modified = modified.add(entity)
			}
		}
	}
	return
}

func addRules(
	fineGrainedRules rules,
	coarseGrainedRules rules,
	adjacencyList map[string]set,
	modified set,
) (rr, crr rules) {
	rr = rules{}
	for m1 := range modified {
		for m2 := range modified {
			if m1 != m2 {
				r := rule{[]string{m1}, m2}
				fineGrainedRules.add(r)
				rr.add(r)
				adjacencyList[m1] = adjacencyList[m1].add(m2)
			}
		}
	}
	crr = increaseGranularity(rr)
	for k := range crr {
		coarseGrainedRules[k]++
	}
	return rr, crr
}

func deleteRules(rr rules, deleted []string) {
	for _, d := range deleted {
		for key := range rr {
			rule := key.asRule()
			if d == rule.Consequent {
				delete(rr, key)
			} else {
				for _, s := range rule.Antecedent {
					if d == s {
						delete(rr, key)
						break
					}
				}
			}
		}
	}
}

func increaseGranularity(rr rules) rules {
	result := rules{}
	for key, supportCount := range rr {
		r := key.asRule()
		file := naming.FileFromHR(r.Consequent)
		if file != "" {
			result[rule{r.Antecedent, file}.asString()] += supportCount
		}
	}
	return result
}

func aggregate(
	ch chan ruleWithCount,
	rr rules,
	conjunctiveRules rules,
	fineGrainedRules rules,
	commitsCountByAntecedents map[string]int,
	adjacencyList map[string]set,
	commitsCount int,
	minSupport float64,
	minConfidence float64,
	minSupportCount int,
) {
	var keys []ruleAsString
	for k := range rr {
		keys = append(keys, k)
	}
	for i, rs1 := range keys {
		c1 := rr[rs1]
		r1 := rs1.asRule()
	next:
		for j := i + 1; j < len(keys); j++ {
			rs2 := keys[j]
			c2 := rr[rs2]
			if rs1 == rs2 {
				continue
			}
			r2 := rs2.asRule()
			if r1.Consequent != r2.Consequent {
				continue
			}
			antecedents := append([]string{}, r1.Antecedent...)
			antecedents = append(antecedents, r2.Antecedent...)
			sort.Strings(antecedents)
			antecedentsAtSameFile := true
			sameFileAsConsequent := true
			fileOfAntecedents := naming.FileFromHR(antecedents[0])
			for _, a := range antecedents {
				if naming.FileFromHR(a) != fileOfAntecedents {
					antecedentsAtSameFile = false
					break
				}
				if naming.FileFromHR(a) != naming.FileFromHR(r1.Consequent) {
					sameFileAsConsequent = false
				}
			}
			if !antecedentsAtSameFile || sameFileAsConsequent {
				continue
			}
			commonCommitsCount := fineGrainedRules[rule{
				Antecedent: antecedents[1:],
				Consequent: antecedents[0],
			}.asString()]
			antecedentsAsString := strings.Join(antecedents, "\t")
			mu.Lock()
			antecedentsCount :=
				commitsCountByAntecedents[strings.Join(r1.Antecedent, "\t")] +
					commitsCountByAntecedents[strings.Join(r2.Antecedent, "\t")] -
					commonCommitsCount
			commitsCountByAntecedents[antecedentsAsString] = antecedentsCount
			mu.Unlock()
			r := rule{antecedents, r1.Consequent}
			rs := r.asString()
			supportCount := c1 + c2 - conjunctiveRules[rs]
			support := float64(supportCount) / float64(commitsCount)
			confidence := float64(supportCount) / float64(antecedentsCount)
			if support < minSupport || supportCount < minSupportCount ||
				confidence < minConfidence {
				continue
			}
			if *freeAggregatesOnly {
				for _, a1 := range antecedents {
					for adj := range adjacencyList[a1] {
						found := false
						for _, a2 := range antecedents {
							if adj == a2 {
								found = true
								break
							}
						}
						if naming.FileFromHR(a1) == naming.FileFromHR(adj) && !found {
							continue next
						}
					}
				}
			}
			ch <- ruleWithCount{r, supportCount}
		}
	}
	close(ch)
}

func printOutput(
	rules rules,
	aggrch chan ruleWithCount,
	commitsCountByAntecedents map[string]int,
	commitsCount int,
	commitsByRule map[ruleAsString]set,
) {
	switch *output {
	case "count":
		for k, v := range commitsCountByAntecedents {
			fmt.Fprintf(out, "%v\t%v\n", k, v)
		}
	case "rules", "rules-and-commits":
		for key, supportCount := range rules {
			r := key.asRule()
			printRule(
				ruleWithCount{r, supportCount},
				commitsCountByAntecedents,
				commitsCount,
				commitsByRule[key],
			)
		}
		if aggrch != nil {
			for r := range aggrch {
				printRule(
					r,
					commitsCountByAntecedents,
					commitsCount,
					commitsByRule[r.r.asString()],
				)
			}
		}
	}
}

func printRule(
	rc ruleWithCount,
	commitsCountByAntecedents map[string]int,
	commitsCount int,
	commits set,
) {
	mu.Lock()
	defer mu.Unlock()
	antecedents := strings.Join(rc.r.Antecedent, "\t")
	support := float64(rc.c) / float64(commitsCount)
	confidence :=
		float64(rc.c) / float64(commitsCountByAntecedents[antecedents])
	if support < *minSupport || rc.c < *minSupportCount ||
		confidence < *minConfidence {
		return
	}
	var commitsAsString string
	if *output == "rules-and-commits" {
		commitsAsString = fmt.Sprintf("\t%v", commits)
	}
	fmt.Fprintf(
		out,
		"%v\t%v\t%v\t%.4f\t%v\t%v%v\n",
		antecedents,
		rc.r.Consequent,
		rc.c,
		confidence,
		commitsCountByAntecedents[antecedents],
		commitsCount,
		commitsAsString,
	)
}

func scanOutput(b []byte) (ch chan string) {
	ch = make(chan string)
	go func() {
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
		if scanner.Err() != nil {
			panic(scanner.Err())
		}
	}()
	return ch
}

func execCmd(args []string) []byte {
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		panic(err)
	}
	return out
}

func (r rule) asString() ruleAsString {
	sort.Strings(r.Antecedent)
	return ruleAsString(
		fmt.Sprintf("%v\t%v", strings.Join(r.Antecedent, "\t"), r.Consequent))
}

func (rr rules) add(r rule) {
	rr[r.asString()]++
}

func (rs ruleAsString) asRule() rule {
	ss := strings.Split(string(rs), "\t")
	return rule{Antecedent: ss[0 : len(ss)-1], Consequent: ss[len(ss)-1]}
}

func (s set) add(str ...string) set {
	if s == nil {
		s = set{}
	}
	for _, each := range str {
		s[each] = struct{}{}
	}
	return s
}

func (s set) String() string {
	var str []string
	for each := range s {
		str = append(str, each)
	}
	return strings.Join(str, ",")
}
