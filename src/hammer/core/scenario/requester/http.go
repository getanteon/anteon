package requester

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
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
	proxyAddr *url.URL
	packet    types.ScenarioItem
	client    *http.Client
	request   *http.Request
}

// Create a client with scenarioItem and use same client for each request
func (h *httpRequester) Init(s types.ScenarioItem, proxyAddr *url.URL) (err error) {
	h.packet = s
	h.proxyAddr = proxyAddr

	// TlsConfig
	tlsConfig := h.initTlsConfig()

	// Transport segment
	tr := h.initTransport(tlsConfig)

	// http client
	h.client = &http.Client{Transport: tr, Timeout: time.Duration(h.packet.Timeout) * time.Second}
	if val, ok := h.packet.Custom["disableRedirect"]; ok {
		val := val.(bool)
		if val {
			h.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}

	// Request instance
	err = h.initRequestInstance()
	if err != nil {
		return
	}

	return
}

func (h *httpRequester) Send() (res *types.ResponseItem) {
	var statusCode int
	var contentLength int64
	var requestErr types.RequestError
	var reqStartTime = time.Now()

	durations := &duration{}
	trace := h.newTrace(durations)
	httpReq := h.prepareReq(trace)

	// Action
	httpRes, err := h.client.Do(httpReq)
	resDur := time.Since(durations.resStart)

	// Error checking
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

		// From the DOC: If the Body is not both read to EOF and closed,
		// the Client's underlying RoundTripper (typically Transport)
		// may not be able to re-use a persistent TCP connection to the server for a subsequent "keep-alive" request.
		io.Copy(ioutil.Discard, httpRes.Body)
		httpRes.Body.Close()
	}

	// Finalize
	res = &types.ResponseItem{
		ScenarioItemID: h.packet.ID,
		RequestID:      uuid.New(),
		StatusCode:     statusCode,
		RequestTime:    reqStartTime,
		Duration:       durations.totalDuration(),
		ContentLenth:   contentLength,
		Err:            requestErr,
		Custom: map[string]interface{}{
			"dnsDuration":           durations.dnsDur,
			"connDuration":          durations.connDur,
			"reqDuration":           durations.reqDur,
			"resDuration":           resDur,
			"serverProcessDuration": durations.serverProcessDur,
		},
	}
	if h.packet.Protocol == types.ProtocolHTTPS {
		res.Custom["tlsDuration"] = durations.tlsDur
	}

	return
}

func (h *httpRequester) prepareReq(trace *httptrace.ClientTrace) *http.Request {
	httpReq := h.request.Clone(context.TODO())
	httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(h.packet.Payload))
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))
	// httpReq.URL.RawQuery += uuid.NewString() // TODO: this can be a feature. like -cache_bypass flag?
	return httpReq
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

func (h *httpRequester) initTransport(tlsConfig *tls.Config) *http.Transport {
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyURL(h.proxyAddr),
		// MaxIdleConnsPerHost: 100, TODO: Let's think about this.
	}

	tr.DisableKeepAlives = true
	if val, ok := h.packet.Custom["disableKeepAlives"]; ok {
		tr.DisableKeepAlives = val.(bool)
	}
	if val, ok := h.packet.Custom["disableCompression"]; ok {
		tr.DisableCompression = val.(bool)
	}
	if val, ok := h.packet.Custom["h2"]; ok {
		val := val.(bool)
		if val {
			http2.ConfigureTransport(tr)
		}
	}
	return tr
}

func (h *httpRequester) initTlsConfig() *tls.Config {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if val, ok := h.packet.Custom["hostName"]; ok {
		tlsConfig.ServerName = val.(string)
	}
	return tlsConfig
}

func (h *httpRequester) newTrace(duration *duration) *httptrace.ClientTrace {
	var dnsStart, connStart, tlsStart, reqStart, serverProcessStart time.Time

	return &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			duration.dnsDur = time.Since(dnsStart)
		},
		ConnectStart: func(network, addr string) {
			connStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			duration.connDur = time.Since(connStart)
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, e error) {
			duration.tlsDur = time.Since(tlsStart)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			reqStart = time.Now()
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			duration.reqDur = time.Since(reqStart)
			serverProcessStart = time.Now()
		},
		GotFirstResponseByte: func() {
			duration.serverProcessDur = time.Since(serverProcessStart)
			duration.resStart = time.Now()
		},
	}
}

func (d *duration) totalDuration() time.Duration {
	return d.dnsDur + d.connDur + d.tlsDur + d.reqDur + d.serverProcessDur + d.resDur
}

func (h *httpRequester) initRequestInstance() (err error) {
	h.request, err = http.NewRequest(h.packet.Method, h.packet.URL, bytes.NewBufferString(h.packet.Payload))
	if err != nil {
		return
	}

	// Headers
	header := make(http.Header)
	for k, v := range h.packet.Headers {
		header.Set(k, v)
	}

	ua := header.Get("User-Agent")
	if ua == "" {
		ua = types.DdosifyUserAgent
	} else {
		ua += " " + types.DdosifyUserAgent
	}
	header.Set("User-Agent", ua)

	h.request.Header = header

	// Auth should be set after header assignment.
	if h.packet.Auth != (types.Auth{}) {
		h.request.SetBasicAuth(h.packet.Auth.Username, h.packet.Auth.Password)
	}
	return
}

type duration struct {
	// Time at response reading start
	resStart time.Time

	// DNS lookup duration. If IP:Port porvided instead of domain, this will be 0
	dnsDur time.Duration

	// TCP connection setup duration
	connDur time.Duration

	// TLS handshake duration. For HTTP this will be 0
	tlsDur time.Duration

	// Request write duration
	reqDur time.Duration

	// Duration between full request write to first response. AKA Time To First Byte (TTFB)
	serverProcessDur time.Duration

	// Resposne read duration
	resDur time.Duration
}
