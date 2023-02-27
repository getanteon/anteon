/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/types"
)

const ConfigTypeJson = "jsonReader"

func init() {
	AvailableConfigReader[ConfigTypeJson] = &JsonReader{}
}

type timeRunCount []struct {
	Duration int `json:"duration"`
	Count    int `json:"count"`
}

type auth struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type multipartFormData struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Src   string `json:"src"`
}

type RegexCaptureConf struct {
	Exp *string `json:"exp"`
	No  int     `json:"matchNo"`
}
type capturePath struct {
	JsonPath  *string           `json:"json_path"`
	XPath     *string           `json:"xpath"`
	RegExp    *RegexCaptureConf `json:"regexp"`
	From      string            `json:"from"`       // body,header
	HeaderKey *string           `json:"header_key"` // header key
}

type step struct {
	Id               uint16                 `json:"id"`
	Name             string                 `json:"name"`
	Url              string                 `json:"url"`
	Auth             auth                   `json:"auth"`
	Method           string                 `json:"method"`
	Headers          map[string]string      `json:"headers"`
	Payload          string                 `json:"payload"`
	PayloadFile      string                 `json:"payload_file"`
	PayloadMultipart []multipartFormData    `json:"payload_multipart"`
	Timeout          int                    `json:"timeout"`
	Sleep            string                 `json:"sleep"`
	Others           map[string]interface{} `json:"others"`
	CertPath         string                 `json:"cert_path"`
	CertKeyPath      string                 `json:"cert_key_path"`
	CaptureEnv       map[string]capturePath `json:"capture_env"`
	Assertions       []string               `json:"assertion"`
}

func (s *step) UnmarshalJSON(data []byte) error {
	type stepAlias step
	defaultFields := &stepAlias{
		Method:  types.DefaultMethod,
		Timeout: types.DefaultTimeout,
	}

	err := json.Unmarshal(data, defaultFields)
	if err != nil {
		return err
	}

	*s = step(*defaultFields)
	return nil
}

type Tag struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

func (t *Tag) UnmarshalJSON(data []byte) error {
	// default values
	t.Type = "string"
	type tempTag Tag
	return json.Unmarshal(data, (*tempTag)(t))
}

type CsvConf struct {
	Path          string         `json:"path"`
	Delimiter     string         `json:"delimiter"`
	SkipFirstLine bool           `json:"skip_first_line"`
	Vars          map[string]Tag `json:"vars"` // "0":"name", "1":"city","2":"team"
	SkipEmptyLine bool           `json:"skip_empty_line"`
	AllowQuota    bool           `json:"allow_quota"`
	Order         string         `json:"order"`
}

func (c *CsvConf) UnmarshalJSON(data []byte) error {
	// default values
	c.SkipEmptyLine = true
	c.SkipFirstLine = false
	c.AllowQuota = false
	c.Delimiter = ","
	c.Order = "random"

	type tempCsv CsvConf
	return json.Unmarshal(data, (*tempCsv)(c))
}

type JsonReader struct {
	ReqCount     *int                   `json:"request_count"`
	IterCount    *int                   `json:"iteration_count"`
	LoadType     string                 `json:"load_type"`
	Duration     int                    `json:"duration"`
	TimeRunCount timeRunCount           `json:"manual_load"`
	Steps        []step                 `json:"steps"`
	Output       string                 `json:"output"`
	Proxy        string                 `json:"proxy"`
	Envs         map[string]interface{} `json:"env"`
	Data         map[string]CsvConf     `json:"data"`
	Debug        bool                   `json:"debug"`
	SamplingRate *int                   `json:"sampling_rate"`
	EngineMode   string                 `json:"engine_mode"`
}

func (j *JsonReader) UnmarshalJSON(data []byte) error {
	type jsonReaderAlias JsonReader
	defaultFields := &jsonReaderAlias{
		LoadType:   types.DefaultLoadType,
		Duration:   types.DefaultDuration,
		Output:     types.DefaultOutputType,
		EngineMode: types.EngineModeDdosify,
	}

	err := json.Unmarshal(data, defaultFields)
	if err != nil {
		return err
	}

	*j = JsonReader(*defaultFields)
	return nil
}

func (j *JsonReader) Init(jsonByte []byte) (err error) {
	if !json.Valid(jsonByte) {
		err = fmt.Errorf("provided json is invalid")
		return
	}

	err = json.Unmarshal(jsonByte, &j)
	if err != nil {
		return
	}
	return
}

