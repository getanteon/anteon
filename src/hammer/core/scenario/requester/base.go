package requester

import (
	"fmt"
	"net/url"
	"strings"

	"ddosify.com/hammer/core/types"
)

type Requester interface {
	Init(types.ScenarioItem, *url.URL) error
	Send() *types.ResponseItem
}

func NewRequester(s types.ScenarioItem) (Requester, error) {
	if strings.EqualFold(s.Protocol, "http") ||
		strings.EqualFold(s.Protocol, "https") {
		return &httpRequester{}, nil
	}
	return nil, fmt.Errorf("No proper requester")
}
