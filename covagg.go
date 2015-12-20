package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type lineData struct {
	line  string
	count int64
}

func main() {
	fileName := flag.String("file", "", "Full path for the file to be parsed")
	output := flag.String("output", "agg_cov.cov", "Output file fpr aggregation results")
	flag.Parse()
	if fileName == nil || len(*fileName) == 0 {
		fmt.Errorf("file flag is mandatory run -h")
	}
	fmt.Println("Going to parse file: ", *fileName)
	pf, err := os.Open(*fileName)
	if err != nil {
		fmt.Errorf("Failed opening file %v", err)
		return
	}
	defer pf.Close()

	buf := bufio.NewReader(pf)
	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	// encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count
	// Regex taken from: https://github.com/golang/tools/blob/master/cover/profile.go (line 113)
	lineRe := regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)
	s := bufio.NewScanner(buf)
	uniqBuf := make(map[string]*lineData)
	mod := ""
	for s.Scan() {
		line := s.Text()
		data := lineData{}
		shortLine := ""
		if strings.Contains(line, "mode:") {
			if len(mod) == 0 {
				mod = line
			} else if mod != line {
				fmt.Errorf("Cannot aggregate coverage in different modes %s, %s\n exiting", mod, line)
				return
			}
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			fmt.Errorf("line %s doesn't match expected format: %v", line, lineRe)
			return
		}
		count := m[7]
		if last := strings.LastIndex(line, count); last >= 0 {
			shortLine = line[:last]
			data.line = shortLine
			data.count, err = strconv.ParseInt(count, 10, 64)
			if err != nil {
				fmt.Errorf("Failed parsing line count %s, %s", line, err.Error())
			}
		}
		oldData, ok := uniqBuf[shortLine]
		if !ok {
			uniqBuf[shortLine] = &data
		} else {
			oldData.count += data.count
		}
	}
	f, err := os.Create(*output)
	defer f.Close()
	if err != nil {
		fmt.Errorf("Failed opening output file %s\nexiting", err.Error())
		return
	}
	_, err = f.WriteString(mod + "\n")
	if err != nil {
		fmt.Errorf("Failed writing to file %s\nexiting", err.Error())
		return
	}
	for _, value := range uniqBuf {
		str := fmt.Sprintf("%s%d%s", value.line, value.count, "\n")
		_, err = f.WriteString(str)
		if err != nil {
			fmt.Errorf("Failed writing %s to file %s\nexiting", value.line, err.Error())
			return
		}
	}
	fmt.Println("Done writing aggregated coverage results to:", *output)
}
