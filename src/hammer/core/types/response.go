package types

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	StartTime   time.Time
	AvgDuration time.Duration
	Errors      map[string]int
	StatusCodes map[int]int

	ResponseItems []*ResponseItem

	// Protocol spesific total metrics
	Custom map[string]interface{}
}

type ResponseItem struct {
	ScenarioItemID int16
	RequestID      uuid.UUID
	StatusCode     int

	// Time of the request call.
	RequestTime time.Time

	//
	Duration     time.Duration
	ContentLenth int64

	Err RequestError

	// Protocol spesific metrics. For ex: DNSLookupDuration: 1s for HTTP
	Custom map[string]interface{}
}
