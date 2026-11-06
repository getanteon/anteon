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

import "fmt"

// Capacity says how a run is read as capacity. GPUs divides its throughput; each tier is an SLO
// a stage meets when the p99 of its first token and of its time per output token are within it.
type Capacity struct {
	GPUs  int            `json:"gpus,omitempty"`
	Tiers []CapacityTier `json:"tiers,omitempty"`
}

type CapacityTier struct {
	Name         string  `json:"name"`
	FirstTokenMs float64 `json:"first_token_ms"`
	TPOTMs       float64 `json:"tpot_ms"`
}

// DefaultCapacityTiers are MLPerf Inference's conversational and interactive latency limits for its
// Llama3.1-8B benchmark (inference_rules.adoc), on the 99th percentile.
var DefaultCapacityTiers = []CapacityTier{
	{Name: "conversational", FirstTokenMs: 2000, TPOTMs: 100},
	{Name: "interactive", FirstTokenMs: 500, TPOTMs: 30},
}

// GPUCount is GPUs, 1 when unset.
func (c *Capacity) GPUCount() int {
	if c == nil || c.GPUs < 1 {
		return 1
	}
	return c.GPUs
}

// TierList is Tiers, the defaults when none is given.
func (c *Capacity) TierList() []CapacityTier {
	if c == nil || len(c.Tiers) == 0 {
		return DefaultCapacityTiers
	}
	return c.Tiers
}

func (c *Capacity) validate() error {
	if c == nil {
		return nil
	}
	if c.GPUs < 0 {
		return fmt.Errorf("capacity gpus can not be negative")
	}
	for i, t := range c.Tiers {
		if t.Name == "" {
			return fmt.Errorf("capacity tier %d has no name", i+1)
		}
		if t.FirstTokenMs <= 0 && t.TPOTMs <= 0 {
			return fmt.Errorf("capacity tier %q needs first_token_ms, tpot_ms or both", t.Name)
		}
	}
	return nil
}
