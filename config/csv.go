package config

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func validateConf(conf CsvConf) error {
	if !(conf.Order == "random" || conf.Order == "sequential") {
		return fmt.Errorf("unsupported order %s, should be random|sequential", conf.Order)
	}
	return nil
}

func readCsv(conf CsvConf) ([]map[string]interface{}, error) {
	err := validateConf(conf)
	if err != nil {
		return nil, err
	}

	var reader io.Reader

	if _, err = os.Stat(conf.Path); err == nil { // local file path
		f, err := os.Open(conf.Path)
		if err != nil {
			return nil, err
		}
		reader = f
		defer f.Close()
	} else if _, err = url.ParseRequestURI(conf.Path); err == nil { // url
		req, err := http.NewRequest(http.MethodGet, conf.Path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		reader = resp.Body
		defer resp.Body.Close()
	} else {
		return nil, fmt.Errorf("given path is neither local path nor url")
	}

	// read csv values using csv.Reader
	csvReader := csv.NewReader(reader)
	csvReader.Comma = []rune(conf.Delimiter)[0]
	csvReader.TrimLeadingSpace = true
	csvReader.LazyQuotes = conf.AllowQuota

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
		for index, tag := range conf.Vars {
			i, err := strconv.Atoi(index)
			if err != nil {
				return nil, err
			}

			if i >= len(row) {
				return nil, fmt.Errorf("index number out of range, check your vars or delimiter")
			}

			// convert
			var val interface{}
			switch tag.Type {
			case "json":
				json.Unmarshal([]byte(row[i]), &val)
			case "int":
				var err error
				val, err = strconv.Atoi(row[i])
				if err != nil {
					return nil, err
				}
			case "float":
				var err error
				val, err = strconv.ParseFloat(row[i], 64)
				if err != nil {
					return nil, err
				}
			case "bool":
				var err error
				val, err = strconv.ParseBool(row[i])
				if err != nil {
					return nil, err
				}
			default:
				val = row[i]
			}
			x[tag.Tag] = val
		}
		rt = append(rt, x)
	}

	return rt, nil
}

func emptyLine(row []string) bool {
	for _, field := range row {
		if field != "" {
			return false
		}
	}
	return true
}