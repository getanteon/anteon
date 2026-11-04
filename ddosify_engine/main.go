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

package main

import (
	"os"

	"go.ddosify.com/ddosify/v2/cli"
)

// Set by the release build: -X main.GitVersion=... -X main.GitCommit=... -X main.BuildDate=...
var (
	GitVersion = "development"
	GitCommit  = "unknown"
	BuildDate  = "unknown"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], cli.BuildInfo{Version: GitVersion, Commit: GitCommit, Date: BuildDate},
		cli.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}))
}
