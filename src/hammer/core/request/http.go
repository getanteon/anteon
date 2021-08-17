package request

import (
	"fmt"
	"math/rand"
	"net/url"

	"ddosify.com/hammer/core/types"
	"github.com/google/uuid"
)

type httpRequest struct {
	request
}

func (h *httpRequest) init(p types.Packet, s types.Scenario) {
	h.request.Packet = p
	h.request.Scenario = s
	fmt.Println("Http Request Service initialized.")
}

func (h *httpRequest) Send(proxyAddr *url.URL) (res types.Response, err error) {

	// DO request

	if rand.Intn(10)%2 == 0 {
		err = &types.Error{Type: types.ErrorProxy, Reason: types.ReasonProxyTimeout}
	}

	res = types.Response{
		RequestID: uuid.New(),
	}
	fmt.Println("[HTTP] Sending request.")

	return
}
