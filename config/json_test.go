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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/report"
	"go.ddosify.com/ddosify/core/types"
)

func TestCreateHammerDefaultValues(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_empty.json"), ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    types.DefaultIterCount,
		LoadType:          types.DefaultLoadType,
		TestDuration:      types.DefaultDuration,
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %#v, \nFound: %#v", expectedHammer, h)
	}
}

func TestCreateHammer(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config.json"), ConfigTypeJson)
	addr, _ := url.Parse("http://proxy_host:80")
	expectedHammer := types.Hammer{
		IterationCount:    1555,
		LoadType:          types.LoadTypeWaved,
		TestDuration:      21,
		ReportDestination: report.OutputTypeStdout,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{
				{
					ID:      1,
					Name:    "Example Name 1",
					URL:     "https://app.servdown.com/accounts/login/?next=/",
					Method:  http.MethodGet,
					Timeout: 3,
					Sleep:   "1000",
					Payload: "payload str",
					Custom: map[string]interface{}{
						"keep-alive": true,
					},
				},
				{
					ID:      2,
					Name:    "Example Name 2",
					URL:     "http://test.com",
					Method:  http.MethodPut,
					Timeout: 2,
					Sleep:   "300-500",
					Headers: map[string]string{
						"ContenType":    "application/xml",
						"X-ddosify-key": "ajkndalnasd",
					},
				},
			},
		},
		Proxy: proxy.Proxy{
			Strategy: "single",
			Addr:     addr,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammer error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %v,\n Found: %v", expectedHammer, h)
	}
}

func TestCreateHammerWithIterationCountInsteadOfReqCount(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_iteration_count.json"), ConfigTypeJson)
	addr, _ := url.Parse("http://proxy_host:80")
	expectedHammer := types.Hammer{
		IterationCount:    1555,
		LoadType:          types.LoadTypeWaved,
		TestDuration:      21,
		ReportDestination: report.OutputTypeStdout,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{
				{
					ID:      1,
					Name:    "Example Name 1",
					URL:     "https://app.servdown.com/accounts/login/?next=/",
					Method:  http.MethodGet,
					Timeout: 3,
					Sleep:   "1000",
					Payload: "payload str",
					Custom: map[string]interface{}{
						"keep-alive": true,
					},
				},
				{
					ID:      2,
					Name:    "Example Name 2",
					URL:     "http://test.com",
					Method:  http.MethodPut,
					Timeout: 2,
					Sleep:   "300-500",
					Headers: map[string]string{
						"ContenType":    "application/xml",
						"X-ddosify-key": "ajkndalnasd",
					},
				},
			},
		},
		Proxy: proxy.Proxy{
			Strategy: "single",
			Addr:     addr,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammer error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %v,\n Found: %v", expectedHammer, h)
	}
}

func TestCreateHammerWithIterationCountOverridesReqCount(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_iteration_count_over_req_count.json"),
		ConfigTypeJson)
	addr, _ := url.Parse("http://proxy_host:80")
	expectedHammer := types.Hammer{
		IterationCount:    333,
		LoadType:          types.LoadTypeWaved,
		TestDuration:      21,
		ReportDestination: report.OutputTypeStdout,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{
				{
					ID:   1,
					Name: "Example Name 1",
					URL:  "https://app.servdown.com/accounts/login/?next=/",
					// Protocol: types.ProtocolHTTPS,
					Method:  http.MethodGet,
					Timeout: 3,
					Sleep:   "1000",
					Payload: "payload str",
					Custom: map[string]interface{}{
						"keep-alive": true,
					},
				},
				{
					ID:   2,
					Name: "Example Name 2",
					URL:  "http://test.com",
					// Protocol: types.ProtocolHTTP,
					Method:  http.MethodPut,
					Timeout: 2,
					Sleep:   "300-500",
					Headers: map[string]string{
						"ContenType":    "application/xml",
						"X-ddosify-key": "ajkndalnasd",
					},
				},
			},
		},
		Proxy: proxy.Proxy{
			Strategy: "single",
			Addr:     addr,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammer error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %v,\n Found: %v", expectedHammer, h)
	}
}

