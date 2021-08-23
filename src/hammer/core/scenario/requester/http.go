package requester

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ddosify.com/hammer/core/types"
	"github.com/google/uuid"
	"golang.org/x/net/http2"
)

type httpRequester struct {
	packet  types.ScenarioItem
	client  *http.Client
	request *http.Request
}

// Create a client with scenarioItem and use same client for each request
func (h *httpRequester) Init(s types.ScenarioItem) (err error) {
	h.packet = s

	// TlsConfig
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if val, ok := s.Custom["hostName"]; ok {
		tlsConfig.ServerName = val.(string)
	}

	// Transport segment
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		// MaxIdleConnsPerHost: 100, Let's think about this.
	}
	if val, ok := s.Custom["disableKeepAlives"]; ok {
		tr.DisableKeepAlives = val.(bool)
	}
	if val, ok := s.Custom["disableCompression"]; ok {
		tr.DisableCompression = val.(bool)
	}
	if val, ok := s.Custom["h2"]; ok {
		val := val.(bool)
		if val {
			http2.ConfigureTransport(tr)
		}
	}

	// http client
	h.client = &http.Client{Transport: tr, Timeout: time.Duration(s.Timeout) * time.Second}
	if val, ok := s.Custom["disableRedirect"]; ok {
		val := val.(bool)
		if val {
			h.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}

	// Request instance
	h.request, err = http.NewRequest(h.packet.Method, h.packet.URL, bytes.NewBufferString(h.packet.Payload))
	if err != nil {
		return
	}

	header := make(http.Header)
	for k, v := range h.packet.Headers {
		header.Set(k, v)
	}
	h.request.Header = header

	// fmt.Println("Http Requester.")
	return
}

func (h *httpRequester) Send(proxyAddr *url.URL) (res *types.ResponseItem) {
	if proxyAddr != nil {
		h.client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyAddr) // bind ProxyService.GetNewProxy at init method?
	}
	// trace := &httptrace.ClientTrace{
	// 	GetConn: func(h string) {
	// 		connStart = now()
	// 	},
	// 	GotConn: func(connInfo httptrace.GotConnInfo) {
	// 		if !connInfo.Reused {
	// 			connDuration = now() - connStart
	// 		}
	// 		reqStart = now()
	// 	},
	// 	WroteRequest: func(w httptrace.WroteRequestInfo) {
	// 		reqDuration = now() - reqStart
	// 		delayStart = now()
	// 	},
	// 	GotFirstResponseByte: func() {
	// 		delayDuration = now() - delayStart
	// 		resStart = now()
	// 	},
	// }
	httpReq := h.request.Clone(context.TODO())
	// httpReq.URL.RawQuery += uuid.NewString() // TODO: this can be a feature. like -cache_bypass flag?
	httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(h.packet.Payload))

	var statusCode int
	var contentLength int64
	var requestErr types.RequestError
	httpRes, err := h.client.Do(httpReq)
	// fmt.Println(httpRes.StatusCode)
	if err != nil {
		ue, ok := err.(*url.Error)

		// TODO:REFACTOR
		// Currently we can't detect exact error type by returned err.
		// But we need to find an elegant way instead of this.
		if ok {
			if strings.Contains(ue.Err.Error(), "proxyconnect") {
				if strings.Contains(ue.Err.Error(), "connection refused") {
					requestErr = types.RequestError{Type: types.ErrorProxy, Reason: types.ReasonProxyFailed}
				} else if strings.Contains(ue.Err.Error(), "Client.Timeout") {
					requestErr = types.RequestError{Type: types.ErrorProxy, Reason: types.ReasonProxyTimeout}
				} else {
					requestErr = types.RequestError{Type: types.ErrorProxy, Reason: err.Error()}
				}
			} else if ok && strings.Contains(ue.Err.Error(), context.DeadlineExceeded.Error()) {
				requestErr = types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnTimeout}
			} else {
				requestErr = types.RequestError{Type: types.ErrorConn, Reason: ue.Err.Error()}
			}
		} else {
			requestErr = types.RequestError{Type: types.ErrorConn, Reason: err.Error()}
		}

	} else {
		contentLength = httpRes.ContentLength
		statusCode = httpRes.StatusCode
		httpRes.Body.Close()
	}

	defer func() {
		h.client.Transport.(*http.Transport).Proxy = nil
	}()
	// fmt.Println("S: ", httpRes.StatusCode)
	res = &types.ResponseItem{
		ScenarioItemID: h.packet.ID,
		RequestID:      uuid.New(),
		StatusCode:     statusCode,
		ContentLenth:   contentLength,
		Err:            requestErr,
	}
	return
}
