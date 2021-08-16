package types

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

//TODO: JSON marsh,unmarsh kalkabilir buradan. İhityac var mı emin olamadım.

type Scenario struct {
	Scenario []ScenarioItem
}

func (sc *Scenario) String() string {
	var s string
	for i, sci := range sc.Scenario {
		s += fmt.Sprintf("Index: %d \n %s \n", i, sci.String())
	}
	return s
}

type ScenarioItem struct {
	// Target URL
	URL url.URL `json:"url,string"`

	// Connection timeout duration of the request in miliseconds
	Timeout int `json:"timeout,int"`
}

func (sci *ScenarioItem) String() string {
	return fmt.Sprintf("URL: %s, Timeout: %d", &sci.URL, sci.Timeout)
}

// We need to implement Unmarshaler interface for URL
func (sci *ScenarioItem) UnmarshalJSON(j []byte) error {
	var rawStrings map[string]string

	err := json.Unmarshal(j, &rawStrings)
	if err != nil {
		return err
	}

	for k, v := range rawStrings {
		if strings.ToLower(k) == "timeout" {
			sci.Timeout, err = strconv.Atoi(v)
			if err != nil {
				return err
			}
		}
		if strings.ToLower(k) == "url" {
			u, err := url.Parse(v)
			if err != nil {
				return err
			}
			sci.URL = *u
		}
	}
	return nil
}

// TODO: REMOVE
// USAGE EXAMPLE
// scJson := `{"scenario": [{"url": "google.com","timeout": "55"}, {"url": "facebook.com","timeout": "17"}]}`
// var sc ScenarioContext
// json.Unmarshal([]byte(scJson), &sc)
// fmt.Println(sc.String())