func TestCreateHammerManualLoad(t *testing.T) {
	t.Parallel()

	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_manual_load.json"), ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    35,
		LoadType:          types.DefaultLoadType,
		TestDuration:      18,
		TimeRunCountMap:   types.TimeRunCount{{Duration: 5, Count: 5}, {Duration: 6, Count: 10}, {Duration: 7, Count: 20}},
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerManualLoad error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("Expected: %v, Found: %v", expectedHammer, h)
	}
}

func TestCreateHammerManualLoadOverrideOthers(t *testing.T) {
	t.Parallel()

	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_manual_load_override.json"), ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    35,
		LoadType:          types.DefaultLoadType,
		TestDuration:      18,
		TimeRunCountMap:   types.TimeRunCount{{Duration: 5, Count: 5}, {Duration: 6, Count: 10}, {Duration: 7, Count: 20}},
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerManualLoad error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("Expected: %v, Found: %v", expectedHammer, h)
	}
}

func TestCreateHammerPayload(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_payload.json"), ConfigTypeJson)
	expectedPayloads := []string{"payload from string", "Payloaf from file."}
	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerPayload error occurred: %v", err)
	}

	steps := h.Scenario.Steps

	if steps[0].Payload != expectedPayloads[0] {
		t.Errorf("Expected: %v, Found: %v", expectedPayloads[0], steps[0].Payload)
	}

	if steps[1].Payload != expectedPayloads[1] {
		t.Errorf("Expected: %v, Found: %v", expectedPayloads[1], steps[1].Payload)
	}
}

func TestCreateHammerMultipartPayload(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_multipart_payload.json"), ConfigTypeJson)

	h, err := jsonReader.CreateHammer()
	if err != nil {
		t.Errorf("TestCreateHammerMultipartPayload error occurred: %v", err)
	}
	steps := h.Scenario.Steps

	// Content-Type Header Check
	val, ok := steps[0].Headers["Content-Type"]
	if !ok {
		t.Error("Content-Type header should be exist")
	}

	rgx := "multipart/form-data; boundary=.*"
	r, _ := regexp.Compile(rgx)
	if !r.MatchString(val) {
		t.Errorf("Expected: %v, Found: %v", rgx, val)
	}

	// Payload Check - Ensure that payload contains 4 form field.
	if c := strings.Count(steps[0].Payload, "Content-Disposition: form-data;"); c != 4 {
		t.Errorf("Expected: %v, Found: %v", 4, c)
	}
}

func TestCreateHammerAuth(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_auth.json"), ConfigTypeJson)
	expectedAuths := []types.Auth{
		{
			Type:     types.AuthHttpBasic,
			Username: "kursat",
			Password: "12345",
		},
		{}}

	h, err := jsonReader.CreateHammer()
	if err != nil {
		t.Errorf("TestCreateHammerAuth error occurred: %v", err)
	}

	steps := h.Scenario.Steps
	if steps[0].Auth != expectedAuths[0] {
		t.Errorf("Expected: %v, Found: %v", expectedAuths[0], steps[0].Auth)
	}

	if steps[1].Auth != expectedAuths[1] {
		t.Errorf("Expected: %v, Found: %v", expectedAuths[1], steps[1].Auth)
	}
}

func TestCreateHammerGlobalEnvs(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_global_envs.json"), ConfigTypeJson)
	expectedGlobalEnvs := map[string]interface{}{
		"HTTPBIN": "https://httpbin.ddosify.com",
		"LOCAL":   "http://localhost:8084/hello",
	}

	h, err := jsonReader.CreateHammer()
	if err != nil {
		t.Errorf("TestCreateHammerGlobalEnvs error occurred: %v", err)
	}

	globalEnvs := h.Scenario.Envs

	if !reflect.DeepEqual(globalEnvs, expectedGlobalEnvs) {
		t.Errorf("TestCreateHammerGlobalEnvs global envs got: %#v expected: %#v", globalEnvs, expectedGlobalEnvs)
	}
}

