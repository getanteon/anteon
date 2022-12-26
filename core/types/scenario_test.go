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
		Headers: map[string]string{
			"{{ARGENTINA}}": "{{ARGENTINA}}",
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
		Headers:       map[string]string{},
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
		Headers:       map[string]string{},
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
	headerKey := "headerKey"
	st := ScenarioStep{
		ID:       22,
		Name:     "",
		Method:   http.MethodGet,
		Auth:     Auth{},
		Cert:     tls.Certificate{},
		CertPool: &x509.CertPool{},
		Headers:  map[string]string{},
		Payload:  "",
		URL:      url,
		Timeout:  0,
		Sleep:    "",
		Custom:   map[string]interface{}{},
		EnvsToCapture: []EnvCaptureConf{{
			Name: "FromHeader",
			Key:  &headerKey,
		},
		},
	}

	definedEnvs := map[string]struct{}{}
	err := st.validate(definedEnvs)

	var captureConfigError CaptureConfigError

	if !errors.As(err, &captureConfigError) {
		t.Errorf("Should be CaptureConfigError")
	}

	t.Logf("%v", captureConfigError)
}
