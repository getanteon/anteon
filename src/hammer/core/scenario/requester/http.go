package requester

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"ddosify.com/hammer/core/types"
	"github.com/google/uuid"
)

type httpRequester struct {
	client *http.Client
}

func (h *httpRequester) Init(s types.ScenarioItem) (err error) {
	// Create a client with scenarioItem and use same client for each request
	h.client = &http.Client{}
	fmt.Println("Http Requester.")
	return
}

func (h *httpRequester) Send(proxyAddr *url.URL) (res *types.ResponseItem, err error) {

	// DO request
	fmt.Println("Sendin req.")
	if rand.Intn(10)%2 == 0 {
		err = &types.Error{Type: types.ErrorProxy, Reason: types.ReasonProxyTimeout}
	}

	time.Sleep(2 * time.Second)
	res = &types.ResponseItem{
		RequestID: uuid.New(),
	}

	return
}
