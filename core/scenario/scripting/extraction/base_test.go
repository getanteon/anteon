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
		JsonPath: new(string),
		Xpath:    new(string),
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
		JsonPath: new(string),
		Xpath:    new(string),
		RegExp:   &types.RegexCaptureConf{},
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
