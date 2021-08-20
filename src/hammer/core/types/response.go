package types

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	ResponseItems []*ResponseItem

	// Protocol spesific total metrics
	Custom map[string]interface{}
}

type ResponseItem struct {
	ScenarioItemID int16
	RequestID      uuid.UUID
	StatusCode     int

	RequestTime  time.Time
	ResponseTime time.Time
	Duration     time.Duration
	ContentLenth int64

	// Protocol spesific metrics. For ex: DNSLookupDuration: 1s for HTTP
	Custom map[string]interface{}
}