func TestCreateHammerCaptureEnvs(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_capture_environment.json"), ConfigTypeJson)
	json_path := "num"
	expectedEnvsToCapture0 := []types.EnvCaptureConf{{
		Name:     "NUM",
		From:     types.Body,
		JsonPath: &json_path,
	}}

	regex := "[a-z]+_[0-9]+"
	expectedEnvsToCapture1 := []types.EnvCaptureConf{{
		Name: "REGEX_MATCH_ENV",
		From: types.Body,
		RegExp: &types.RegexCaptureConf{
			Exp: &regex,
			No:  1,
		},
	}}

	h, err := jsonReader.CreateHammer()
	if err != nil {
		t.Errorf("TestCreateHammerCaptureEnvs error occurred: %v", err)
	}

	envsToCapture0 := h.Scenario.Steps[0].EnvsToCapture

	if !reflect.DeepEqual(envsToCapture0, expectedEnvsToCapture0) {
		t.Errorf("TestCreateHammerCaptureEnvs global envs got: %#v expected: %#v", envsToCapture0, expectedEnvsToCapture0)
	}

	envsToCapture1 := h.Scenario.Steps[1].EnvsToCapture

	if !reflect.DeepEqual(envsToCapture1, expectedEnvsToCapture1) {
		t.Errorf("TestCreateHammerCaptureEnvs global envs got: %#v expected: %#v", envsToCapture1, expectedEnvsToCapture1)
	}
}

func TestCreateHammerInvalidTarget(t *testing.T) {
	t.Parallel()
	jsonReader, _ := NewConfigReader(readConfigFile("config_testdata/config_invalid_target.json"), ConfigTypeJson)

	_, err := jsonReader.CreateHammer()
	if err == nil {
		t.Errorf("TestCreateHammerProtocol error occurred")
	}
}

func TestCreateHammerTLS(t *testing.T) {
	t.Parallel()

	// prepare TLS files
	cert, certKey := generateCerts()
	certFile, keyFile, err := createCertPairFiles(cert, certKey)
	if err != nil {
		t.Fatalf("Failed to prepare certs %v", err)
	}
	defer os.Remove(certFile.Name())
	defer os.Remove(keyFile.Name())

	config := buildJSONTLSConfig(certFile.Name(), keyFile.Name())

	jsonReader, _ := NewConfigReader(config, ConfigTypeJson)

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occurred: %v", err)
	}

	certVal, _, err := types.ParseTLS(certFile.Name(), keyFile.Name())
	if err != nil {
		t.Fatalf("Failed to gen certs %v", err)
	}

	// We compare only Certificte because CertPool has pointers inside and it's hard to compare it
	if !reflect.DeepEqual(certVal, h.Scenario.Steps[0].Cert) {
		t.Errorf("\nExpected: %#v, \nFound: %#v", certVal, h.Scenario.Steps[0].Cert)
	}
}

func TestCreateHammerTLSWithOnlyCertPath(t *testing.T) {
	t.Parallel()

	// prepare TLS files
	cert, certKey := generateCerts()
	certFile, keyFile, err := createCertPairFiles(cert, certKey)
	if err != nil {
		t.Fatalf("Failed to prepare certs %v", err)
	}
	defer os.Remove(certFile.Name())
	defer os.Remove(keyFile.Name())

	config := buildJSONTLSConfig(certFile.Name(), "")

	jsonReader, _ := NewConfigReader(config, ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    types.DefaultIterCount,
		LoadType:          types.DefaultLoadType,
		TestDuration:      types.DefaultDuration,
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %#v, \nFound: %#v", expectedHammer, h)
	}
}

func TestCreateHammerTLSWithOnlyKeyPath(t *testing.T) {
	t.Parallel()

	// prepare TLS files
	cert, certKey := generateCerts()
	certFile, keyFile, err := createCertPairFiles(cert, certKey)
	if err != nil {
		t.Fatalf("Failed to prepare certs %v", err)
	}
	defer os.Remove(certFile.Name())
	defer os.Remove(keyFile.Name())

	config := buildJSONTLSConfig("", keyFile.Name())

	jsonReader, _ := NewConfigReader(config, ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    types.DefaultIterCount,
		LoadType:          types.DefaultLoadType,
		TestDuration:      types.DefaultDuration,
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %#v, \nFound: %#v", expectedHammer, h)
	}
}

