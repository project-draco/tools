package main

import (
	"math"
	"sort"
	"strings"
	"testing"
)

type lines = []string
type commits = map[string][]string

func init() {
	*maxCommitLength = 50
	*minCommits = 0
	*ignore = ""
	*output = "rules"
	*limit = math.MaxInt64
	*minSupport = 0
	*minConfidence = 0
}
func TestCollect(t *testing.T) {
	tests := []struct {
		name               string
		commits            commits
		want               lines
		granularity        string
		aggregationLevel   int
		freeAggregatesOnly bool
	}{
		{
			"one commit, no mapping between entities and files",
			commits{"1": lines{"M m1", "M m2"}},
			lines{
				"m1\tm2\t1\t1.0000\t1\t1",
				"m2\tm1\t1\t1.0000\t1\t1",
			},
			"fine",
			0,
			false,
		},
		{
			"one commit, with mapping between entities and files",
			commits{"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"}},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t1\t1.0000\t1\t1",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t1",
			},
			"coarse",
			0,
			false,
		},
		{
			"two commits, with mapping between entities and files",
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f1/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t1\t0.5000\t2\t2",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t2",
				"f1/[CN]/m1\tf1/[CN]/\t1\t0.5000\t2\t2",
				"f1/[CN]/m3\tf1/[CN]/\t1\t1.0000\t1\t2",
			},
			"coarse",
			0,
			false,
		},
		{
			"two commits, with different sufixes in the same method",
			commits{
				"1": lines{"M f1/[CN]/m1/body", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1/parameters", "M f1/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t1\t0.5000\t2\t2",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t2",
				"f1/[CN]/m1\tf1/[CN]/\t1\t0.5000\t2\t2",
				"f1/[CN]/m3\tf1/[CN]/\t1\t1.0000\t1\t2",
			},
			"coarse",
			0,
			false,
		},
		{
			`three commits, with mapping between entities and files
			and with more than one commit envolving the same entity and file`,
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f1/[CN]/m3"},
				"3": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t2\t0.6667\t3\t3",
				"f1/[CN]/m1\tf1/[CN]/\t1\t0.3333\t3\t3",
				"f2/[CN]/m2\tf1/[CN]/\t2\t1.0000\t2\t3",
				"f1/[CN]/m3\tf1/[CN]/\t1\t1.0000\t1\t3",
			},
			"coarse",
			0,
			false,
		},
		{
			`three commits, with mapping between entities and files
			and commits with different number of files modified`,
			commits{
				"1": lines{"M f1/[CN]/m1"},
				"2": lines{"M f1/[CN]/m1", "M f1/[CN]/m3"},
				"3": lines{"M f1/[CN]/m1", "M f2/[CN]/m2", "M f1/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf1/[CN]/\t2\t1.0000\t2\t2",
				"f1/[CN]/m1\tf2/[CN]/\t1\t0.5000\t2\t2",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t2",
				"f1/[CN]/m3\tf1/[CN]/\t2\t1.0000\t2\t2",
				"f1/[CN]/m3\tf2/[CN]/\t1\t0.5000\t2\t2",
			},
			"coarse",
			0,
			false,
		},
		{
			`three commits, with mapping between entities and files
			and deletion`,
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f1/[CN]/m3"},
				"3": lines{"D\tf1/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t1\t0.5000\t2\t2",
				"f1/[CN]/m1\tf1/[CN]/\t1\t0.5000\t2\t2",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t2",
			},
			"coarse",
			0,
			false,
		},
		{
			`two commits, with mapping between entities and files,
			and aggregation level 2`,
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f2/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t2\t1.0000\t2\t2",
				"f2/[CN]/m2\tf1/[CN]/\t1\t1.0000\t1\t2",
				"f2/[CN]/m3\tf1/[CN]/\t1\t1.0000\t1\t2",
				"f2/[CN]/m2\tf2/[CN]/m3\tf1/[CN]/\t2\t1.0000\t2\t2",
			},
			"coarse",
			2,
			false,
		},
		{
			`two commits, with mapping between entities and files,
			and aggregation level 2, and overlapping of commits`,
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f2/[CN]/m3"},
				"3": lines{"M f1/[CN]/m1", "M f2/[CN]/m2", "M f2/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t3\t1.0000\t3\t3",
				"f2/[CN]/m2\tf1/[CN]/\t2\t1.0000\t2\t3",
				"f2/[CN]/m3\tf1/[CN]/\t2\t1.0000\t2\t3",
				"f2/[CN]/m2\tf2/[CN]/\t1\t0.5000\t2\t3",
				"f2/[CN]/m3\tf2/[CN]/\t1\t0.5000\t2\t3",
				"f2/[CN]/m2\tf2/[CN]/m3\tf1/[CN]/\t3\t1.0000\t3\t3",
			},
			"coarse",
			2,
			false,
		},
		{
			`two commits, with mapping between entities and files,
			and aggregation level 2, and overlapping of commits,
			and no free aggregates`,
			commits{
				"1": lines{"M f1/[CN]/m1", "M f2/[CN]/m2"},
				"2": lines{"M f1/[CN]/m1", "M f2/[CN]/m3"},
				"3": lines{"M f2/[CN]/m2", "M f2/[CN]/m4"},
				"4": lines{"M f1/[CN]/m1", "M f2/[CN]/m2", "M f2/[CN]/m3"},
			},
			lines{
				"f1/[CN]/m1\tf2/[CN]/\t3\t1.0000\t3\t4",
				"f2/[CN]/m2\tf1/[CN]/\t2\t0.6667\t3\t4",
				"f2/[CN]/m3\tf1/[CN]/\t2\t1.0000\t2\t4",
				"f2/[CN]/m2\tf2/[CN]/\t2\t0.6667\t3\t4",
				"f2/[CN]/m3\tf2/[CN]/\t1\t0.5000\t2\t4",
				"f2/[CN]/m4\tf2/[CN]/\t1\t1.0000\t1\t4",
			},
			"coarse",
			2,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b strings.Builder
			out = &b
			executorFunc = executor(test.commits)
			*granularity = test.granularity
			*aggregationLevel = test.aggregationLevel
			*freeAggregatesOnly = test.freeAggregatesOnly
			collect()
			got := strings.Split(strings.TrimSpace(b.String()), "\n")
			sort.Strings(got)
			sort.Strings(test.want)
			gotString := strings.Join(got, "\n")
			wantString := strings.Join(test.want, "\n")
			if gotString != wantString {
				t.Errorf("Got\n%v\nwant\n%v", gotString, wantString)
			}
		})
	}
}

func BenchmarkCollect(b *testing.B) {
	var sb strings.Builder
	out = &sb
	cc := commits{
		"1": lines{"M f1/[CN]/m1"},
		"2": lines{"M f1/[CN]/m1", "M f1/[CN]/m3"},
		"3": lines{"M f1/[CN]/m1", "M f2/[CN]/m2", "M f1/[CN]/m3"},
	}
	executorFunc = executor(cc)
	*output = "rules"
	*granularity = "fine"
	*aggregationLevel = 0
	for i := 0; i < b.N; i++ {
		collect()
	}
}

func executor(commits commits) func(args []string) []byte {
	return func(args []string) []byte {
		switch args[1] {
		case "log":
			cc := []string{}
			for commit := range commits {
				cc = append(cc, commit)
			}
			sort.Strings(cc)
			return []byte(strings.Join(cc, "\n"))
		case "diff-tree":
			commit := args[len(args)-1] // last argument is the commit id
			return []byte(strings.Join(commits[commit], "\n"))
		}
		return nil
	}
}
