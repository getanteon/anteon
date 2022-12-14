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

package requester

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"testing"
	"time"

	"go.ddosify.com/ddosify/core/types"
	"golang.org/x/net/http2"
)

func TestInit(t *testing.T) {
	s := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
	}
	p, _ := url.Parse("https://127.0.0.1:80")
	ctx := context.TODO()

	h := &HttpRequester{}
	h.Init(ctx, s, p, false)

	if !reflect.DeepEqual(h.packet, s) {
		t.Errorf("Expected %v, Found %v", s, h.packet)
	}
	if !reflect.DeepEqual(h.proxyAddr, p) {
		t.Errorf("Expected %v, Found %v", p, h.proxyAddr)
	}
	if !reflect.DeepEqual(h.ctx, ctx) {
		t.Errorf("Expected %v, Found %v", ctx, h.ctx)
	}
}

func TestInitClient(t *testing.T) {
	p, _ := url.Parse("https://127.0.0.1:80")
	ctx := context.TODO()

	// Basic Client
	s := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
	}
	expectedTLS := &tls.Config{
		InsecureSkipVerify: true,
	}
	expectedTr := &http.Transport{
		TLSClientConfig:   expectedTLS,
		Proxy:             http.ProxyURL(p),
		DisableKeepAlives: false,
	}
	expectedClient := &http.Client{
		Transport: expectedTr,
		Timeout:   time.Duration(types.DefaultTimeout) * time.Second,
	}

	// Client with custom data
	sWithCustomData := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
		Custom: map[string]interface{}{
			"disable-redirect":    true,
			"keep-alive":          false,
			"disable-compression": true,
			"hostname":            "dummy.com",
		},
	}
	expectedTLSCustomData := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "dummy.com",
	}
	expectedTrCustomData := &http.Transport{
		TLSClientConfig:    expectedTLSCustomData,
		Proxy:              http.ProxyURL(p),
		DisableKeepAlives:  true,
		DisableCompression: true,
	}
	expectedClientWithCustomData := &http.Client{
		Transport: expectedTrCustomData,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Duration(types.DefaultTimeout) * time.Second,
	}

	// H2 Client
	sHTTP2 := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
		Custom: map[string]interface{}{
			"h2": true,
		},
	}
	expectedTLSHTTP2 := &tls.Config{
		InsecureSkipVerify: true,
	}
	expectedTrHTTP2 := &http.Transport{
		TLSClientConfig:   expectedTLSHTTP2,
		Proxy:             http.ProxyURL(p),
		DisableKeepAlives: false,
	}
	http2.ConfigureTransport(expectedTrHTTP2)
	expectedClientHTTP2 := &http.Client{
		Transport: expectedTrHTTP2,
		Timeout:   time.Duration(types.DefaultTimeout) * time.Second,
	}

	// Sub Tests
	tests := []struct {
		name         string
		scenarioItem types.ScenarioStep
		proxy        *url.URL
		ctx          context.Context
		tls          *tls.Config
		transport    *http.Transport
		client       *http.Client
	}{
		{"Basic", s, p, ctx, expectedTLS, expectedTr, expectedClient},
		{"Custom", sWithCustomData, p, ctx, expectedTLSCustomData, expectedTrCustomData, expectedClientWithCustomData},
		{"HTTP2", sHTTP2, p, ctx, expectedTLSHTTP2, expectedTrHTTP2, expectedClientHTTP2},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			h := &HttpRequester{}
			h.Init(test.ctx, test.scenarioItem, test.proxy, false)

			transport := h.client.Transport.(*http.Transport)
			tls := transport.TLSClientConfig

			// TLS Assert (Also check HTTP2 vs HTTP)
			if !reflect.DeepEqual(test.tls, tls) {
				t.Errorf("\nTLS Expected %#v, \nFound %#v", test.tls, tls)
			}

			// Transport Assert
			if reflect.TypeOf(test.transport) != reflect.TypeOf(transport) {
				// Compare HTTP2 configured transport vs HTTP transport
				t.Errorf("Transport Type Expected %#v, Found %#v", test.transport, transport)
			}

			pFunc := transport.Proxy == nil
			expectedPFunc := test.transport.Proxy == nil
			if pFunc != expectedPFunc {
				t.Errorf("Proxy Expected %v, Found %v", expectedPFunc, pFunc)
			}
			if test.transport.DisableKeepAlives != transport.DisableKeepAlives {
				t.Errorf("DisableKeepAlives Expected %v, Found %v", test.transport.DisableKeepAlives, transport.DisableKeepAlives)
			}
			if test.transport.DisableCompression != transport.DisableCompression {
				t.Errorf("DisableCompression Expected %v, Found %v",
					test.transport.DisableCompression, transport.DisableCompression)
			}

			// Client Assert
			if test.client.Timeout != h.client.Timeout {
				t.Errorf("Timeout Expected %v, Found %v", test.client.Timeout, h.client.Timeout)
			}

			crFunc := h.client.CheckRedirect == nil
			expectedCRFunc := test.client.CheckRedirect == nil
			if expectedCRFunc != crFunc {
				t.Errorf("CheckRedirect Expected %v, Found %v", expectedCRFunc, crFunc)
			}

		}
		t.Run(test.name, tf)
	}
}

