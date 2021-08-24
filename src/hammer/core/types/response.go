package types

import (
	"time"

	"github.com/google/uuid"
)

//Equivalent to Scenario. Each Scenario has a Response after request is done.
type Response struct {
	// Response starting time for Scenario
	StartTime time.Time

	// // Total duration of the all ResponseItem.Duration
	// Duration time.Duration

	// // Error distributation of ResponseItems. Ex: map[scenario_item_id][conn_err:conn_timeout] = 3
	// ErrorDist map[int]map[string]int

	// // Status code distribution of the ResponseItems. Ex: map[scenario_item_id][200] = 12
	// StatusCodes map[int]map[int]int

	ResponseItems []*ResponseItem

	// // Protocol spesific total metrics
	// Custom map[string]interface{}
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
