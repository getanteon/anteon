package extraction

import (
	"errors"
	"net/http"
	"runtime"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

func TestHttpHeaderKey_NotSpecified(t *testing.T) {
	ce := types.EnvCaptureConf{
		JsonPath: nil,
		Xpath:    nil,
		RegExp:   &types.RegexCaptureConf{},
		Name:     "",
		From:     types.Header,
		Key:      nil,
	}

	_, err := Extract(http.Header{}, ce)

	if err == nil {
		t.Errorf("Expected error when header key not specified")
	}
}

func TestExtract_TypeAssertErrorRecover(t *testing.T) {
	headerKey := "x"
	ce := types.EnvCaptureConf{
		JsonPath: nil,
		Xpath:    nil,
		RegExp:   nil,
		Name:     "",
		From:     types.Header,
		Key:      &headerKey,
	}

	// source should be http.Header
	_, err := Extract("sdfds", ce)

	var assertError *runtime.TypeAssertionError
	if !errors.As(err, &assertError) {
		t.Errorf("Expected error must be TypeAssertionError, got %v", err)
	}
}

func TestExtract_NilSource(t *testing.T) {
	headerKey := "x"
	ce := types.EnvCaptureConf{
		JsonPath: nil,
		Xpath:    nil,
		RegExp:   nil,
		Name:     "",
		From:     types.Header,
		Key:      &headerKey,
	}

	_, err := Extract(nil, ce)

	if err == nil {
		t.Errorf("error expected, got nil")
	}
}

func TestExtract_InvalidXml(t *testing.T) {
	xpath := ""
	ce := types.EnvCaptureConf{
		JsonPath: nil,
		Xpath:    &xpath,
		RegExp:   nil,
		Name:     "",
		From:     types.Body,
		Key:      nil,
	}

	_, err := Extract([]byte("xxx"), ce)

	if err == nil {
		t.Errorf("error expected, got nil")
	}
}

func TestCookieName_NotSpecified(t *testing.T) {
	ce := types.EnvCaptureConf{
		JsonPath:   nil,
		Xpath:      nil,
		RegExp:     &types.RegexCaptureConf{},
		Name:       "",
		From:       types.Cookie,
		Key:        nil,
		CookieName: nil,
	}

	_, err := Extract(map[string]*http.Cookie{}, ce)

	if err == nil {
		t.Errorf("Expected error when cookie key not specified")
	}
}
