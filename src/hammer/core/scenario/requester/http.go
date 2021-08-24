package requester

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
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

	return
}

func (h *httpRequester) Send(proxyAddr *url.URL) (res *types.ResponseItem) {
	var statusCode int
	var contentLength int64
	var requestErr types.RequestError

	var dnsStart, connStart, tlsHandshakeStart, resStart, reqStart, delayStart time.Time
	var dnsDur, connDur, tlsHandshakeDur, resDur, reqDur, delayDur time.Duration
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsDur = time.Since(dnsStart)
		},
		GetConn: func(h string) {
			connStart = time.Now()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			if !connInfo.Reused {
				connDur = time.Since(connStart)
			}
			reqStart = time.Now()
		},
		TLSHandshakeStart: func() {
			tlsHandshakeStart = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, e error) {
			if cs.HandshakeComplete && !cs.DidResume {
				tlsHandshakeDur = time.Since(tlsHandshakeStart)
			}
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			reqDur = time.Since(reqStart)
			delayStart = time.Now()
		},
		GotFirstResponseByte: func() {
			delayDur = time.Since(delayStart)
			resStart = time.Now()
		},
	}

	if proxyAddr != nil {
		h.client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyAddr) // bind ProxyService.GetNewProxy at init method?
	}
	defer func() {
		h.client.Transport.(*http.Transport).Proxy = nil
	}()

	httpReq := h.request.Clone(context.TODO())
	// httpReq.URL.RawQuery += uuid.NewString() // TODO: this can be a feature. like -cache_bypass flag?
	httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(h.packet.Payload))
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))

	start := time.Now()
	httpRes, err := h.client.Do(httpReq)
	resDur = time.Since(resStart)
	duration := time.Since(start)

	if err != nil {
		ue, ok := err.(*url.Error)

		if ok {
			requestErr = fetchErrType(ok, ue, err)
		} else {
			requestErr = types.RequestError{Type: types.ErrorUnkown, Reason: err.Error()}
		}

	} else {
		contentLength = httpRes.ContentLength
		statusCode = httpRes.StatusCode
		httpRes.Body.Close()
	}

	res = &types.ResponseItem{
		ScenarioItemID: h.packet.ID,
		RequestID:      uuid.New(),
		StatusCode:     statusCode,
		RequestTime:    start,
		Duration:       duration,
		ContentLenth:   contentLength,
		Err:            requestErr,
		Custom: map[string]interface{}{
			"dnsDuration":   dnsDur,
			"connDuration":  connDur,
			"reqDuration":   reqDur,
			"resDuration":   resDur,
			"delayDuration": delayDur,
		},
	}
	if h.packet.Protocol == types.ProtocolHTTPS {
		res.Custom["tlsDuration"] = tlsHandshakeDur
	}

	return
}

// TODO:REFACTOR
// Currently we can't detect exact error type by returned err.
// But we need to find an elegant way instead of this.
func fetchErrType(ok bool, ue *url.Error, err error) types.RequestError {
	var requestErr types.RequestError
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

	return requestErr
}
