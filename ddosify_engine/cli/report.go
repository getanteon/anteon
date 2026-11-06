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
	"io"
	"os"

	"github.com/spf13/cobra"

	elive "go.ddosify.com/ddosify/v2/core/live"
)

func newReportCmd(s Streams) *cobra.Command {
	var out, title string
	cmd := &cobra.Command{
		Use:   "report [LOG]",
		Short: "Write the HTML report of a finished run from its --json output",
		Long: `Reads the output of "ddosify run --json": the "report" line, and the per-second "progress"
lines and the per-request "request" lines (request_log in the config) when they are there.
Lines that are not JSON, such as a pod log's status lines, are skipped.
With request lines the report adds latency by start time and each stage's steady state.
LOG is a file, or - and no argument for stdin; -o - writes the report to stdout. The report loads
nothing from the network.`,
		Example: `  ddosify run --config config.json --json > run.ndjson
  ddosify report run.ndjson -o report.html
  kubectl logs job/loadtest | ddosify report -o report.html
  docker run -i --rm IMAGE report -o - < run.ndjson > report.html`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var in io.Reader = s.In
			name := "stdin"
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return usageErr(err)
				}
				defer f.Close()
				in, name = f, args[0]
			}
			if elive.LogReport == nil {
				return runtimeErr(fmt.Errorf("this build has no HTML report"))
			}
			var f *os.File
			open := func() (io.Writer, error) {
				if out == "-" {
					return s.Out, nil
				}
				var err error
				f, err = os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
				return f, err
			}
			err := elive.LogReport(in, title, open)
			if f != nil {
				if cerr := f.Close(); err == nil {
					err = cerr
				}
			}
			if err != nil {
				return usageErr(fmt.Errorf("%s: %w", name, err))
			}
			if out != "-" {
				fmt.Fprintf(s.Err, "HTML report written to %s\n", out)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "report.html", "the HTML file to write, or - for stdout")
	cmd.Flags().StringVar(&title, "title", "", "the report's title")
	return cmd
}
