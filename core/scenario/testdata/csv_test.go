package testdata

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

func TestValidateCsvConf(t *testing.T) {
	t.Parallel()
	conf := types.CsvConf{
		Path:          "",
		Delimiter:     "",
		SkipFirstLine: false,
		Vars:          map[string]types.Tag{},
		SkipEmptyLine: false,
		AllowQuota:    false,
		Order:         "",
	}

	conf.Order = "invalidOrder"
	err := validateConf(conf)

	if err == nil {
		t.Errorf("TestValidateCsvConf should be errored")
	}
}

func TestReadCsv(t *testing.T) {
	t.Parallel()
	conf := types.CsvConf{
		Path:          "../../../config/config_testdata/test.csv",
		Delimiter:     ";",
		SkipFirstLine: true,
		Vars: map[string]types.Tag{
			"0": {Tag: "name", Type: "string"},
			"3": {Tag: "payload", Type: "json"},
			"4": {Tag: "age", Type: "int"},
			"5": {Tag: "percent", Type: "float"},
			"6": {Tag: "boolField", Type: "bool"},
		},
		SkipEmptyLine: true,
		AllowQuota:    true,
		Order:         "sequential",
	}

	rows, err := ReadCsv(conf)

	if err != nil {
		t.Errorf("TestReadCsv %v", err)
	}

	firstName := rows[0]["name"].(string)
	expectedName := "Kenan"
	if !strings.EqualFold(firstName, expectedName) {
		t.Errorf("TestReadCsv found: %s , expected: %s", firstName, expectedName)
	}

	firstAge := rows[0]["age"].(int)
	expectedAge := 25
	if firstAge != expectedAge {
		t.Errorf("TestReadCsv found: %d , expected: %d", firstAge, expectedAge)
	}

	firstPercent := rows[0]["percent"].(float64)
	expectedPercent := 22.3
	if firstPercent != expectedPercent {
		t.Errorf("TestReadCsv found: %f , expected: %f", firstPercent, expectedPercent)
	}

	firstBool := rows[0]["boolField"].(bool)
	expectedBool := true
	if firstBool != expectedBool {
		t.Errorf("TestReadCsv found: %t , expected: %t", firstBool, expectedBool)
	}

	firstPayload := rows[0]["payload"].(map[string]interface{})
	expectedPayload := map[string]interface{}{
		"data": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "Kenan",
			},
		},
	}
	if !reflect.DeepEqual(firstPayload, expectedPayload) {
		t.Errorf("TestReadCsv found: %#v , expected: %#v", firstPayload, expectedPayload)
	}

	secondPayload := rows[1]["payload"].([]interface{})
	expectedPayload2 := []interface{}{5.0, 6.0, 7.0} // underlying type float64
	if !reflect.DeepEqual(secondPayload, expectedPayload2) {
		t.Errorf("TestReadCsv found: %#v , expected: %#v", secondPayload, expectedPayload2)
	}
}

var table = []struct {
	conf    types.CsvConf
	latency float64
}{
	{
		conf: types.CsvConf{
			Path:          "config_testdata/test.csv",
			Delimiter:     ";",
			SkipFirstLine: true,
			Vars: map[string]types.Tag{
				"0": {Tag: "name", Type: "string"},
				"3": {Tag: "payload", Type: "json"},
				"4": {Tag: "age", Type: "int"},
				"5": {Tag: "percent", Type: "float"},
				"6": {Tag: "boolField", Type: "bool"},
			},
			SkipEmptyLine: true,
			AllowQuota:    true,
			Order:         "sequential",
		},
	},
}

func TestBenchmarkCsvRead(t *testing.T) {
	for _, v := range table {

		res := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ReadCsv(v.conf)
			}
		})

		fmt.Printf("ns:%d", res.T.Nanoseconds())
		fmt.Printf("N:%d", res.N)
	}
}
