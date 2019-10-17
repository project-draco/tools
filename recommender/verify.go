package main

/*
func main() {
	if len(os.Args) < 4 {
		fmt.Printf("usage: %v <co-change.mdg> <static.mdg> <static-errors.txt>\n", path.Base(os.Args[0]))
		os.Exit(1)
	}
	finder, err := newFinder(os.Args[2], os.Args[3])
	check(err)
	ccf, err := os.Open(os.Args[1])
	check(err)
	defer ccf.Close()
	s := bufio.NewScanner(ccf)
	found, notfound := 0, 0
	for s.Scan() {
		arr := strings.Split(s.Text(), "\t")
		if len(arr) == 1 {
			arr = strings.Split(s.Text(), " ")
		}
		for i := 0; i < 2; i++ {
			if strings.HasSuffix(arr[i], "/package") || strings.HasSuffix(arr[i], "/extend") {
				continue
			}
			if finder.find(entity(arr[i])) {
				found++
			} else {
				fmt.Println(arr[i])
				notfound++
			}
		}
	}
	check(s.Err())
	fmt.Printf("%v found, %v not found\n", found, notfound)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
*/