func (j *JsonReader) CreateHammer() (h types.Hammer, err error) {
	// Scenario
	s := types.Scenario{
		Envs: j.Envs,
	}
	var si types.ScenarioStep
	for _, step := range j.Steps {
		si, err = stepToScenarioStep(step)
		if err != nil {
			return
		}

		s.Steps = append(s.Steps, si)
	}

	// Proxy
	var proxyURL *url.URL
	if j.Proxy != "" {
		proxyURL, err = url.Parse(j.Proxy)
		if err != nil {
			return
		}
	}
	p := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     proxyURL,
	}

	// for backwards compatibility
	var iterationCount int
	if j.IterCount != nil {
		iterationCount = *j.IterCount
	} else if j.ReqCount != nil {
		iterationCount = *j.ReqCount
	} else {
		iterationCount = types.DefaultIterCount
	}
	j.IterCount = &iterationCount

	// TimeRunCount
	if len(j.TimeRunCount) > 0 {
		*j.IterCount, j.Duration = 0, 0
		for _, t := range j.TimeRunCount {
			*j.IterCount += t.Count
			j.Duration += t.Duration
		}
	}

	var samplingRate int
	if j.SamplingRate != nil {
		samplingRate = *j.SamplingRate
	} else {
		samplingRate = types.DefaultSamplingCount
	}

	testDataConf := make(map[string]types.CsvConf)
	for key, val := range j.Data {
		vars := make(map[string]types.Tag)
		for k, v := range val.Vars {
			vars[k] = types.Tag{
				Tag:  v.Tag,
				Type: v.Type,
			}
		}
		testDataConf[key] = types.CsvConf{
			Path:          val.Path,
			Delimiter:     val.Delimiter,
			SkipFirstLine: val.SkipFirstLine,
			Vars:          vars,
			SkipEmptyLine: val.SkipEmptyLine,
			AllowQuota:    val.AllowQuota,
			Order:         val.Order,
		}
	}

	// Hammer
	h = types.Hammer{
		IterationCount:    *j.IterCount,
		LoadType:          strings.ToLower(j.LoadType),
		TestDuration:      j.Duration,
		TimeRunCountMap:   types.TimeRunCount(j.TimeRunCount),
		Scenario:          s,
		Proxy:             p,
		ReportDestination: j.Output,
		Debug:             j.Debug,
		SamplingRate:      samplingRate,
		EngineMode:        j.EngineMode,
		TestDataConf:      testDataConf,
	}
	return
}

func stepToScenarioStep(s step) (types.ScenarioStep, error) {
	var payload string
	var err error
	if len(s.PayloadMultipart) > 0 {
		if s.Headers == nil {
			s.Headers = make(map[string]string)
		}

		payload, s.Headers["Content-Type"], err = prepareMultipartPayload(s.PayloadMultipart)
		if err != nil {
			return types.ScenarioStep{}, err
		}
	} else if s.PayloadFile != "" {
		buf, err := ioutil.ReadFile(s.PayloadFile)
		if err != nil {
			return types.ScenarioStep{}, err
		}

		payload = string(buf)
	} else {
		payload = s.Payload
	}

	// Set default Auth type if not set
	if s.Auth != (auth{}) && s.Auth.Type == "" {
		s.Auth.Type = types.AuthHttpBasic
	}

	err = types.IsTargetValid(s.Url)
	if err != nil {
		return types.ScenarioStep{}, err
	}

	var capturedEnvs []types.EnvCaptureConf
	for name, path := range s.CaptureEnv {
		capConf := types.EnvCaptureConf{
			JsonPath: path.JsonPath,
			Xpath:    path.XPath,
			Name:     name,
			From:     types.SourceType(path.From),
			Key:      path.HeaderKey,
		}

		if path.RegExp != nil {
			capConf.RegExp = &types.RegexCaptureConf{
				Exp: path.RegExp.Exp,
				No:  path.RegExp.No,
			}
		}

		capturedEnvs = append(capturedEnvs, capConf)
	}

	item := types.ScenarioStep{
		ID:            s.Id,
		Name:          s.Name,
		URL:           s.Url,
		Auth:          types.Auth(s.Auth),
		Method:        strings.ToUpper(s.Method),
		Headers:       s.Headers,
		Payload:       payload,
		Timeout:       s.Timeout,
		Sleep:         strings.ReplaceAll(s.Sleep, " ", ""),
		Custom:        s.Others,
		EnvsToCapture: capturedEnvs,
		Assertions:    s.Assertions,
	}

	if s.CertPath != "" && s.CertKeyPath != "" {
		cert, pool, err := types.ParseTLS(s.CertPath, s.CertKeyPath)
		if err != nil {
			return item, err
		}

		item.Cert = cert
		item.CertPool = pool
	}

	return item, nil
}

func prepareMultipartPayload(parts []multipartFormData) (body string, contentType string, err error) {
	byteBody := &bytes.Buffer{}
	writer := multipart.NewWriter(byteBody)

	for _, part := range parts {
		var err error

		if strings.EqualFold(part.Type, "file") {
			if strings.EqualFold(part.Src, "remote") {
				response, err := http.Get(part.Value)
				if err != nil {
					return "", "", err
				}
				defer response.Body.Close()

				u, _ := url.Parse(part.Value)
				formPart, err := writer.CreateFormFile(part.Name, path.Base(u.Path))
				if err != nil {
					return "", "", err
				}

				_, err = io.Copy(formPart, response.Body)
				if err != nil {
					return "", "", err
				}
			} else {
				file, err := os.Open(part.Value)
				defer file.Close()
				if err != nil {
					return "", "", err
				}

				formPart, err := writer.CreateFormFile(part.Name, filepath.Base(file.Name()))
				if err != nil {
					return "", "", err
				}

				_, err = io.Copy(formPart, file)
				if err != nil {
					return "", "", err
				}
			}

		} else {
			// If we have to specify Content-Type in Content-Disposition, we should use writer.CreatePart directly.
			err = writer.WriteField(part.Name, part.Value)
			if err != nil {
				return "", "", err
			}
		}
	}

	writer.Close()
	return byteBody.String(), writer.FormDataContentType(), err
}
