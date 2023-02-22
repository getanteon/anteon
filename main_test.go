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

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/types"
)

var tempRun func(h types.Hammer)

func TestMain(m *testing.M) {
	// Mock run function to prevent engine starting
	tempRun = run
	run = func(h types.Hammer) {}
	os.Exit(m.Run())
}

func resetFlags() {
	*iterCount = types.DefaultIterCount
	*loadType = types.DefaultLoadType
	*duration = types.DefaultDuration

	*method = types.DefaultMethod
	*payload = ""
	*auth = ""
	headers = header{}

	*target = ""
	*timeout = types.DefaultTimeout

	*proxyFlag = ""
	*output = types.DefaultOutputType

	*configPath = ""

	*certPath = ""
	*certKeyPath = ""
}

func TestDefaultFlagValues(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-t=example.com"}

	flag.Parse()

	if *iterCount != types.DefaultIterCount {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultIterCount, *iterCount)
	}
	if *loadType != types.DefaultLoadType {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultLoadType, *loadType)
	}
	if *duration != types.DefaultDuration {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultDuration, *duration)
	}
	if *method != types.DefaultMethod {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultMethod, *method)
	}
	if *payload != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *payload)
	}
	if *auth != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *auth)
	}
	if reflect.DeepEqual(headers, []string{}) {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", []string{}, headers)
	}
	if *timeout != types.DefaultTimeout {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultTimeout, *timeout)
	}
	if *proxyFlag != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *proxyFlag)
	}
	if *output != types.DefaultOutputType {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", types.DefaultOutputType, *output)
	}
	if *configPath != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *configPath)
	}
	if *certPath != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *certPath)
	}
	if *certKeyPath != "" {
		t.Errorf("TestDefaultFlagValues failed, expected %#v, found %#v", "", *certKeyPath)
	}
}

func TestCreateHammer(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		fromFlags bool
		fromFile  bool
	}{
		{"Flag", "-t=dummy.com -config=", true, false},
		{"File", "-config=dummy.json -t=", false, true},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			oldFileFunc := createHammerFromConfigFile
			oldFlagFunc := createHammerFromFlags
			defer func() {
				os.Args = oldArgs
				createHammerFromConfigFile = oldFileFunc
				createHammerFromFlags = oldFlagFunc
			}()

			fromFileCalled := false
			fromFlagsCalled := false
			createHammerFromConfigFile = func(debug bool) (h types.Hammer, err error) {
				fromFileCalled = true
				return
			}
			createHammerFromFlags = func() (h types.Hammer, err error) {
				fromFlagsCalled = true
				return
			}

			// Act
			os.Args = []string{"cmd", test.args}
			flag.Parse()
			createHammer()

			// Assert
			if fromFileCalled != test.fromFile {
				t.Errorf("createHammerFromConfigFileCalled expected %v found %v", test.fromFile, fromFileCalled)
			}
			if fromFlagsCalled != test.fromFlags {
				t.Errorf("createHammerFromFlagsCalled expected %v found %v", test.fromFlags, fromFlagsCalled)
			}
		}

		t.Run(test.name, tf)
	}
}

func TestDebugFlagOverridesConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"DebugFlagShouldOverrideConfig",
			[]string{"-config", "config/config_testdata/config_debug_false.json", "-debug"}},
		{"UseConfigDebugKeyWhenNoDebugFlagSpecified",
			[]string{"-config", "config/config_testdata/config_debug_mode.json", "-debug", "false"}},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			// Act
			os.Args = append([]string{"cmd"}, test.args...)
			flag.Parse()
			h, err := createHammer()

			if err != nil {
				t.Errorf("createHammer return %v", err)
			}

			// Assert
			if h.Debug != *debug {
				t.Errorf("debug flag did not override config file")
			}

		}

		t.Run(test.name, tf)
	}
}

func TestCreateScenario(t *testing.T) {
	url := "https://test.com"
	valid := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     url,
				Timeout: types.DefaultTimeout,
				Headers: map[string][]string{},
			},
		},
	}
	validWithAuth := types.Scenario{
		Steps: []types.ScenarioStep{
			{
				ID:      1,
				Method:  types.DefaultMethod,
				URL:     url,
				Timeout: types.DefaultTimeout,
				Headers: map[string][]string{},
				Auth: types.Auth{
					Type:     types.AuthHttpBasic,
					Username: "testuser",
					Password: "pass",
				},
			},
		},
	}

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
		expected  types.Scenario
	}{
		{"InvalidAuth", []string{"-t=https://test.com", "-a=no_pass_included"}, true, types.Scenario{}},
		{"InvalidTarget", []string{"-t=asds.x.x.x"}, true, types.Scenario{}},
		{"Valid", []string{"-t=https://test.com"}, false, valid},
		{"ValidWithAuth", []string{"-t=https://test.com", "-a=testuser:pass"}, false, validWithAuth},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = append([]string{"cmd"}, test.args...)

			// Act
			flag.Parse()
			s, err := createScenario()

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if !reflect.DeepEqual(test.expected, s) {
					t.Errorf("Expected %#v, Found %#v", test.expected, s)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestCreateScenarioTLS(t *testing.T) {
	// prepare TLS files
	cert, certKey := generateCerts()
	certFile, keyFile, err := createCertPairFiles(cert, certKey)
	if err != nil {
		t.Fatalf("Failed to prepare certs %v", err)
	}
	defer os.Remove(certFile.Name())
	defer os.Remove(keyFile.Name())

	certVal, _, err := types.ParseTLS(certFile.Name(), keyFile.Name())
	if err != nil {
		t.Fatalf("Failed to gen certs %v", err)
	}

	certPathArg := fmt.Sprintf("--cert_path=%s", certFile.Name())
	keyPathArg := fmt.Sprintf("--cert_key_path=%s", keyFile.Name())

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
		expected  tls.Certificate
	}{
		{"MissingKey", []string{"-t=https://test.com", certPathArg}, false, tls.Certificate{}},
		{"MissingCert", []string{"-t=https://test.com", keyPathArg}, false, tls.Certificate{}},
		{"WithTLS", []string{"-t=https://test.com", certPathArg, keyPathArg}, false, certVal},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = append([]string{"cmd"}, test.args...)

			// Act
			flag.Parse()
			s, err := createScenario()

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if !reflect.DeepEqual(test.expected, s.Steps[0].Cert) {
					t.Errorf("Expected %v, Found %v", test.expected, s)
				}
			}
		}

		t.Run(test.name, tf)
	}
}

