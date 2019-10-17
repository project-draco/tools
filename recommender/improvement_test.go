package main

import (
	"math"
	"strings"
	"testing"
)

func TestImprovements(t *testing.T) {
	for testcase, d := range []struct {
		mdg           string
		reassignments []map[string]string
		inheritance   string
		improvement   []float64
	}{
		{
			`
			p_C1.java/[CN]/C1/[MT]/m1(int) p_C2.java/[CN]/C2/[MT]/m2(String)
			p_C3.java/[CN]/C3/[MT]/m3(int) p_C4.java/[CN]/C4/[MT]/m4(int)
			p_C.java/[CN]/C/[MT]/m3(int) p_C.java/[CN]/C/[MT]/m4(int)
			`,
			[]map[string]string{{entity("p_C1.java/[CN]/C1/[MT]/m1(int)").queryString(): "C2"}},
			`
			p_C1.java/[CN]/	p_C.java/[CN]/
			p_C2.java/[CN]/	p_C.java/[CN]/
			p_C3.java/[CN]/	p_C.java/[CN]/
			p_C4.java/[CN]/	p_C.java/[CN]/
			`,
			[]float64{0.5, 0.5, 0.875, 1.25},
		},
		{
			`
			p_C1.java/[CN]/C1/[MT]/m1() p_C2.java/[CN]/C2/[MT]/m2()
			p_C3.java/[CN]/C3/[MT]/m3() p_C1.java/[CN]/C1/[MT]/m1()
			`,
			[]map[string]string{{entity("p_C1.java/[CN]/C1/[MT]/m1()").queryString(): "C2"}},
			`
			p_C1.java/[CN]/ p_C.java/[CN]/
			`,
			[]float64{0.33, 0.5, 1, 1},
		},
		{
			`
			p_C2.java/[CN]/C2/[MT]/m2() p_C1.java/[CN]/C1/[MT]/m1()
			p_C3.java/[CN]/C3/[MT]/m3() p_C1.java/[CN]/C1/[MT]/m1()
			p_C3.java/[CN]/C3/[MT]/m3() p_C1.java/[CN]/C1/[MT]/m11()
			p_C4.java/[CN]/C4/[MT]/m4() p_C1.java/[CN]/C1/[MT]/m1()
			p_C4.java/[CN]/C4/[MT]/m4() p_C1.java/[CN]/C1/[MT]/m11()
			`,
			[]map[string]string{{entity("p_C1.java/[CN]/C1/[MT]/m1()").queryString(): "C2"}},
			`
			p_C1.java/[CN]/ p_C.java/[CN]/
			`,
			[]float64{1.3333, 1.3333, 1, 1},
		},
	} {
		inh, err := newInheritance(strings.NewReader(d.inheritance))
		if err != nil {
			t.Fatal(err)
		}
		f, err := newFinder(strings.NewReader(d.mdg), nil)
		if err != nil {
			t.Fatal(err)
		}
		ii := improvements(d.reassignments, inh, nil, f, strings.NewReader(d.mdg))
		for i, metric := range []string{"pc", "cbo", "cam", "nop"} {
			if math.Signbit(ii[0][metric]) != math.Signbit(d.improvement[i]) ||
				math.Floor(math.Abs(ii[0][metric])*100+0.5) !=
					math.Floor(math.Abs(d.improvement[i])*100+0.5) {
				t.Errorf("Case %v/%v: Expected %v but was %v\n",
					testcase, i, d.improvement[i], ii[0][metric])
			}
		}
	}
}
