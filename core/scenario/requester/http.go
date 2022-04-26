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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.ddosify.com/ddosify/core/types"
	"golang.org/x/net/http2"
)

type HttpRequester struct {
	ctx       context.Context
	proxyAddr *url.URL
	packet    types.ScenarioItem
	client    *http.Client
	request   *http.Request
}

// Init creates a client with the given scenarioItem. HttpRequester uses the same http.Client for all requests
func (h *HttpRequester) Init(ctx context.Context, s types.ScenarioItem, proxyAddr *url.URL) (err error) {
	h.ctx = ctx
	h.packet = s
	h.proxyAddr = proxyAddr

	// TlsConfig
	tlsConfig := h.initTLSConfig()

	// Transport segment
	tr := h.initTransport(tlsConfig)

	// http client
	h.client = &http.Client{Transport: tr, Timeout: time.Duration(h.packet.Timeout) * time.Second}
	if val, ok := h.packet.Custom["disable-redirect"]; ok {
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

func (h *HttpRequester) Done() {
	// MaxIdleConnsPerHost and MaxIdleConns at Transport layer configuration
	// let us reuse the connections when keep-alive enabled(default)
	// When the Job is finished, we have to Close idle connections to prevent sockets to lock in at the TIME_WAIT state.
	// Otherwise, the next job can't use these sockets because they are reserved for the current target host.
	h.client.CloseIdleConnections()
}

func (h *HttpRequester) Send() (res *types.ResponseItem) {
	var statusCode int
	var contentLength int64
	var requestErr types.RequestError
	var reqStartTime = time.Now()

	durations := &duration{}
	trace := newTrace(durations, h.proxyAddr)
	httpReq := h.prepareReq(trace)

	// Action
	httpRes, err := h.client.Do(httpReq)
	durations.setResDur()

	// Error checking
	if err != nil {
		ue, ok := err.(*url.Error)

		if ok {
			requestErr = fetchErrType(ue.Err.Error())
		} else {
			requestErr = types.RequestError{Type: types.ErrorUnkown, Reason: err.Error()}
		}

	} else {
		contentLength = httpRes.ContentLength
		statusCode = httpRes.StatusCode
	}

	// From the DOC: If the Body is not both read to EOF and closed,
	// the Client's underlying RoundTripper (typically Transport)
	// may not be able to re-use a persistent TCP connection to the server for a subsequent "keep-alive" request.
	if httpRes != nil {
		io.Copy(ioutil.Discard, httpRes.Body)
		httpRes.Body.Close()
	}

	// Finalize
	res = &types.ResponseItem{
		ScenarioItemID:   h.packet.ID,
		ScenarioItemName: h.packet.Name,
		RequestID:        uuid.New(),
		StatusCode:       statusCode,
		RequestTime:      reqStartTime,
		Duration:         durations.totalDuration(),
		ContentLength:    contentLength,
		Err:              requestErr,
		Custom: map[string]interface{}{
			"dnsDuration":           durations.getDNSDur(),
			"connDuration":          durations.getConnDur(),
			"reqDuration":           durations.getReqDur(),
			"resDuration":           durations.getResDur(),
			"serverProcessDuration": durations.getServerProcessDur(),
		},
	}
	if h.packet.Protocol == types.ProtocolHTTPS {
		res.Custom["tlsDuration"] = durations.getTLSDur()
	}

	return
}

func (h *HttpRequester) prepareReq(trace *httptrace.ClientTrace) *http.Request {
	httpReq := h.request.Clone(h.ctx)
	httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(h.packet.Payload))
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))
	// httpReq.URL.RawQuery += uuid.NewString() // TODO: this can be a feature. like -cache_bypass flag?
	return httpReq
}

// TODO:REFACTOR
// Currently we can't detect exact error type by returned err.
// But we need to find an elegant way instead of this.
func fetchErrType(err string) types.RequestError {
	var requestErr types.RequestError
	if strings.Contains(err, "proxyconnect") {
		if strings.Contains(err, "connection refused") {
			requestErr = types.RequestError{Type: types.ErrorProxy, Reason: types.ReasonProxyFailed}
		} else if strings.Contains(err, "Client.Timeout") {
			requestErr = types.RequestError{Type: types.ErrorProxy, Reason: types.ReasonProxyTimeout}
		} else {
			requestErr = types.RequestError{Type: types.ErrorProxy, Reason: err}
		}
	} else if strings.Contains(err, context.DeadlineExceeded.Error()) {
		requestErr = types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnTimeout}
	} else if strings.Contains(err, "i/o timeout") {
		requestErr = types.RequestError{Type: types.ErrorConn, Reason: types.ReasonReadTimeout}
	} else if strings.Contains(err, "connection refused") {
		requestErr = types.RequestError{Type: types.ErrorConn, Reason: types.ReasonConnRefused}
	} else if strings.Contains(err, context.Canceled.Error()) {
		requestErr = types.RequestError{Type: types.ErrorIntented, Reason: types.ReasonCtxCanceled}
	} else if strings.Contains(err, "connection reset by peer") {
		requestErr = types.RequestError{Type: types.ErrorConn, Reason: "connection reset by peer"}
	} else {
		requestErr = types.RequestError{Type: types.ErrorConn, Reason: err}
	}

	return requestErr
}