func TestCreateProxy(t *testing.T) {
	addr, _ := url.Parse("http://127.0.0.1:80")
	withAddr := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     addr,
	}
	withoutAddr := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     nil,
	}

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
		expected  proxy.Proxy
	}{
		{"InvalidProxy", []string{"-t=https://test.com", "-P=127.0.0.1:09"}, true, proxy.Proxy{}},
		{"ValidWithAddr", []string{"-t=https://test.com", "-P=http://127.0.0.1:80"}, false, withAddr},
		{"ValidWithoutAddr", []string{"-t=https://test.com"}, false, withoutAddr},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			resetFlags()
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = append([]string{"cmd"}, test.args...)

			// Act
			flag.Parse()
			p, err := createProxy()

			// Assert
			t.Log(test.args)
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if test.expected.Strategy != p.Strategy {
					t.Errorf("Expected Strategy %v, Found %v", test.expected.Strategy, p.Strategy)
				}
				if (test.expected.Addr != nil && *test.expected.Addr != *p.Addr) || (test.expected.Addr == nil && p.Addr != nil) {
					t.Errorf("Expected Addr %v, Found %v", test.expected.Addr, p.Addr)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestParseHeaders(t *testing.T) {
	validSingleHeader := map[string][]string{"header": {"value"}}
	validMultiHeader := map[string][]string{"header-1": {"value-1"}, "header-2": {"value-2"}}

	invalidHeader := header{}
	invalidHeader.Set("invalid|header?: value-1")

	tests := []struct {
		name      string
		args      header
		shouldErr bool
		expected  map[string][]string
	}{
		{"ValidSingleHeder", []string{"header: value"}, false, validSingleHeader},
		{"ValidMultiHeader", []string{"header-1: value-1", "header-2: value-2"}, false, validMultiHeader},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			headers := header{}
			for _, h := range test.args {
				headers.Set(h)
			}

			// Arrange
			h, err := parseHeaders(headers)

			// Assert
			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}
				if !reflect.DeepEqual(test.expected, h) {
					t.Errorf("Expected  %#v, Found %#v", test.expected, h)
				}
			}

		}

		t.Run(test.name, tf)
	}
}

func TestRun(t *testing.T) {
	// Arrange
	resetFlags()
	runCalled := false
	run = func(h types.Hammer) {
		runCalled = true
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Act
	os.Args = []string{"cmd", "-t=test.com"}
	main()

	// Assert
	if !runCalled {
		t.Errorf("Run should be called")
	}
}

func TestTargetEmpty(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "asd"}
		main()
		return
	}

	// Since we reexecute the test here, this test case doesnt' increment the coverage.
	cmd := exec.Command(os.Args[0], "-test.run=TestTargetEmpty")
	cmd.Env = append(os.Environ(), "TARGET_EMPTY=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	t.Errorf("TestTargetEmpty should be failed with exit code 1 found: %v", err)
}

func TestTargetInvalidHammer(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("TARGET_EMPTY") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "-t=dummy.com -l invalidLoadType"}
		main()
		return
	}

	// Since we reexecute the test here, this test case doesnt' increment the coverage.
	cmd := exec.Command(os.Args[0], "-test.run=TestTargetEmpty")
	cmd.Env = append(os.Environ(), "TARGET_EMPTY=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}

	t.Errorf("TestTargetEmpty should be failed with exit code 1 found: %v", err)
}

func TestVersion(t *testing.T) {
	// Below cmd code triggers this block
	if os.Getenv("VERSION") == "1" {
		resetFlags()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"cmd", "-version"}
		main()
		return
	}

	// Since we reexecute the test here, this test case doesnt' increment the coverage.
	cmd := exec.Command(os.Args[0], "-test.run=TestVersion")
	cmd.Env = append(os.Environ(), "VERSION=1")
	err := cmd.Run()

	if err == nil {
		return
	}

	t.Errorf("TestVersion should not be failed")
}

func Test_versionTemplate(t *testing.T) {
	GitVersion = "v0.0.2"
	GitCommit = "akjsghsajghas"
	BuildDate = "2021-10-03T15:16:52Z"
	tests := []struct {
		name string
		want string
	}{
		{name: "version", want: "Version:        v0.0.2\nGit commit:     akjsghsajghas\nBuilt           2021-10-03T15:16:52Z\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionTemplate(); !strings.Contains(got, tt.want) {
				t.Errorf("versionTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
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
