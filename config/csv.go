package config

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

func readCsv(conf CsvConf) ([]map[string]interface{}, error) {
	if conf.Src == "local" {
		f, err := os.Open(conf.Path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		// read csv values using csv.Reader
		csvReader := csv.NewReader(f)
		csvReader.Comma = []rune(conf.Delimiter)[0]
		csvReader.TrimLeadingSpace = true
		data, err := csvReader.ReadAll()
		if err != nil {
			return nil, err
		}

		if conf.SkipFirstLine {
			data = data[1:]
		}

		rt := make([]map[string]interface{}, 0) // unclear how many empty line exist

		for _, row := range data {
			if conf.SkipEmptyLine && emptyLine(row) {
				continue
			}
			x := map[string]interface{}{}
			for index, tag := range conf.Vars { // "0":"name", "1":"city","2":"team"
				i, err := strconv.Atoi(index)
				if err != nil {
					return nil, err
				}
				x[tag] = row[i]
			}
			rt = append(rt, x)
		}

		return rt, nil

	} else if conf.Src == "remote" {
		// TODOcorr, http call
	}

	return nil, fmt.Errorf("csv read error")
}

func emptyLine(row []string) bool {
	for _, field := range row {
		if field != "" {
			return false
		}
	}
	return true
}
