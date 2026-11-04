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
	"encoding/json"
	"fmt"
	"runtime"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newVersionCmd(info BuildInfo, s Streams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version, commit, build date and Go version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
				Date    string `json:"date"`
				Go      string `json:"go"`
				OSArch  string `json:"os_arch"`
			}{info.Version, info.Commit, info.Date, runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH}
			if asJSON {
				enc := json.NewEncoder(s.Out)
				enc.SetIndent("", "  ")
				return enc.Encode(v)
			}
			w := tabwriter.NewWriter(s.Out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Version:\t%s\nGit commit:\t%s\nBuilt:\t%s\nGo version:\t%s\nOS/Arch:\t%s\n", v.Version, v.Commit, v.Date, v.Go, v.OSArch)
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return cmd
}

func newDocsCmd(s Streams) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate the man pages or the Markdown reference of every command",
	}
	man := &cobra.Command{
		Use:   "man",
		Short: "Write man pages to --dir",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			if err := doc.GenManTree(root, &doc.GenManHeader{Title: "DDOSIFY", Section: "1", Source: "Ddosify"}, dir); err != nil {
				return runtimeErr(err)
			}
			fmt.Fprintln(s.Err, "Man pages written to", dir)
			return nil
		},
	}
	md := &cobra.Command{
		Use:   "markdown",
		Short: "Write the Markdown reference to --dir",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			if err := doc.GenMarkdownTree(root, dir); err != nil {
				return runtimeErr(err)
			}
			fmt.Fprintln(s.Err, "Markdown written to", dir)
			return nil
		},
	}
	for _, c := range []*cobra.Command{man, md} {
		c.Flags().StringVar(&dir, "dir", ".", "directory to write to; it must exist")
		cmd.AddCommand(c)
	}
	return cmd
}