func TestCreateHammerTLSWithWithEmptyPath(t *testing.T) {
	t.Parallel()

	config := buildJSONTLSConfig("", "")

	jsonReader, _ := NewConfigReader(config, ConfigTypeJson)
	expectedHammer := types.Hammer{
		IterationCount:    types.DefaultIterCount,
		LoadType:          types.DefaultLoadType,
		TestDuration:      types.DefaultDuration,
		ReportDestination: types.DefaultOutputType,
		Scenario: types.Scenario{
			Steps: []types.ScenarioStep{{
				ID:      1,
				URL:     "test.com",
				Method:  types.DefaultMethod,
				Timeout: types.DefaultTimeout,
			}},
		},
		Proxy: proxy.Proxy{
			Strategy: proxy.ProxyTypeSingle,
		},
		SamplingRate: types.DefaultSamplingCount,
		TestDataConf: make(map[string]types.CsvConf),
	}

	h, err := jsonReader.CreateHammer()

	if err != nil {
		t.Errorf("TestCreateHammerDefaultValues error occurred: %v", err)
	}

	if !reflect.DeepEqual(expectedHammer, h) {
		t.Errorf("\nExpected: %#v, \nFound: %#v", expectedHammer, h)
	}
}

func buildJSONTLSConfig(certPath, keyPath string) []byte {
	format := `
	{
		"steps": [
			{
				"id": 1,
				"url": "test.com",
				"cert_path": %q,
				"cert_key_path": %q
			}
		]
	}`

	config := fmt.Sprintf(format, certPath, keyPath)

	fmt.Println(config)

	return []byte(config)
}

func createCertPairFiles(cert string, certKey string) (*os.File, *os.File, error) {
	certFile, err := os.CreateTemp("", ".pem")
	if err != nil {
		return nil, nil, err
	}

	_, err = io.WriteString(certFile, cert)
	if err != nil {
		return nil, nil, err
	}

	keyFile, err := os.CreateTemp("", ".pem")
	if err != nil {
		return nil, nil, err
	}

	_, err = io.WriteString(keyFile, certKey)
	if err != nil {
		return nil, nil, err
	}

	return certFile, keyFile, nil
}