func (h *HttpRequester) initTransport(tlsConfig *tls.Config) *http.Transport {
	tr := &http.Transport{
		TLSClientConfig:     tlsConfig,
		Proxy:               http.ProxyURL(h.proxyAddr),
		MaxIdleConnsPerHost: 60000,
		MaxIdleConns:        0,
	}

	tr.DisableKeepAlives = false
	if val, ok := h.packet.Custom["keep-alive"]; ok {
		tr.DisableKeepAlives = !val.(bool)
	}
	if val, ok := h.packet.Custom["disable-compression"]; ok {
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

func (h *HttpRequester) initTLSConfig() *tls.Config {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if val, ok := h.packet.Custom["hostname"]; ok {
		tlsConfig.ServerName = val.(string)
	}
	return tlsConfig
}

func (h *HttpRequester) initRequestInstance() (err error) {
	h.request, err = http.NewRequest(h.packet.Method, h.packet.URL, bytes.NewBufferString(h.packet.Payload))
	if err != nil {
		return
	}

	// Headers
	header := make(http.Header)
	for k, v := range h.packet.Headers {
		if strings.EqualFold(k, "Host") {
			h.request.Host = v
		} else {
			header.Set(k, v)
		}
	}

	h.request.Header = header

	// Auth should be set after header assignment.
	if h.packet.Auth != (types.Auth{}) {
		h.request.SetBasicAuth(h.packet.Auth.Username, h.packet.Auth.Password)
	}

	// If keep-alive is false, prevent the reuse of the previous TCP connection at the request layer also.
	h.request.Close = false
	if val, ok := h.packet.Custom["keep-alive"]; ok {
		h.request.Close = !val.(bool)
	}
	return
}

func newTrace(duration *duration, proxyAddr *url.URL) *httptrace.ClientTrace {
	var dnsStart, connStart, tlsStart, reqStart, serverProcessStart time.Time

	// According to the doc in the trace.go;
	// Some of the hooks below can be triggered multiple times in case of retried connections, "Happy Eyeballs" etc..
	// Also, some of the hooks can be triggered after the TCP roundtrip if the request is not successfully finished.
	// To fetch the time only at the first trigger and prevent data race we need to use the mutex mechanism.
	// For start times, except resStart, this mutex is been using.
	// For duration calculations, "duration" struct internally uses another mutex.
	var m sync.Mutex

	return &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			m.Lock()
			if dnsStart.IsZero() {
				dnsStart = time.Now()
			}
			m.Unlock()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			m.Lock()
			// no need to handle error in here. We can detect it at http.Client.Do return.
			if dnsInfo.Err == nil {
				duration.setDNSDur(time.Since(dnsStart))
			}
			m.Unlock()
		},
		ConnectStart: func(network, addr string) {
			m.Lock()
			if connStart.IsZero() {
				connStart = time.Now()
			}
			m.Unlock()
		},
		ConnectDone: func(network, addr string, err error) {
			m.Lock()
			// no need to handle error in here. We can detect it at http.Client.Do return.
			if err == nil {
				duration.setConnDur(time.Since(connStart))
			}
			m.Unlock()
		},
		TLSHandshakeStart: func() {
			m.Lock()
			// This hook can be hit 2 times;
			// If both proxy and target are HTTPS
			//	First hit is for proxy, second is for target.
			//  To catch the second TLS start time (for target), we can't perform tlsStart.IsZero() check here.
			tlsStart = time.Now()
			m.Unlock()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, e error) {
			m.Lock()
			// This hook can be hit 2 times;
			// If proxy: HTTPS, target: HTTPS
			//	First hit is for proxy, second is for target TLS
			//  We need to calculate TLS duration if and only if the TLS handshake process is for the target.

			if e == nil {
				if proxyAddr == nil || proxyAddr.Hostname() != cs.ServerName {
					duration.setTLSDur(time.Since(tlsStart))
				}
			}
			m.Unlock()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			m.Lock()
			if reqStart.IsZero() {
				reqStart = time.Now()
			}
			m.Unlock()
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			m.Lock()
			// no need to handle error in here. We can detect it at http.Client.Do return.
			if w.Err == nil {
				duration.setReqDur(time.Since(reqStart))
				serverProcessStart = time.Now()
			}
			m.Unlock()
		},
		GotFirstResponseByte: func() {
			m.Lock()
			duration.setServerProcessDur(time.Since(serverProcessStart))
			duration.setResStartTime(time.Now())
			m.Unlock()
		},
	}
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

	mu sync.Mutex
}

func (d *duration) setResStartTime(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.resStart.IsZero() {
		d.resStart = t
	}
}

func (d *duration) setDNSDur(t time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dnsDur == 0 {
		d.dnsDur = t
	}
}

func (d *duration) getDNSDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dnsDur
}

func (d *duration) setTLSDur(t time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.tlsDur == 0 {
		d.tlsDur = t
	}
}

func (d *duration) getTLSDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.tlsDur
}

func (d *duration) setConnDur(t time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.connDur == 0 {
		d.connDur = t
	}
}

func (d *duration) getConnDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.connDur
}

func (d *duration) setReqDur(t time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.reqDur == 0 {
		d.reqDur = t
	}
}

func (d *duration) getReqDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.reqDur
}

func (d *duration) setServerProcessDur(t time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.serverProcessDur == 0 {
		d.serverProcessDur = t
	}
}

func (d *duration) getServerProcessDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.serverProcessDur
}

func (d *duration) setResDur() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resDur = time.Since(d.resStart)
}

func (d *duration) getResDur() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.resDur
}

func (d *duration) totalDuration() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.dnsDur + d.connDur + d.tlsDur + d.reqDur + d.serverProcessDur + d.resDur
}
