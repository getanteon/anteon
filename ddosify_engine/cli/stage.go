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

const stageFormat = "DURATION:COUNT[:NAME[:warmup]]"

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
	if !flags.Changed("stage") {
		return nil
	}
	if flags.Changed("iterations") || flags.Changed("duration") {
		return fmt.Errorf("--stage sets the iterations and the duration; drop --iterations and --duration")
	}
	stages, err := parseStages(o.stages)
	if err != nil {
		return err
	}
	h.TimeRunCountMap = stages
	h.IterationCount, h.TestDuration = stages.Totals()
	return nil
}
