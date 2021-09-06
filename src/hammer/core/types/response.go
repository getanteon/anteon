package types

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

//Equivalent to Scenario. Each Scenario has a Response after request is done.
type Response struct {
	// First request start time for the Scenario
	StartTime time.Time

	ProxyAddr     *url.URL
	ResponseItems []*ResponseItem
}

//Equivalent to ScenarioItem.
type ResponseItem struct {
	// ID of the ScenarioItem
	ScenarioItemID int16

	// Each request has a unique ID.
	RequestID uuid.UUID

	// Returned status code. Has different meaning for different protocols.
	StatusCode int

	// Time of the request call.
	RequestTime time.Time

	// Total duration. From request sending to full response recieving.
	Duration time.Duration

	// Response content length
	ContentLenth int64

	// Error occured at request time.
	Err RequestError

	// Protocol spesific metrics. For ex: DNSLookupDuration: 1s for HTTP
	Custom map[string]interface{}
}
