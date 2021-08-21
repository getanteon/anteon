package requester

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	fmt.Println("Http Requester.")
	return
}

func (h *httpRequester) Send(proxyAddr *url.URL) (res *types.ResponseItem, err error) {
	if proxyAddr != nil {
		fmt.Println("sd")
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
	httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(h.packet.Payload))

	httpRes, err := h.client.Do(httpReq)
	if err != nil {
		ue, ok := err.(*url.Error)

		// TODO: Currently we can't detect proxy error by returned err. But we need to find an elegant way instead of this.
		if ok && ue.Err.Error() == "proxyconnect tcp: dial tcp :0: connect: connection refused" {
			err = &types.Error{Type: types.ErrorProxy, Reason: types.ReasonProxyFailed}
		}
		fmt.Println("err: ", ue.Err.Error())
		fmt.Println("err: ", ue.Err)
		return nil, err
	}

	defer func() {
		defer httpRes.Body.Close()
		h.client.Transport.(*http.Transport).Proxy = nil
	}()

	res = &types.ResponseItem{
		RequestID: uuid.New(),
	}

	return
}
