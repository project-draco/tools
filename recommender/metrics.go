package main

func cam(s *structure) float64 {
	result := 0.0
	for _, f := range s.Files() {
		allParametersTypes := map[string]string{}
		for _, m := range s.Methods(f) {
			for _, p := range m.Parameters {
				allParametersTypes[p] = p
			}
		}
		if len(allParametersTypes) == 0 {
			continue
		}
		sum := 0.0
		for _, m := range s.Methods(f) {
			parametersTypes := map[string]string{}
			for _, p := range m.Parameters {
				parametersTypes[p] = p
			}
			sum += float64(len(parametersTypes)) / float64(len(allParametersTypes))
		}
		if len(s.Methods(f)) != 0 {
			result += sum / float64(len(s.Methods(f)))
		}
	}
	if len(s.Files()) == 0 {
		return 0
	}
	return result / float64(len(s.Files()))
}

func cbo(m [][]int) float64 {
	count := 0
	for i, successors := range m {
		for _, succ := range successors {
			if succ != i {
				count++
			}
		}
	}
	return float64(count) / float64(len(m))
}

func cis(s *structure, f *finder) float64 {
	sum := 0.0
	for _, file := range s.Files() {
		sum += float64(s.PublicMethodsCount(file, f))
	}
	return sum / float64(len(s.Files()))
}

func dam(s *structure, f *finder) float64 {
	sum := 0.0
	for _, file := range s.Files() {
		mc := float64(len(s.Methods(file)))
		if mc == 0 {
			continue
		}
		sum += (mc - float64(s.PublicMethodsCount(file, f))) / mc
	}
	if len(s.Files()) == 0 {
		return 0
	}
	return sum / float64(len(s.Files()))
}

func dsc(s *structure) float64 {
	return float64(len(s.Files()))
}

func moa(s *structure, ft *fieldTypes) float64 {
	sum := 0.0
	for _, f := range s.Files() {
		types := map[string]string{}
		for _, field := range s.Fields(f) {
			t := ft.TypeOf(field)
			types[t] = t
		}
		sum += float64(len(types))
	}
	return sum / float64(len(s.Files()))
}

func mpc(m [][]int, w [][]int) float64 {
	sum := 0
	for i, successors := range m {
		for j, succ := range successors {
			if succ != i {
				sum += w[i][j]
			}
		}
	}
	if len(m) == 0 {
		return 0
	}
	return float64(sum) / float64(len(m))
}

func nom(s *structure) float64 {
	sum := 0.0
	for _, f := range s.Files() {
		sum += float64(len(s.Methods(f)))
	}
	return sum / float64(len(s.Files()))
}

func nop(s *structure, inh *inheritance) float64 {
	sum := 0.0
	for _, c := range inh.OutboundList() {
		child, parent := inh.File(c[0]), inh.File(c[1])
		for _, cm := range s.Methods(child) {
			for _, pm := range s.Methods(parent) {
				if cm.String() == pm.String() {
					sum++
				}
			}
		}
	}
	return sum / float64(len(s.Files()))
}

func propagationCost(m [][]int) int {
	buf := make([]bool, len(m)*len(m))
	tc := make([][]bool, len(m))
	for i := 0; i < len(m); i++ {
		tc[i] = buf[i*len(m) : (i+1)*len(m)]
		dfs(m, tc, i, i)
	}
	pc := 0
	for i := 0; i < len(m); i++ {
		for j := 0; j < len(m); j++ {
			if i != j && tc[i][j] {
				pc++
			}
		}
	}
	return pc
}

func dfs(m [][]int, tc [][]bool, s, v int) {
	tc[s][v] = true
	for i := 0; i < len(m[v]); i++ {
		if !tc[s][m[v][i]] {
			dfs(m, tc, s, m[v][i])
		}
	}
}
