package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"go.ddosify.com/ddosify/core/types"
)

func validateConf(conf types.CsvConf) error {
	if !(conf.Order == "random" || conf.Order == "sequential") {
		return fmt.Errorf("unsupported order %s, should be random|sequential", conf.Order)
	}
	return nil
}

type RemoteCsvError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (nf RemoteCsvError) Error() string {
	if nf.wrappedErr != nil {
		return fmt.Sprintf("%s,%s", nf.msg, nf.wrappedErr.Error())
	}
	return nf.msg
}

func (nf RemoteCsvError) Unwrap() error {
	return nf.wrappedErr
}

func ReadCsv(conf types.CsvConf) ([]map[string]interface{}, error) {
	err := validateConf(conf)
	if err != nil {
		return nil, err
	}

	var reader io.Reader

	var pUrl *url.URL
	if pUrl, err = url.ParseRequestURI(conf.Path); err == nil && pUrl.IsAbs() { // url
		req, err := http.NewRequest(http.MethodGet, conf.Path, nil)
		if err != nil {
			return nil, wrapAsCsvError("can not create request", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, wrapAsCsvError("can not get response", err)
		}

		if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
			return nil, wrapAsCsvError(fmt.Sprintf("request to remote url (%s) failed. Status Code: %d", conf.Path, resp.StatusCode), nil)
		}
		reader = resp.Body
		defer resp.Body.Close()
	} else if _, err = os.Stat(conf.Path); err == nil { // local file path
		f, err := os.Open(conf.Path)
		if err != nil {
			return nil, err
		}
		reader = f
		defer f.Close()
	} else {
		return nil, wrapAsCsvError(fmt.Sprintf("can not parse path: %s", conf.Path), err)
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
				err := json.Unmarshal([]byte(row[i]), &val)
				if err != nil {
					return nil, fmt.Errorf("can not convert %s to json,%v", row[i], err)
				}
			case "int":
				var err error
				val, err = strconv.Atoi(row[i])
				if err != nil {
					return nil, fmt.Errorf("can not convert %s to int,%v", row[i], err)
				}
			case "float":
				var err error
				val, err = strconv.ParseFloat(row[i], 64)
				if err != nil {
					return nil, fmt.Errorf("can not convert %s to float,%v", row[i], err)
				}
			case "bool":
				var err error
				val, err = strconv.ParseBool(row[i])
				if err != nil {
					return nil, fmt.Errorf("can not convert %s to bool,%v", row[i], err)
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

func wrapAsCsvError(msg string, err error) RemoteCsvError {
	var csvReqError RemoteCsvError
	csvReqError.msg = msg
	csvReqError.wrappedErr = err
	return csvReqError
}
