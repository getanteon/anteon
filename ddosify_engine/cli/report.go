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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	elive "go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

func newReportCmd(s Streams) *cobra.Command {
	var out, title string
	var labels, tiers, resources, per []string
	var gpus int
	cmd := &cobra.Command{
		Use:   "report [LOG...]",
		Short: "Write the HTML report of a finished run, or compare runs, from their --json output",
		Long: `Reads the output of "ddosify run --json": the "report" line, and the per-second "progress"
lines and the per-request "request" lines (request_log in the config) when they are there.
Lines that are not JSON, such as a pod log's status lines, are skipped.
With request lines the report adds latency by start time and each stage's steady state.
LOG is a file, or - and no argument for stdin; -o - writes the report to stdout. The report loads
nothing from the network.

With two logs or more the page compares the runs: their throughput against speed per user on one
chart, and the concurrency each SLO tier allows. --label names each run, in the order of the logs;
the file name is the default.

--resources adds the system under test's CPU, memory and GPU over the run, and per stage, from CSV
files of timestamp,source,metric,value; repeat it. In a comparison each file names its run,
--resources LABEL=FILE. See "HTML report" in the README for the format.`,
		Example: `  ddosify run --config config.json --json > run.ndjson
  ddosify report run.ndjson -o report.html
  kubectl logs job/loadtest | ddosify report -o report.html
  docker run -i --rm IMAGE report -o - < run.ndjson > report.html
  ddosify report a.ndjson b.ndjson --label "model A" --label "model B" --gpus 1 -o compare.html
  ddosify report run.ndjson --resources pods.csv --resources gpus.csv --resources-per masking -o report.html
  ddosify report a.ndjson b.ndjson --label cpu --label gpu --resources cpu=a.csv --resources gpu=b.csv -o compare.html`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(labels) > 0 && len(labels) != len(args) {
				return usageErr(fmt.Errorf("%d --label for %d logs: give one for each log, or none", len(labels), len(args)))
			}
			if cmd.Flags().Changed("gpus") && gpus < 1 {
				return usageErr(fmt.Errorf("--gpus must be 1 or more"))
			}
			opts := elive.ReportOptions{Title: title, GPUs: gpus, ResourcesPer: per, Stderr: s.Err}
			if len(per) > 0 && len(resources) == 0 {
				return usageErr(fmt.Errorf("--resources-per needs --resources"))
			}
			runLabels := make([]string, len(args))
			for i, a := range args {
				runLabels[i] = strings.TrimSuffix(filepath.Base(a), filepath.Ext(a))
				if len(labels) > 0 {
					runLabels[i] = labels[i]
				}
			}
			for _, v := range resources {
				run, path, err := resourceArg(v, runLabels)
				if err != nil {
					return usageErr(err)
				}
				fh, err := os.Open(path)
				if err != nil {
					return usageErr(fmt.Errorf("--resources: %w", err))
				}
				defer fh.Close()
				opts.Resources = append(opts.Resources, elive.ResourceLog{Run: run, Name: path, In: fh})
			}
			for _, t := range tiers {
				tier, err := parseTier(t)
				if err != nil {
					return usageErr(err)
				}
				opts.Tiers = append(opts.Tiers, tier)
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
			closeOut := func(err error) error {
				if f != nil {
					if cerr := f.Close(); err == nil {
						err = cerr
					}
				}
				return err
			}
			if len(args) > 1 {
				if elive.CompareLogs == nil {
					return runtimeErr(fmt.Errorf("this build has no HTML report"))
				}
				logs := make([]elive.LabeledLog, 0, len(args))
				for i, a := range args {
					if a == "-" {
						return usageErr(fmt.Errorf("a comparison reads files; stdin can not be one of several logs"))
					}
					fh, err := os.Open(a)
					if err != nil {
						return usageErr(err)
					}
					defer fh.Close()
					logs = append(logs, elive.LabeledLog{Label: runLabels[i], In: fh})
				}
				if err := closeOut(elive.CompareLogs(logs, opts, open)); err != nil {
					return usageErr(err)
				}
				if out != "-" {
					fmt.Fprintf(s.Err, "HTML comparison of %d runs written to %s\n", len(args), out)
				}
				return nil
			}
			var in io.Reader = s.In
			name := "stdin"
			if len(args) == 1 && args[0] != "-" {
				fh, err := os.Open(args[0])
				if err != nil {
					return usageErr(err)
				}
				defer fh.Close()
				in, name = fh, args[0]
			}
			if elive.LogReport == nil {
				return runtimeErr(fmt.Errorf("this build has no HTML report"))
			}
			if err := closeOut(elive.LogReport(in, opts, open)); err != nil {
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
	cmd.Flags().StringArrayVar(&labels, "label", nil, "a run's name in a comparison, one for each log, in their order")
	cmd.Flags().IntVar(&gpus, "gpus", 0, "GPUs that served each run, for throughput per GPU; over the runs' capacity.gpus")
	cmd.Flags().StringArrayVar(&resources, "resources", nil, "server resource samples, a CSV of timestamp,source,metric,value; repeat it. In a comparison LABEL=FILE, LABEL a run's --label")
	cmd.Flags().StringArrayVar(&per, "resources-per", nil, "throughput per CPU core counts the CPU sources whose name holds this text; repeat it. Every CPU source by default")
	cmd.Flags().StringArrayVar(&tiers, "tier", nil, "an SLO tier, "+tierFormat+"; repeat it, and the tiers replace the defaults (MLPerf's conversational and interactive)")
	return cmd
}

const tierFormat = "NAME:TTFT_MS:TPOT_MS (a limit of 0 is not checked)"

// parseTier reads NAME:TTFT_MS:TPOT_MS.
func parseTier(v string) (types.CapacityTier, error) {
	f := strings.Split(v, ":")
	if len(f) != 3 || strings.TrimSpace(f[0]) == "" {
		return types.CapacityTier{}, fmt.Errorf("--tier %q: the format is %s", v, tierFormat)
	}
	ft, err1 := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
	tp, err2 := strconv.ParseFloat(strings.TrimSpace(f[2]), 64)
	if err1 != nil || err2 != nil || ft < 0 || tp < 0 || (ft == 0 && tp == 0) {
		return types.CapacityTier{}, fmt.Errorf("--tier %q: TTFT_MS and TPOT_MS are milliseconds of 0 or more, not both 0", v)
	}
	return types.CapacityTier{Name: strings.TrimSpace(f[0]), FirstTokenMs: ft, TPOTMs: tp}, nil
}

// resourceArg reads a --resources value: a file for a single run, LABEL=FILE in a comparison, where
// LABEL is one of the runs' labels; the longest label that fits wins, so a label may hold a "=".
func resourceArg(v string, labels []string) (run, path string, err error) {
	if len(labels) < 2 {
		return "", v, nil
	}
	for _, l := range labels {
		if strings.HasPrefix(v, l+"=") && len(l) >= len(run) {
			run, path = l, v[len(l)+1:]
		}
	}
	if path == "" {
		return "", "", fmt.Errorf("--resources %q: in a comparison give LABEL=FILE, LABEL one of %q", v, labels)
	}
	return run, path, nil
}