func TestInitRequest(t *testing.T) {
	p, _ := url.Parse("https://127.0.0.1:80")
	ctx := context.TODO()

	// Invalid request
	sInvalid := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   ":31:31:#",
		URL:      "https://test.com",
		Payload:  "payloadtest",
	}

	// Basic request
	s := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Payload:  "payloadtest",
	}
	expected, _ := http.NewRequest(s.Method, s.URL, bytes.NewBufferString(s.Payload))
	expected.Close = false
	expected.Header = make(http.Header)

	// Request with auth
	sWithAuth := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Payload:  "payloadtest",
		Auth: types.Auth{
			Username: "test",
			Password: "123",
		},
	}
	expectedWithAuth, _ := http.NewRequest(sWithAuth.Method, sWithAuth.URL, bytes.NewBufferString(sWithAuth.Payload))
	expectedWithAuth.Close = false
	expectedWithAuth.Header = make(http.Header)
	expectedWithAuth.SetBasicAuth(sWithAuth.Auth.Username, sWithAuth.Auth.Password)

	// Request With Headers
	sWithHeaders := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.localhost",
		Payload:  "payloadtest",
		Auth: types.Auth{
			Username: "test",
			Password: "123",
		},
		Headers: map[string]string{
			"Header1":    "Value1",
			"Header2":    "Value2",
			"User-Agent": "Firefox",
			"Host":       "test.com",
		},
	}
	expectedWithHeaders, _ := http.NewRequest(sWithHeaders.Method,
		sWithHeaders.URL, bytes.NewBufferString(sWithHeaders.Payload))
	expectedWithHeaders.Close = false
	expectedWithHeaders.Header = make(http.Header)
	expectedWithHeaders.Header.Set("Header1", "Value1")
	expectedWithHeaders.Header.Set("Header2", "Value2")
	expectedWithHeaders.Header.Set("User-Agent", "Firefox")
	expectedWithHeaders.Host = "test.com"
	expectedWithHeaders.SetBasicAuth(sWithHeaders.Auth.Username, sWithHeaders.Auth.Password)

	// Request keep-alive condition
	sWithoutKeepAlive := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Payload:  "payloadtest",
		Auth: types.Auth{
			Username: "test",
			Password: "123",
		},
		Headers: map[string]string{
			"Header1": "Value1",
			"Header2": "Value2",
		},
		Custom: map[string]interface{}{
			"keep-alive": false,
		},
	}
	expectedWithoutKeepAlive, _ := http.NewRequest(sWithoutKeepAlive.Method,
		sWithoutKeepAlive.URL, bytes.NewBufferString(sWithoutKeepAlive.Payload))
	expectedWithoutKeepAlive.Close = true
	expectedWithoutKeepAlive.Header = make(http.Header)
	expectedWithoutKeepAlive.Header.Set("Header1", "Value1")
	expectedWithoutKeepAlive.Header.Set("Header2", "Value2")
	expectedWithoutKeepAlive.SetBasicAuth(sWithoutKeepAlive.Auth.Username, sWithoutKeepAlive.Auth.Password)

	// Sub Tests
	tests := []struct {
		name         string
		scenarioItem types.ScenarioStep
		shouldErr    bool
		request      *http.Request
	}{
		{"Invalid", sInvalid, true, nil},
		{"Basic", s, false, expected},
		{"WithAuth", sWithAuth, false, expectedWithAuth},
		{"WithHeaders", sWithHeaders, false, expectedWithHeaders},
		{"WithoutKeepAlive", sWithoutKeepAlive, false, expectedWithoutKeepAlive},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			h := &HttpRequester{}
			err := h.Init(ctx, test.scenarioItem, p, false)

			if test.shouldErr {
				if err == nil {
					t.Errorf("Should be errored")
				}
			} else {
				if err != nil {
					t.Errorf("Errored: %v", err)
				}

				if !reflect.DeepEqual(h.request.URL, test.request.URL) {
					t.Errorf("URL Expected: %#v, Found: \n%#v", test.request.URL, h.request.URL)
				}
				if !reflect.DeepEqual(h.request.Host, test.request.Host) {
					t.Errorf("Host Expected: %#v, Found: \n%#v", test.request.Host, h.request.Host)
				}
				if !reflect.DeepEqual(h.request.Body, test.request.Body) {
					t.Errorf("Body Expected: %#v, Found: \n%#v", test.request.Body, h.request.Body)
				}
				if !reflect.DeepEqual(h.request.Header, test.request.Header) {
					t.Errorf("Header Expected: %#v, Found: \n%#v", test.request.Header, h.request.Header)
				}
				if !reflect.DeepEqual(h.request.Close, test.request.Close) {
					t.Errorf("Close Expected: %#v, Found: \n%#v", test.request.Close, h.request.Close)
				}
			}
		}
		t.Run(test.name, tf)
	}
}

