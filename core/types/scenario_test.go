package types

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"testing"
)

func TestScenarioStepValid_EnvVariableInHeader(t *testing.T) {
	url := "https://test.com"
	st := ScenarioStep{
		ID:       22,
		Name:     "",
		Method:   http.MethodGet,
		Auth:     Auth{},
		Cert:     tls.Certificate{},
		CertPool: &x509.CertPool{},
		Headers: map[string][]string{
			"{{ARGENTINA}}": {"{{ARGENTINA}}"},
		},
		Payload:       "",
		URL:           url,
		Timeout:       0,
		Sleep:         "",
		Custom:        map[string]interface{}{},
		EnvsToCapture: []EnvCaptureConf{},
	}

	definedEnvs := map[string]struct{}{}
	err := st.validate(definedEnvs)

	var environmentNotDefined EnvironmentNotDefinedError

	if !errors.As(err, &environmentNotDefined) {
		t.Errorf("Should be EnvironmentNotDefinedError")
	}

	t.Logf("%v", environmentNotDefined)
}

func TestScenarioStepValid_EnvVariableInPayload(t *testing.T) {
	url := "https://test.com"
	st := ScenarioStep{
		ID:            22,
		Name:          "",
		Method:        http.MethodGet,
		Auth:          Auth{},
		Cert:          tls.Certificate{},
		CertPool:      &x509.CertPool{},
		Headers:       map[string][]string{},
		Payload:       "{{ARGENTINA}}",
		URL:           url,
		Timeout:       0,
		Sleep:         "",
		Custom:        map[string]interface{}{},
		EnvsToCapture: []EnvCaptureConf{},
	}

	definedEnvs := map[string]struct{}{}
	err := st.validate(definedEnvs)

	var environmentNotDefined EnvironmentNotDefinedError

	if !errors.As(err, &environmentNotDefined) {
		t.Errorf("Should be EnvironmentNotDefinedError")
	}

	t.Logf("%v", environmentNotDefined)
}

func TestScenarioStepValid_EnvVariableInURL(t *testing.T) {
	url := "https://test.com/{{ARGENTINA}}"
	st := ScenarioStep{
		ID:            22,
		Name:          "",
		Method:        http.MethodGet,
		Auth:          Auth{},
		Cert:          tls.Certificate{},
		CertPool:      &x509.CertPool{},
		Headers:       map[string][]string{},
		Payload:       "",
		URL:           url,
		Timeout:       0,
		Sleep:         "",
		Custom:        map[string]interface{}{},
		EnvsToCapture: []EnvCaptureConf{},
	}

	definedEnvs := map[string]struct{}{}
	err := st.validate(definedEnvs)

	var environmentNotDefined EnvironmentNotDefinedError

	if !errors.As(err, &environmentNotDefined) {
		t.Errorf("Should be EnvironmentNotDefinedError")
	}

	t.Logf("%v", environmentNotDefined)
}

func TestScenarioStep_InvalidCaptureConfig(t *testing.T) {
	url := "https://test.com"

	stEmptyFromField := ScenarioStep{
		ID:     22,
		Name:   "",
		Method: http.MethodGet,
		URL:    url,
		EnvsToCapture: []EnvCaptureConf{{
			Name: "FromHeader",
			From: "",
		}},
	}

	stNoHeaderKey := ScenarioStep{
		ID:     22,
		Name:   "",
		Method: http.MethodGet,
		URL:    url,
		EnvsToCapture: []EnvCaptureConf{{
			Name: "FromHeader",
			From: SourceType(Header),
		}},
	}

	stNoBodySpecifierKey := ScenarioStep{
		ID:     22,
		Name:   "",
		Method: http.MethodGet,
		URL:    url,
		EnvsToCapture: []EnvCaptureConf{{
			Name: "FromBody",
			From: SourceType(Body),
		}},
	}

	definedEnvs := map[string]struct{}{}

	tests := []struct {
		name string
		st   ScenarioStep
	}{
		{"NoHeaderKey", stNoHeaderKey},
		{"NoBodySpecifierKey", stNoBodySpecifierKey},
		{"EmptyFromField", stEmptyFromField},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			// Arrange
			err := test.st.validate(definedEnvs)

			var captureConfigError CaptureConfigError

			if !errors.As(err, &captureConfigError) {
				t.Errorf("Should be CaptureConfigError")
			}
		}

		t.Run(test.name, tf)
	}
}
