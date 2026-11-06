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

package report

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.ddosify.com/ddosify/v2/core/types"
)

// maxFirstErrors bounds the distinct errors told as they happen; the report has them all.
const maxFirstErrors = 20

var quoted = regexp.MustCompile(`"[^"]*"`)

// firstErrors writes the first failure of each kind on stderr the moment it happens, so a run
// whose data goes to a file says at once why its requests fail. A kind is the step, the outcome
// and the reason with its quoted URL left out, so a URL with an id in it does not repeat it.
// Only the report's goroutine calls note.
type firstErrors struct {
	seen map[string]bool
}

func (f *firstErrors) note(w io.Writer, scr *types.ScenarioResult) {
	for _, sr := range scr.StepResults {
		outcome := OutcomeOf(sr)
		if outcome == OutcomeOK || outcome == OutcomeInterrupted {
			continue
		}
		reason := sr.Err.Reason
		if reason == "" && sr.StatusCode != 0 {
			reason = fmt.Sprintf("HTTP %d", sr.StatusCode)
		}
		if reason == "" {
			reason = sr.Err.Type
		}
		key := fmt.Sprintf("%d|%s|%s", sr.StepID, outcome, quoted.ReplaceAllString(reason, `""`))
		if f.seen == nil {
			f.seen = map[string]bool{}
		}
		if f.seen[key] || len(f.seen) >= maxFirstErrors {
			continue
		}
		f.seen[key] = true
		shown := reason
		if sr.Err.Detail != "" {
			shown = sr.Err.Detail
		} else if sr.Err.Type == "" && len(sr.RespBody) > 0 {
			if line := firstLine(sr.RespBody); line != "" {
				shown += ": " + line
			}
		}
		if len(shown) > 500 {
			shown = shown[:500] + "..."
		}
		step := fmt.Sprintf("step %d", sr.StepID)
		if sr.StepName != "" {
			step += fmt.Sprintf(" %q", sr.StepName)
		}
		fmt.Fprintf(w, "Error: %s: %s: %s\n", step, outcome, strings.TrimSpace(shown))
		if len(f.seen) == maxFirstErrors {
			fmt.Fprintf(w, "Error: %d kinds of errors so far; more are in the report only\n", maxFirstErrors)
		}
	}
}

// firstLine is the first non-empty line of a response body, at most 200 bytes and valid UTF-8.
func firstLine(b []byte) string {
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if len(l) > 200 {
				l = strings.ToValidUTF8(l[:200], "") + "..."
			}
			return l
		}
	}
	return ""
}
