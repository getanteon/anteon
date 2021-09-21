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
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"ddosify.com/hammer/core/types"
	"golang.org/x/net/http2"
)

func TestInit(t *testing.T) {
	s := types.ScenarioItem{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
	}
	p, _ := url.Parse("https://127.0.0.1:80")
	ctx := context.TODO()

	h := &httpRequester{}
	h.Init(s, p, ctx)

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

func TestClient(t *testing.T) {
	p, _ := url.Parse("https://127.0.0.1:80")
	ctx := context.TODO()

	// Basic Client
	s := types.ScenarioItem{
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
		DisableKeepAlives: true,
	}
	expectedClient := &http.Client{
		Transport: expectedTr,
		Timeout:   time.Duration(types.DefaultTimeout) * time.Second,
	}

	// Client with custom data
	sWithCustomData := types.ScenarioItem{
		ID:       1,
		Protocol: types.ProtocolHTTPS,
		Method:   http.MethodGet,
		URL:      "https://test.com",
		Timeout:  types.DefaultTimeout,
		Custom: map[string]interface{}{
			"disableRedirect":    true,
			"keepAlive":          true,
			"disableCompression": true,
			"hostName":           "dummy.com",
		},
	}
	expectedTLSCustomData := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "dummy.com",
	}
	expectedTrCustomData := &http.Transport{
		TLSClientConfig:    expectedTLSCustomData,
		Proxy:              http.ProxyURL(p),
		DisableKeepAlives:  false,
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
	sHTTP2 := types.ScenarioItem{
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
		DisableKeepAlives: true,
	}
	http2.ConfigureTransport(expectedTrHTTP2)
	expectedClientHTTP2 := &http.Client{
		Transport: expectedTrHTTP2,
		Timeout:   time.Duration(types.DefaultTimeout) * time.Second,
	}

	// Sub Tests
	tests := []struct {
		name         string
		scenarioItem types.ScenarioItem
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
			h := &httpRequester{}
			h.Init(test.scenarioItem, test.proxy, test.ctx)

			transport := h.client.Transport.(*http.Transport)
			tls := transport.TLSClientConfig

			// TLS Assert (Also check HTTP2 vs HTTP)
			if !reflect.DeepEqual(test.tls, tls) {
				t.Errorf("\nTLS Expected %#v, \nFound %#v", test.tls, tls)
			}

			// Transport Assert

			// Compare HTTP2 configured transport vs HTTP transport
			if reflect.TypeOf(test.transport) != reflect.TypeOf(transport) {
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

// func TestRequest(t *testing.T) {
// 	// Basic request
// 	s := types.ScenarioItem{
// 		ID:       1,
// 		Protocol: types.ProtocolHTTPS,
// 		Method:   http.MethodGet,
// 		URL:      "https://test.com",
// 		Payload:  "payloadtest",
// 	}
// 	expectedRequest := &http.Request{
// 		Method: s.Method,
// 		Proto:  s.Protocol,
// 		Body:   bytes.NewBufferString(s.Payload),
// 	}
// }
