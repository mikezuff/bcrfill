package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
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

	ess := expandSurveySet(ss, species)
	fmt.Fprintf(os.Stderr, "expanded from %d to %d\n", len(ss), len(ess))
	writeSurveySet(ess)
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
		return false
	}

	for i, e := range surveyHeader {
		if e != in[i] {
			return false
		}
	}
	return true
}

func readSurveySet() surveySet {
	cr := csv.NewReader(os.Stdin)
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

func expandSurveySet(ss surveySet, species []int) surveySet {
	ess := make([]survey, 0, len(ss))
	i := 0
	tuples := 0
	for {
		if i == len(ss) {
			fmt.Fprintf(os.Stderr, "%d tuples\n", tuples)
			return ess
		}

		route := ss[i].Route
		year := ss[i].Year
		tuples++
		inTuple := true
		for _, aou := range species {
			if inTuple && aou == ss[i].Species {
				ess = append(ess, ss[i])
				/* eat the real entry, if next entry has different
				{route, year} then we just fill with empties */
				i++
				if i == len(ss) || (ss[i].Route != route || ss[i].Year != year) {
					inTuple = false
				}
			} else {
				ess = append(ess, emptySurvey(route, year, aou))
			}
		}
	}
}
