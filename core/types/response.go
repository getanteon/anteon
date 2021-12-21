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

package types

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

// Response is corresponding to Scenario. Each Scenario has a Response after the request is done.
type Response struct {
	// First request start time for the Scenario
	StartTime time.Time

	ProxyAddr     *url.URL
	ResponseItems []*ResponseItem

	// Dynamic field for extra data needs in response object consumers.
	Others map[string]interface{}
}

// ResponseItem is corresponding to ScenarioItem.
type ResponseItem struct {
	// ID of the ScenarioItem
	ScenarioItemID int16

	// Name of the ScenarioItem
	ScenarioItemName string

	// Each request has a unique ID.
	RequestID uuid.UUID

	// Returned status code. Has different meaning for different protocols.
	StatusCode int

	// Time of the request call.
	RequestTime time.Time

	// Total duration. From request sending to full response receiving.
	Duration time.Duration

	// Response content length
	ContentLenth int64

	// Error occurred at request time.
	Err RequestError

	// Protocol spesific metrics. For ex: DNSLookupDuration: 1s for HTTP
	Custom map[string]interface{}
}
