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

package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"go.ddosify.com/ddosify/v2/core/types"
)

const stageFormat = "DURATION:COUNT[:NAME[:warmup]], or c=CONCURRENCY:COUNT[:NAME[:warmup]] for a closed-loop stage"

// minPerLevel is the fewest iterations --concurrency gives a level by default.
const minPerLevel = 20

// parseStages reads --stage values, each DURATION:COUNT[:NAME[:warmup]], into manual_load stages.
func parseStages(values []string) (types.TimeRunCount, error) {
	out := make(types.TimeRunCount, len(values))
	for i, v := range values {
		bad := func(format string, a ...interface{}) error {
			return fmt.Errorf("--stage %q: %s, the format is %s", v, fmt.Sprintf(format, a...), stageFormat)
		}
		f := strings.Split(v, ":")
		if len(f) < 2 {
			return nil, bad("no COUNT")
		}
		if len(f) > 4 {
			return nil, bad("too many fields; a NAME can not hold a colon")
		}
		if c, ok := strings.CutPrefix(strings.TrimSpace(f[0]), "c="); ok {
			n, err := strconv.Atoi(c)
			if err != nil || n < 1 {
				return nil, bad("CONCURRENCY %q must be a whole number of 1 or more", c)
			}
			out[i].Concurrency = n
		}
		if out[i].Concurrency > 0 {
			n, err := strconv.Atoi(strings.TrimSpace(f[1]))
			if err != nil || n < 1 {
				return nil, bad("COUNT %q must be a whole number of 1 or more", f[1])
			}
			out[i].Count = n
			if len(f) > 2 {
				out[i].Name = f[2]
			}
			if len(f) == 4 {
				if f[3] != "warmup" {
					return nil, bad("the fourth field can only be warmup, got %q", f[3])
				}
				out[i].Warmup = true
			}
			continue
		}
		d, err := time.ParseDuration(strings.TrimSpace(f[0]))
		if err != nil {
			return nil, bad("DURATION %q is not a duration such as 60s or 4m", f[0])
		}
		if d <= 0 {
			return nil, bad("DURATION %q must be greater than 0", f[0])
		}
		if d%time.Second != 0 {
			return nil, bad("DURATION %q must be a whole number of seconds", f[0])
		}
		n, err := strconv.Atoi(strings.TrimSpace(f[1]))
		if err != nil || n < 0 {
			return nil, bad("COUNT %q must be a whole number of 0 or more", f[1])
		}
		out[i].Duration, out[i].Count = int(d/time.Second), n
		if len(f) > 2 {
			out[i].Name = f[2]
		}
		if len(f) == 4 {
			if f[3] != "warmup" {
				return nil, bad("the fourth field can only be warmup, got %q", f[3])
			}
			out[i].Warmup = true
		}
	}
	return out, nil
}

// applyStages puts --stage in place of the config's manual_load, with the iteration count and
// the duration it adds up to, as a manual_load in the config file would.
func applyStages(h *types.Hammer, flags *pflag.FlagSet, o *runOptions) error {
	if flags.Changed("gpus") {
		if o.gpus < 1 {
			return fmt.Errorf("--gpus must be 1 or more")
		}
		if h.Capacity == nil {
			h.Capacity = &types.Capacity{}
		}
		h.Capacity.GPUs = o.gpus
	}
	if (flags.Changed("per-level") || flags.Changed("warmup-iterations")) && !flags.Changed("concurrency") {
		return fmt.Errorf("--per-level and --warmup-iterations go with --concurrency")
	}
	if !flags.Changed("stage") && !flags.Changed("concurrency") {
		return nil
	}
	if flags.Changed("stage") && flags.Changed("concurrency") {
		return fmt.Errorf("--stage and --concurrency both set the stages; use one")
	}
	if flags.Changed("iterations") || flags.Changed("duration") {
		return fmt.Errorf("--stage and --concurrency set the iterations and the duration; drop --iterations and --duration")
	}
	var stages types.TimeRunCount
	var err error
	if flags.Changed("concurrency") {
		stages, err = concurrencyStages(o.concurrency, o.perLevel, o.warmupIterations)
	} else {
		stages, err = parseStages(o.stages)
	}
	if err != nil {
		return err
	}
	h.TimeRunCountMap = stages
	h.IterationCount, h.TestDuration = stages.Totals()
	return nil
}

// concurrencyStages is one closed-loop stage per level, named c=LEVEL, each of perLevel iterations
// or, with 0, of max(10 x level, minPerLevel), so every level runs ten rounds at least; warmup
// iterations at the first level come before them when given.
func concurrencyStages(levels []int, perLevel, warmup int) (types.TimeRunCount, error) {
	if len(levels) == 0 {
		return nil, fmt.Errorf("--concurrency needs one level or more, such as 1,2,4,8")
	}
	if perLevel < 0 || warmup < 0 {
		return nil, fmt.Errorf("--per-level and --warmup-iterations can not be negative")
	}
	var out types.TimeRunCount
	add := func(c, n int, name string, warm bool) {
		out = append(out, types.TimeRunCount{{Count: n, Name: name, Warmup: warm, Concurrency: c}}...)
	}
	if warmup > 0 {
		add(levels[0], warmup, "warm-up", true)
	}
	for _, c := range levels {
		if c < 1 {
			return nil, fmt.Errorf("--concurrency level %d must be 1 or more", c)
		}
		n := perLevel
		if n == 0 {
			n = max(10*c, minPerLevel)
		}
		add(c, n, fmt.Sprintf("c=%d", c), false)
	}
	return out, nil
}