func TestSendOnDebugModePopulatesDebugInfo(t *testing.T) {
	ctx := context.TODO()
	// Basic request
	payload := "reqbodypayload"
	s := types.ScenarioStep{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://ddosify.com",
		Payload:  payload,
		Headers:  map[string]string{"X": "y"},
	}

	expectedDebugInfo := map[string]interface{}{
		"url":            "https://ddosify.com",
		"method":         http.MethodGet,
		"requestHeaders": http.Header{"X": {"y"}},
		"requestBody":    []byte(payload),
		// did not fill below
		"responseBody":    []byte{},
		"responseHeaders": map[string][]string{},
	}

	// Sub Tests
	tests := []struct {
		name              string
		scenarioStep      types.ScenarioStep
		expectedDebugInfo map[string]interface{}
	}{
		{"Basic", s, expectedDebugInfo},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			h := &HttpRequester{}
			debug := true
			var proxy *url.URL
			_ = h.Init(ctx, test.scenarioStep, proxy, debug)
			res := h.Send()

			if len(res.DebugInfo) == 0 {
				t.Errorf("debugInfo should have been populated on debug mode")
			}

			if test.expectedDebugInfo["method"] != res.DebugInfo["method"] {
				t.Errorf("Method Expected %#v, Found: \n%#v", test.expectedDebugInfo["method"], res.DebugInfo["method"])
			}
			if test.expectedDebugInfo["url"] != res.DebugInfo["url"] {
				t.Errorf("Url Expected %#v, Found: \n%#v", test.expectedDebugInfo["url"], res.DebugInfo["url"])
			}
			if !bytes.Equal(test.expectedDebugInfo["requestBody"].([]byte), res.DebugInfo["requestBody"].([]byte)) {
				t.Errorf("RequestBody Expected %#v, Found: \n%#v", test.expectedDebugInfo["requestBody"],
					res.DebugInfo["requestBody"])
			}
			if !reflect.DeepEqual(test.expectedDebugInfo["requestHeaders"], res.DebugInfo["requestHeaders"]) {
				t.Errorf("RequestHeaders Expected %#v, Found: \n%#v", test.expectedDebugInfo["requestHeaders"],
					res.DebugInfo["requestHeaders"])
			}

		}
		t.Run(test.name, tf)
	}
}

func TestDynamicVariableRegex(t *testing.T) {
	// Sub Tests
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
	}{
		{"Match1", "https://example.com/{{_abc}}", true},
		{"Match2", "https://example.com/{{_timestamp}}", true},
		{"Match3", "https://example.com/aaa/{{_timestamp}}", true},
		{"Match4", "https://example.com/aaa/{{_timestamp}}/bbb", true},
		{"Match5", "https://example.com/{{_timestamp}}/{_abc}", true},
		{"Match6", "https://example.com/{{_abc/{{_timestamp}}", true},
		{"Match7", "https://example.com/_aaa/{{_timestamp}}", true},

		{"Not Match1", "https://example.com/{{_abc", false},
		{"Not Match2", "https://example.com/{{_abc}", false},
		{"Not Match3", "https://example.com/_abc", false},
		{"Not Match4", "https://example.com/{{abc", false},
		{"Not Match5", "https://example.com/abc", false},
		{"Not Match6", "https://example.com/abc/{{cc}}", false},
		{"Not Match7", "https://example.com/abc/{{cc}}/fcf", false},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			re := regexp.MustCompile(DynamicVariableRegex)
			matched := re.MatchString(test.url)

			if test.shouldMatch != matched {
				t.Errorf("Name: %s, ShouldMatch: %t, Matched: %t\n", test.name, test.shouldMatch, matched)
			}

		}
		t.Run(test.name, tf)
	}
}