func generateCerts() (string, string) {
	cert := `-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUS4UhTks8aRCQ1k9IGn437ZyP3MgwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjEwMDUyMjM5MDVaFw0zMjEw
MDIyMjM5MDVaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDMbZctKXBx8v63TXIhM/OB7S6VfPqpzfHufhs6kAHu
jfC2ooCUqzqdg0T8bM1bjahYuAbQA1cWKYBsqfd01Po1ltWmbMf7ZvmSB6VN7kC2
Y670zee91dGDQ2yzmorJuIZAtOBVZesYLg8UHSGzSC/smJOrjYidtlbvzOcX0pv3
RCIUrNMed60EpSch/rzAJLzJmwNSQZ4vJHNlNetSkvTi7cxMWfwpcM/rN1hEmP1X
J43hJp/TNRZVnEsvs/yggP/FwUjG74mU3KfnWiv91AkkarNTNquEMJ+f4OFqMcnF
p0wqg47JTqcAAT0n1B0VB+z0hGXEFMN+IJXsHETZNG+JAgMBAAGjUzBRMB0GA1Ud
DgQWBBSIw+qUKQJjXWti5x/Cnn2GueuX5zAfBgNVHSMEGDAWgBSIw+qUKQJjXWti
5x/Cnn2GueuX5zAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAA
DXzf8VXi4s2GScNfHf0BzMjpyrtRZ0Wbp2Vfh7OwVR6xcx+pqXNjlydM/vu2LvOK
hh7Jbo+JS+o7O24UJ9lLFkCRsZVF+NFqJf+2rdHCaOiZSdZmtjBU0dFuAGS7+lU3
M8P7WCNOm6NAKbs7VZHVcZPzp81SCPQgQIS19xRf4Irbvsijv4YdyL4Qv7aWcclb
MdZX9AH9Fx8tJq4VKvUYsCXAD0kuywMLjh+yj5O/2hMvs5rvaQvm2daQNRDNp884
uTLrNF7W7QaKEL06ZpXJoBqdKsiwn577XTDKvzN0XxQrT+xV9VHO7OXblF+Od3/Y
SzBR+QiQKy3x+LkOxhkk
-----END CERTIFICATE-----`

	certKey := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDMbZctKXBx8v63
TXIhM/OB7S6VfPqpzfHufhs6kAHujfC2ooCUqzqdg0T8bM1bjahYuAbQA1cWKYBs
qfd01Po1ltWmbMf7ZvmSB6VN7kC2Y670zee91dGDQ2yzmorJuIZAtOBVZesYLg8U
HSGzSC/smJOrjYidtlbvzOcX0pv3RCIUrNMed60EpSch/rzAJLzJmwNSQZ4vJHNl
NetSkvTi7cxMWfwpcM/rN1hEmP1XJ43hJp/TNRZVnEsvs/yggP/FwUjG74mU3Kfn
Wiv91AkkarNTNquEMJ+f4OFqMcnFp0wqg47JTqcAAT0n1B0VB+z0hGXEFMN+IJXs
HETZNG+JAgMBAAECggEAM+U6NHfJmNPD/8qER5OFpJ0Ob1qL06F5Yj7XMLWwF9wm
mGaGV7dkKOpTD/Wa6Dv82ZDWAeZnLDQa6vr228zZO9Nvp1EEL3kDsCOKvk7WVLbX
ikPfKZznE/iA1tNLmkvioPiJ3oQB+2Bt6YA/tuCDcf+FtU43uTm5tiSBIdYQS+Om
xN9OEXihk1svxHXQKa/a3nKPVLvdp3P90hDJ0PcRslXSy1V8az+A94JFEnCvnKsK
nF2rItCcXkInL0lYHZKgLHQMXGWkNl8e3PA1GZk3yF6LPNtPI1T5Ek9GwkHNw4JZ
BL/xEWLKB1qR2Z4I3UbWGVyi418kANv1eISb+49egQKBgQDraSRWB8nM5O3Zl9kT
8S5K924o1oXrO17eqQlVtQVmtUdoVvIBc6uHQZOmV1eHYpr6c95h8apNLexI22AY
SWkq9smpCnxLUsdkplwzie0F4bAzD6MCR8WIJxapUSPlyCA+8st1hquYBchKGQhd
6mMY1gzMDacYV/WhtG4E5d0nMQKBgQDeTr793n00VtpKuquFJe6Stu7Ujf64dL0s
3opLovyI0TmtMz5oCqIezwrjqc0Vy0UksWXaz0AboinDP+5n60cTEIt/6H0kryDc
dxfSHEA9BBDoQtxOFi3QGcxXbwu0i9QSoexrKY7FhA2xPji6bCcPycthhIrCpUiZ
s5gVkjHn2QKBgQCGklxLMbiSgGvXb46Qb9be1AMNJVT427+n2UmUzR6BUC+53boK
Sm1LrJkTBerrYdrmQUZnBxcrd40TORT9zTlpbhppn6zeAjwptVAPxlDQg+uNxOqS
ayToaC/0KoYy3OxSD8lvLcT56pRMh3LY/RwZHoPCQiu7Js0r21DpS93YgQKBgAuc
c09RMprsOmSS0WiX7ZkOIvVJIVfDCSpxySlgLu56dxe7yHOosoUHbVsswEB2KHtd
JKPEFWYcFzBSg4I8AK9XOuIIY5jp6L57Hexke1p0fumSrG0LrYLkBg8/Bo58iywZ
9v414nYgipKKXG4oPfYOJShHwvOdrGgSwEvIIgEpAoGAZz0yC9+x+JaoTnyUIRyI
+Aj5a4KhYjFtsZhcn/yCZHDqzJNDz6gAu579ey+J2CVOhjtgB5lowsDrHu32Hqnn
SEfyTru/ynQ8obwaRzdDYml+On86YWOw+brpMXkN+KB6bs2okE2N68v0qGPakxjt
OLDW6kKz5pI4T8lQJhdqjCU=
-----END PRIVATE KEY-----`

	return cert, certKey
}
