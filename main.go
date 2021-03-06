package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

/*
For each route/year combo, I would like a row for each of 87 species (see attached txt file of AOU species codes).
*/

func main() {
	var (
		sFn string
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -s speciesList.csv < bcrnozeroes.csv > bcrwithzeroes.csv\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n Ensure every {species, year} tuple is present in csv.\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.StringVar(&sFn, "s", "", "File containing full species list")
	flag.Parse()

	if sFn == "" {
		exit("-s must be provided")
	}

	species, err := loadSpecies(sFn)
	if err != nil {
		exit(fmt.Sprintf("error loading species: %s", err))
	}

	//	fmt.Fprintf(os.Stderr, "loaded %d species %v\n", len(species), species)

	ss := readSurveySet()
	sort.Stable(ss)
	/*
		fmt.Printf("%d lines:\n", len(recs))
		for i, r := range recs {
			fmt.Println(i, r)
		}
	*/

	out := expandSurveySet(ss, species)
	fmt.Fprintf(os.Stderr, "expanded from %d to %d\n", len(ss), out)
}

func exit(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	flag.Usage()
	os.Exit(1)
}

func loadSpecies(fn string) ([]int, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	species := make([]int, 0, 10)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		tok := scanner.Text()
		i, err := strconv.ParseInt(tok, 10, 64)
		if err != nil {
			exit(fmt.Sprintf("bad species %q", tok))
		}

		species = append(species, int(i))
	}

	return species, scanner.Err()
}

type survey struct {
	Route   int
	Year    int
	Species int
	Data    [7]string
}

func fromStrings(in []string) (survey, error) {
	if len(in) < 10 {
		return survey{}, fmt.Errorf("expected min 10 columns: %v", in)
	}

	route, err := strconv.ParseInt(in[0], 10, 64)
	if err != nil {
		return survey{}, fmt.Errorf("parsing route: %s", err)
	}
	year, err := strconv.ParseInt(in[1], 10, 64)
	if err != nil {
		return survey{}, fmt.Errorf("parsing year: %s", err)
	}
	species, err := strconv.ParseInt(in[2], 10, 64)
	if err != nil {
		return survey{}, fmt.Errorf("parsing species: %s", err)
	}

	s := survey{Route: int(route), Year: int(year), Species: int(species)}
	copy(s.Data[:], in[3:])
	return s, nil
}

func (s survey) toStrings() []string {
	return []string{
		strconv.Itoa(s.Route),
		strconv.Itoa(s.Year),
		strconv.Itoa(s.Species),
		s.Data[0],
		s.Data[1],
		s.Data[2],
		s.Data[3],
		s.Data[4],
		s.Data[5],
		s.Data[6],
	}
}

func emptySurvey(route, year, species int) survey {
	return survey{route, year, species, [7]string{"0", "0", "0", "0", "0", "0", "0"}}
}

type surveySet []survey

func (ss surveySet) Len() int      { return len(ss) }
func (ss surveySet) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss surveySet) Less(i, j int) bool {
	a := &ss[i]
	b := &ss[j]

	return a.Route < b.Route ||
		(a.Route == b.Route && a.Year < b.Year) ||
		(a.Route == b.Route && a.Year == b.Year && a.Species < b.Species)
}

var (
	surveyHeader = []string{"Route", "Year", "AOU", "First", "Second", "Third", "Fourth", "Fifth", "Stops", "Count"}
)

func isHeader(in []string) bool {
	if len(in) < len(surveyHeader) {
		fmt.Fprintf(os.Stderr, "expected %d columns, input has %d\n", len(surveyHeader), len(in))
		return false
	}

	for i, e := range surveyHeader {
		if e != in[i] {
			fmt.Fprintf(os.Stderr, "column %d labeled \"%s\", expected \"%s\"\n", i, in[i], e)
			return false
		}
	}
	return true
}

type replaceReader struct {
	offset int
	r      io.Reader
	from   byte
	to     byte
}

func (rr *replaceReader) Read(p []byte) (n int, err error) {
	n, err = rr.r.Read(p)
	for i, b := range p[:n] {
		if b == rr.from {
			p[i] = rr.to
			fmt.Fprintf(os.Stderr, "replaced 0x%x with 0x%x at %d\n", rr.from, rr.to, rr.offset)
		}
		rr.offset++
	}
	return
}

func readSurveySet() surveySet {
	rr := &replaceReader{
		r:    os.Stdin,
		from: 0xd,
		to:   0xa,
	}
	cr := csv.NewReader(rr)
	recs, err := cr.ReadAll()
	if err != nil {
		exit(err.Error())
	}

	if !isHeader(recs[0]) {
		exit("bad csv header")
	}
	recs = recs[1:]

	ss := make([]survey, len(recs))
	for i, in := range recs {
		ss[i], err = fromStrings(in)
		if err != nil {
			exit(fmt.Sprintf("line %d: %s", i+1, err))
		}
	}

	return surveySet(ss)
}

func writeSurveySet(ss surveySet) {
	cw := csv.NewWriter(os.Stdout)
	cw.Write(surveyHeader)
	for _, s := range ss {
		cw.Write(s.toStrings())
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		exit(fmt.Sprintf("error writing CSV: %s", err))
	}
}

func toMap(l []int) map[int]int {
	m := make(map[int]int, len(l))
	for _, v := range l {
		m[v] = 1
	}
	return m
}

func expandSurveySet(ss surveySet, species []int) int {
	aouMap := toMap(species)
	cw := csv.NewWriter(os.Stdout)
	cw.Write(surveyHeader)
	outCount := 0
	i := 0
	tuples := 0
	seen := make(map[string]int)
	for {
		if i == len(ss) {
			cw.Flush()
			if err := cw.Error(); err != nil {
				exit(fmt.Sprintf("error writing CSV: %s", err))
			}

			fmt.Fprintf(os.Stderr, "unique {route,year} tuples: %d\n", tuples)
			return outCount
		}

		route := ss[i].Route
		year := ss[i].Year
		label := fmt.Sprintf("{%d,%d}", route, year)
		if seenOutCount, ok := seen[label]; ok {
			cw.Flush()
			fmt.Fprintf(os.Stderr, "already seen this {route,year} tuple at line %d: %s, bailing out\n", seenOutCount, label)
			return outCount
		} else {
			seen[label] = outCount
		}

		tuples++
		inTuple := true
		for _, aou := range species {
			if inTuple && aou == ss[i].Species {
				cw.Write(ss[i].toStrings())
				outCount++
				/* eat the real entry, if next entry has different
				{route, year} then we just fill with empties */
				i++
				for i < len(ss) && aouMap[ss[i].Species] == 0 {
					// Ignore invalid species
					fmt.Fprintf(os.Stderr, "ignoring unknown species %d line %d\n", ss[i].Species, i)
					i++
				}
				if i == len(ss) || (ss[i].Route != route || ss[i].Year != year) {
					inTuple = false
				}
			} else {
				cw.Write(emptySurvey(route, year, aou).toStrings())
				outCount++
			}
		}
	}
}
