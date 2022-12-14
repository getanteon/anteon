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

// ScenarioResult is corresponding to Scenario. Each Scenario has a ScenarioResult after the scenario is played.
type ScenarioResult struct {
	// First request start time for the Scenario
	StartTime time.Time

	ProxyAddr   *url.URL
	StepResults []*ScenarioStepResult

	// Dynamic field for extra data needs in response object consumers.
	Others map[string]interface{}
}

// ScenarioStepResult is corresponding to ScenarioStep.
type ScenarioStepResult struct {
	// ID of the ScenarioStep
	StepID uint16

	// Name of the ScenarioStep
	StepName string

	// Each request has a unique ID.
	RequestID uuid.UUID

	// Returned status code. Has different meaning for different protocols.
	StatusCode int

	// Time of the request call.
	RequestTime time.Time

	// Total duration. From request sending to full response receiving.
	Duration time.Duration

	// Response content length
	ContentLength int64

	// Error occurred at request time.
	Err RequestError

	// Detailed Debug Info
	DebugInfo map[string]interface{}

	// Protocol spesific metrics. For ex: DNSLookupDuration: 1s for HTTP
	Custom map[string]interface{}
}
