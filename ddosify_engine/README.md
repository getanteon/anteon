<div align="center">
    <img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon-logo-db.svg#gh-dark-mode-only" alt="Anteon logo dark" width="336px" /><br />
    <img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/anteon-logo-wb.svg#gh-light-mode-only" alt="Anteon logo light" width="336px" /><br />
</div>

<h1 align="center">Ddosify: A high-performance load testing tool</h1>

<p align="center">
    <a href="https://app.codecov.io/gh/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/codecov/c/github/ddosify/ddosify?style=for-the-badge&logo=codecov" alt="go coverage" /></a>&nbsp;
    <a href="https://goreportcard.com/report/github.com/getanteon/ddosify" target="_blank"><img src="https://goreportcard.com/badge/github.com/getanteon/ddosify?style=for-the-badge&logo=go" alt="go report" /></a>&nbsp;
    <a href="https://github.com/getanteon/anteon/blob/master/LICENSE" target="_blank"><img src="https://img.shields.io/badge/LICENSE-AGPL--3.0-orange?style=for-the-badge&logo=none" alt="ddosify license" /></a>
    <a href="https://discord.com/invite/9KdnrSUZQg" target="_blank"><img src="https://img.shields.io/discord/898523141788287017?style=for-the-badge&logo=discord&label=DISCORD" alt="ddosify discord server" /></a>
    <a href="https://hub.docker.com/r/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/docker/v/ddosify/ddosify?style=for-the-badge&logo=docker&label=docker&sort=semver" alt="ddosify docker image" /></a>
</p>

<p align="center">
<img src="https://raw.githubusercontent.com/getanteon/anteon/master/assets/ddosify-quick-start.gif" alt="Ddosify - High-performance load testing tool quick start" />
</p>

<details>
  <summary>Table of Contents</summary>

<!-- vim-markdown-toc GFM -->

- [Features](#features)
- [Tutorials / Blog Posts](#tutorials--blog-posts)
- [Installation](#installation)
  - [Docker](#docker)
  - [Docker Extension](#docker-extension)
  - [Homebrew Tap (macOS and Linux)](#homebrew-tap-macos-and-linux)
  - [Linux](#linux)
    - [Redhat (Fedora, CentOS, RHEL, etc.)](#redhat-fedora-centos-rhel-etc)
    - [Debian (Ubuntu, Linux Mint, etc.)](#debian-ubuntu-linux-mint-etc)
    - [Alpine](#alpine)
  - [FreeBSD](#freebsd)
  - [Windows Executable](#windows-executable)
  - [Using the convenience script (macOS and Linux)](#using-the-convenience-script-macos-and-linux)
  - [Go install from source (macOS, FreeBSD, Linux, Windows)](#go-install-from-source-macos-freebsd-linux-windows)
- [Quick Start](#quick-start)
- [Advanced Usage](#advanced-usage)
  - [CLI Flags](#cli-flags)
  - [Load Types](#load-types)
    - [Linear](#linear)
    - [Incremental](#incremental)
    - [Waved](#waved)
  - [Configuration](#configuration)
- [Parameterization (Dynamic Variables)](#parameterization-dynamic-variables)
  - [Parameterization on URL](#parameterization-on-url)
  - [Parameterization on Headers](#parameterization-on-headers)
  - [Parameterization on Payload (Body)](#parameterization-on-payload-body)
  - [Parameterization on Basic Authentication](#parameterization-on-basic-authentication)
  - [Parameterization on Config File](#parameterization-on-config-file)
  - [Environment Variables](#environment-variables)
- [Assertion](#assertion)
  - [Keywords](#keywords)
  - [Functions](#functions)
  - [Operators](#operators)
  - [Assertion Examples](#assertion-examples)
- [Success Criteria (Pass / Fail)](#success-criteria-pass--fail)
- [Difference Between Success Criteria and Step Assertions](#difference-between-success-criteria-and-step-assertions)
  - [Keywords](#keywords-1)
  - [Functions](#functions-1)
  - [Examples](#examples)
- [Correlation](#correlation)
  - [Capture with json_path](#capture-with-json_path)
  - [Capture with XPath on XML](#capture-with-xpath-on-xml)
  - [Capture with XPath on HTML](#capture-with-xpath-on-html)
  - [Capture with Regular Expressions](#capture-with-regular-expressions)
  - [Capture Header Value](#capture-header-value)
  - [Scenario-Scoped Variables](#scenario-scoped-variables)
  - [Overall Config and Injection](#overall-config-and-injection)
- [Test Data Set](#test-data-set)
- [Cookies](#cookies)
  - [Initial / Custom Cookies](#initial--custom-cookies)
  - [Cookie Capture](#cookie-capture)
  - [Cookie Assertion](#cookie-assertion)
- [Upgrading from v1](#upgrading-from-v1)
- [Verifying a release](#verifying-a-release)
- [Streaming (SSE) and LLM Endpoints](#streaming-sse-and-llm-endpoints)
  - [Stream config](#stream-config)
  - [What is measured](#what-is-measured)
  - [Goodput and SLO](#goodput-and-slo)
  - [Stepped arrival rate and stages](#stepped-arrival-rate-and-stages)
  - [Example: an AI SDK chat API](#example-an-ai-sdk-chat-api)
  - [Tokens, the OpenAI preset, sessions and prefix caching](#tokens-the-openai-preset-sessions-and-prefix-caching)
- [HTML report](#html-report)
- [Common Issues](#common-issues)
  - [macOS Security Issue](#macos-security-issue)
  - [OS Limit - Too Many Open Files](#os-limit---too-many-open-files)
- [Contributing](#contributing)
- [Communication](#communication)
- [More](#more)
- [Disclaimer](#disclaimer)
- [License](#license)

<!-- vim-markdown-toc -->

</details>

- [📥 Installation](https://getanteon.com/docs/ddosify/installation/)
- [🚀 Quickstart Guide](https://getanteon.com/docs/ddosify/quickstart/)
- [⚙️ Configuration](https://getanteon.com/docs/ddosify/configuration/)
- [✨ Examples](https://getanteon.com/docs/ddosify/examples/)

- ✅ **[Scenario-Based](#config-file)** - Create your flow in a JSON file. Without a line of code!

- ✅ **[Different Load Types](#load-types)** - Test your system's limits across different load types.

- ✅ **[Parameterization](#parameterization-dynamic-variables)** - Use dynamic variables just like on Postman.

- ✅ **[Correlation](#correlation)** - Extract variables from earlier phases and pass them on to the following ones.

- ✅ **[Test Data](#test-data-set)** - Import test data from CSV and use it in the scenario.

- ✅ **[Assertion](#assertion)** - Verify that the response matches your expectations.

- ✅ **[Success Criteria](#success-criteria-pass--fail)** - Set the success criteria for your test.

- ✅ **[Cookies](#cookies)** - Pass cookies through steps and set initial cookies if you want.

- ✅ **[Streaming (SSE) and LLM Endpoints](#streaming-sse-and-llm-endpoints)** - Time to first token, gaps between events, goodput against an SLO, and a report per load stage.

- ✅ **Widely Used Protocols** - Currently supporting _HTTP, HTTPS, HTTP/2_. Other protocols are on the way.

## Tutorials / Blog Posts

- [Testing the Performance of User Authentication Flow](https://getanteon.com/blog/testing-the-performance-of-user-authentication-flow)
- [Load Testing a Fintech API with CSV Test Data Import](https://getanteon.com/blog/load-testing-a-fintech-exchange-api-with-csv-test-data-import)

## Installation

`ddosify` is available via [Docker](https://hub.docker.com/r/ddosify/ddosify), [Docker Extension](https://hub.docker.com/extensions/ddosify/ddosify-docker-extension), [Homebrew Tap](#homebrew-tap-macos-and-linux), and downloadable as pre-compiled binaries from the [releases page](https://github.com/getanteon/anteon/releases/latest) for macOS, Linux and Windows.

For shell auto completions, see [Ddosify Completions](https://github.com/getanteon/anteon/tree/master/ddosify_engine/completions).

### Docker

```bash
docker run --rm ddosify/ddosify run https://getanteon.com
docker run --rm -v "$PWD:/cfg:ro" ddosify/ddosify run --config /cfg/config.json
```

The image holds only the static binary on a distroless base, running as a non-root user, with `ddosify` as its entrypoint. Images are signed with keyless cosign and attested; see [Verifying a release](#verifying-a-release).

### Docker Extension

Run Ddosify on Docker Desktop with Ddosify Docker extension. More details [here](https://hub.docker.com/extensions/ddosify/ddosify-docker-extension).

### Homebrew Tap (macOS and Linux)

```bash
brew install ddosify/tap/ddosify
```

### Linux

- For ARM architectures change `ddosify_amd64` to `ddosify_arm64` or `ddosify_armv6`.
- Superuser privilege is required.

#### Redhat (Fedora, CentOS, RHEL, etc.)

```bash
rpm -i https://github.com/ddosify/ddosify/releases/download/v1.0.6/ddosify_amd64.rpm
```

#### Debian (Ubuntu, Linux Mint, etc.)

```bash
wget https://github.com/ddosify/ddosify/releases/download/v1.0.6/ddosify_amd64.deb
dpkg -i ddosify_amd64.deb
```

#### Alpine

```bash
wget https://github.com/ddosify/ddosify/releases/download/v1.0.6/ddosify_amd64.apk
apk add --allow-untrusted ddosify_amd64.apk
```

### FreeBSD

```bash
pkg install ddosify
```

### Windows Executable

- Download zip file for your architecture from the [releases page](https://github.com/ddosify/ddosify/releases/latest).
  - For example, download ddosify version `vx.x.x` with amd64 architecture: `ddosify_x.x.x.zip_windows_amd64`
- Unzip `ddosify_x.x.x_windows_amd64.zip`
- Open Powershell or CMD (Command Prompt) and change directory to unzipped folder: `ddosify_x.x.x_windows_amd64`
- Run ddosify:

```bash
.\ddosify.exe -t https://getanteon.com
```

### Using the convenience script (macOS and Linux)

- The script requires root or sudo privileges to move ddosify binary to `/usr/local/bin`.
- The script attempts to detect your operating system (macOS or Linux) and architecture (arm64, x86, amd64) to download the appropriate binary from the [releases page](https://github.com/getanteon/anteon/tree/master/ddosify_engine/completions).
- By default, the script installs the latest version of `ddosify`.
- If you have problems, check [common issues](#common-issues).
- Required packages: `curl` and `sudo`

```bash
curl -sSfL https://raw.githubusercontent.com/getanteon/anteon/master/scripts/install.sh | sh
```

### Go install from source (macOS, FreeBSD, Linux, Windows)

_Requires Go 1.26 (the module pins toolchain go1.26.10)_

```bash
go install -v go.ddosify.com/ddosify/v2@latest
```

## Quick Start

This section aims to show you how to use Ddosify easily without deep dive into its details.

1.  ### Simple load test

    `ddosify run https://getanteon.com`

    The above command runs a load test with the default value that is 100 requests in 10 seconds.

2.  ### Using some of the features

    `ddosify run https://getanteon.com -n 1000 -d 20 -m PUT -T 7 -P http://proxy_server.com:80`

    Ddosify sends a total of _1000_ _PUT_ requests to *https://getanteon.com* over proxy _http://proxy_server.com:80_ in _20_ seconds with a timeout of _7_ seconds per request.

3.  ### Usage for CI/CD pipelines (JSON output)

    `ddosify run https://getanteon.com -o stdout-json | jq .avg_duration`

    Ddosify outputs the result in JSON format. Then `jq` (or any other command-line JSON processor) fetches the `avg_duration`. The rest depends on your CI/CD flow logic.

4.  ### Scenario based load test

    `ddosify run --config config_examples/config.json`

    Ddosify first sends _HTTP/2 POST_ request to *https://getanteon.com/endpoint_1* using basic auth credentials _test_user:12345_ over proxy _http://proxy_host.com:proxy_port_ and with a timeout of _3_ seconds. Once the response is received, HTTPS GET request will be sent to *https://getanteon.com/endpoint_2* along with the payload included in _config_examples/payload.txt_ file with a timeout of 2 seconds. This flow will be repeated _20_ times in _5_ seconds and response will be written to _stdout_.

5.  ### Load test with Dynamic Variables (Parameterization)

    `ddosify run https://getanteon.com/{{_randomInt}} -d 10 -n 100 -H 'User-Agent: {{_randomUserAgent}}' -b '{"city": "{{_randomCity}}"}'`

    Ddosify sends a total of _100_ _GET_ requests to `https://getanteon.com/{{_randomInt}}` in _10_ seconds. `{{_randomInt}}` path generates random integers between 1 and 1000 in every request. Dynamic variables can be used in _URL_, _headers_, _payload (body)_ and _basic authentication_. In this example, Ddosify generates a random user agent in the header and a random city in the body. The full list of the dynamic variables can be found in the [docs](https://getanteon.com/docs/performance-testing/dynamic-variables-parametrization/).

6.  ### Correlation (Captured Variables)

    `ddosify run --config ddosify_config_correlation.json`

    Ddosify allows you to specify variables at the global level and use them throughout the scenario, as well as extract variables from previous steps and inject them to the next steps in each iteration individually. You can inject those variables in requests _url_, _headers_ and _payload(body)_. The example config can be found in [correlation-config-example](#Correlation).

7.  ### Test Data

    `ddosify run --config ddosify_data_csv.json`

    Ddosify allows you to load test data from a file, tag specific columns for later use. You can inject those variables in requests _url_, _headers_ and _payload (body)_. The example config can be found in [test-data-example](#test-data-set).

## Advanced Usage

You can configure your load test by the CLI options or a config file. Config file supports more features than the CLI. For example, you can't create a scenario-based load test with CLI options.

### Commands

| Command | What it does |
| ------- | ------------ |
| `ddosify run [TARGET_URL]` | Runs a load test, from `--config FILE` or from a target URL and flags |
| `ddosify report [LOG...] [-o FILE] [--title T]` | Writes the [HTML report](#html-report) of a finished run from its `--json` output; `-o -` writes it to stdout. With two logs or more, a page comparing the runs (`--label`, `--gpus`, `--tier`) |
| `ddosify version [--json]` | Prints the version, commit, build date and Go version |
| `ddosify completion bash\|zsh\|fish\|powershell` | Prints a shell completion script |
| `ddosify docs man\|markdown --dir DIR` | Writes the man pages or the Markdown reference of every command |

`ddosify -t URL ...` and `ddosify -config FILE` from v1 still run a test.

### CLI Flags

```bash
ddosify run [TARGET_URL] [FLAGS]
```

| Flag | Description | Default |
| ---- | ----------- | ------- |
| `-f`, `--config` | [Config file](#config-file) of the load test | - |
| `-t`, `--target` | Target URL, when there is no config file; it can also be the argument | - |
| `-n`, `--iterations` | Total iteration count | `100` |
| `-d`, `--duration` | Test duration in seconds | `10` |
| `--stage` | A [`manual_load`](#configuration) stage, `DURATION:COUNT[:NAME[:warmup]]`: `60s:4:warm-up:warmup`, `4m:15:6/min`, or a closed-loop one, `c=CONCURRENCY:COUNT[:NAME[:warmup]]`: `c=16:160`. Repeat it; the stages replace the config's `manual_load`, and the iterations and duration follow from them. `DURATION` is whole seconds, `COUNT` the iterations started in the stage, `NAME` can not hold a `:` | - |
| `--concurrency` | Closed-loop stages, one per level: `1,2,4,8,16,32,64,128`, each named `c=N`; they replace the config's `manual_load` | - |
| `--per-level` | With `--concurrency`: iterations of each level; `0` is max(10 x level, 20) | `0` |
| `--warmup-iterations` | With `--concurrency`: a warm-up stage of this many iterations at the first level, left out of the results | `0` |
| `--gpus` | GPUs serving the model, for the HTML report's throughput per GPU (`capacity.gpus`) | `1` |
| `-l`, `--load-type` | [Load type](#load-types): `linear`, `incremental`, `waved` | `linear` |
| `--load-spread` | `front`, `even` or `poisson`, see `load_spread` | `front` |
| `--seed` | Seeds the `poisson` arrivals and the random data rows, see `seed` | `1` |
| `--engine-mode` | `ddosify`, `distinct-user`, `repeated-user` | `ddosify` |
| `-m`, `--method` | Request method: _GET, POST, PUT, DELETE, HEAD, PATCH, OPTIONS_ | `GET` |
| `-b`, `--body` | Request body | - |
| `-a`, `--auth` | Basic authentication, `username:password` | - |
| `-H`, `--header` | Request header, `-H 'Accept: text/html'`; repeat it. With a config it is added to every step | - |
| `-T`, `--timeout` | Request timeout in seconds; for a stream it covers the whole stream | `5` |
| `-P`, `--proxy` | Proxy, `http://user:pass@proxy_host.com:port` | - |
| `-o`, `--output` | `stdout` or `stdout-json` | `stdout` |
| `--json` | NDJSON on stdout, the same as `-o stdout-json`: a `"type": "progress"` line every second, a `"type": "request"` line per request with `request_log`, and last the report, one line with `"type": "report"`. Stderr is for a person: what the run is doing every 10 seconds, the first error of each kind the moment it happens, and how it ended. See [Seeing what a run does](#seeing-what-a-run-does) | `false` |
| `--debug` | Runs one iteration and prints every request and response | `false` |
| `--cert-path`, `--cert-key-path` | Client certificate and key files | - |
| `--insecure` | Does not verify the target's TLS certificate. Off by default: certificates are verified | `false` |
| `--preset` | Applies a registered config preset; repeat it | - |
| `--html` | Also writes the [HTML report](#html-report) to this file, when the build has the HTML report | - |
| `--observer` | Adds a registered live observer, `name` or `name=arg` | - |
| `--ui` | `auto` shows the live TUI on a terminal and ndjson snapshots when stdout is not a terminal or with `--json`, when the build has them; `tui`, `ndjson` or `plain` choose one. `TERM=dumb` never gets the TUI | `auto` |
| `--no-color` | No colors; `NO_COLOR` and `TERM=dumb` do the same | `false` |
| `--drain` | After an interrupt, how long iterations in flight may finish before they are cut; a second interrupt cuts them now | `30s` |
| `--tokenizer` | A Hugging Face `tokenizer.json` that counts the streamed answers' tokens; see [docs/llm.md](docs/llm.md) | - |
| `--url`, `--model`, `--dataset`, `--api-key`, `--max-tokens` | With `--preset openai`: an OpenAI compatible chat completions test without a config file | - |
| `--input-tokens`, `--ignore-eos` | With `--preset openai`: every prompt exactly this many tokens of the dataset's text, and every answer run to `--max-tokens`; see [docs/llm.md](docs/llm.md#exact-input-lengths-and-concurrency-sweeps) | - |

#### Seeing what a run does

With `--json` the data goes to stdout and a person's view of the run to stderr, so `> run.ndjson` keeps the data in a file and the terminal, or `docker logs`, still says what is happening:

```text
Started: 3 iterations over 1:00; stages: smoke 3 in 1:00; steps: 1 "Create chat", 2 "Turn 1"
Error: step 1 "Create chat": network_error: Post "https://gurubase.example.internal/api/v1beta/gurus/.../chats": tls: failed to verify certificate: x509: certificate signed by unknown authority
0:10/1:00 smoke · started 1 · in flight 0 · ok 0 · failed 1 (network_error 1)
Finished in 1:01: 3 iterations started, requests ok 0, failed 3 (network_error 3); the test failed: no request succeeded; the first error of each kind is above
```

- **An error is shown when it first happens**, once per step, outcome and reason, with the reason in full: a refused certificate, a timeout, an `HTTP 503`. A network error shows the method, the URL and the cause (`dial tcp 10.0.0.1:443: connect: connection refused`); a name that does not resolve says `lookup`, also when the resolver timed out; an HTTP error shows the first line of the response body (`HTTP 401: {"error": "Invalid API key"}`, at most 200 characters). A URL in the reason does not make a new kind, so an id in the path does not repeat it. Twenty kinds at most; the report counts them all.
- **A config that needs an environment variable** (`{{$NAME}}`) that is not set stops before anything is sent, naming every missing one. In a container, pass each with `-e NAME=value`; `-e NAME` alone passes it only when it is exported in the shell that runs `docker`.
- **The exit code** says how it ended: `0` passed, `2` the command or the config is wrong, `3` a success criterion failed, for example no request succeeded.

A pod's log holds both streams; `ddosify report` and other NDJSON readers skip the lines that are not JSON.

**Precedence.** A flag wins over its environment variable, `DDOSIFY_` and the flag's name in capitals with `_` for `-` (`DDOSIFY_ITERATIONS`, `DDOSIFY_INSECURE`, `DDOSIFY_JSON`), which wins over the config file. `--target`, `--method`, `--body`, `--auth` and the certificate flags describe a single request and are refused with `--config`; `--iterations` and `--duration` are refused with a `manual_load` config or with `--stage` or `--concurrency`.

**Output.** The report goes to stdout, banners and errors to stderr. With `--json`, progress lines are data and go to stdout too, and stderr gets the status lines above; without it, progress lines go to stderr. Colors only when stdout is a terminal.

**Exit codes.**

| Code | Meaning |
| ---- | ------- |
| `0` | The test ran and passed |
| `1` | The test ran but its output could not be written, such as the `--html` file, or the engine failed; the report on stdout is still printed |
| `2` | A flag, an argument or the config is wrong, or a file it names (the config, a CSV, a dataset, a tokenizer, the `--html` path) can not be read or written; nothing was sent |
| `3` | The test ran and failed: a [success criterion](#success-criteria-pass--fail), the SLO's `goodput_perc`, or no iteration succeeded (every one refused, failed or cut) |
| `130` | Interrupted: the iterations finished until then are reported, those still in flight after `--drain` are counted as interrupted |

**Interrupting.** The first CTRL+C (or SIGTERM, or `q` in the TUI) stops starting iterations and gives the ones in flight `--drain` (30 s by default) to finish; a second one cuts them at once. Finished iterations are reported as usual. Cut ones are counted as `interrupted`, apart from every other count and from the latencies.

### Load Types

#### Linear

```bash
ddosify run https://getanteon.com -l linear
```

Result:

![linear load](https://raw.githubusercontent.com/getanteon/anteon/master/assets/linear.gif)

_Note:_ If the iteration count is too low for the given duration, the test might be finished earlier than you expect.

#### Incremental

```bash
ddosify run https://getanteon.com -l incremental
```

Result:

![incremental load](https://raw.githubusercontent.com/getanteon/anteon/master/assets/incremental.gif)

#### Waved

```bash
ddosify run https://getanteon.com -l waved
```

Result:

![waved load](https://raw.githubusercontent.com/getanteon/anteon/master/assets/waved.gif)

### Configuration

Configuration file lets you use all the capabilities of Ddosify.

The features you can use by config file:

- Scenario creation
- Environment variables
- Correlation
- Assertions
- Cookies
- Custom load type creation
- Streaming (SSE) responses, goodput and per-stage reports
- Payload from a file
- Multipart/form-data payload
- Extra connection configuration
- HTTP2 support

Usage:

```bash
ddosify run --config <json_config_path>
```

There is an example config file at [config_examples/config.json](https://github.com/getanteon/anteon/blob/master/ddosify_engine/config_examples/config.json). This file contains all of the parameters you can use. Details of each parameter;

- `iteration_count` (_optional_)

  This is the equivalent of the `-n` flag. The difference is that if you have multiple steps in your scenario, this value represents the iteration count of the steps.

- `load_type` (_optional_)

  This is the equivalent of the `-l` flag.

- `duration` (_optional_)

  This is the equivalent of the `-d` flag.

- `manual_load` (_optional_)

  If you are looking for creating your own custom load type, you can use this feature. The example below says that Ddosify will run the scenario 5 times, 10 times, and 20 times, respectively along with the provided durations. `iteration_count` and `duration` will be auto-filled by Ddosify according to `manual_load` configuration. In this example, `iteration_count` will be 35 and the `duration` will be 18 seconds.
  Also `manual_load` overrides `load_type` if you provide both of them. As a result, you don't need to provide these 3 parameters when using `manual_load`. `--stage` on the command line replaces the stages without editing the file.

  ```json
  "manual_load": [
      {"duration": 5, "count": 5},
      {"duration": 6, "count": 10},
      {"duration": 7, "count": 20}
  ]
  ```

  A stage can also carry a `name` and `"warmup": true`. Once any stage does, the report has a row per stage, and a warm-up stage is left out of the stream summary, the outcomes and the goodput. See [Stepped arrival rate and stages](#stepped-arrival-rate-and-stages).

  A stage with `concurrency` is closed-loop: it runs `count` iterations with at most `concurrency` in flight, a new one starting as soon as one ends, and ends when they all have, as `vllm bench serve` does with `--max-concurrency` and `--num-prompts`. Its `duration` is optional and caps it: past it no iteration starts and those in flight finish. The stages run one after the other, and all of them are closed-loop or none is; `load_spread` does not apply. A closed-loop stage is always reported on its own row.

  ```json
  "manual_load": [
      {"concurrency": 1, "count": 4, "name": "warm-up", "warmup": true},
      {"concurrency": 1, "count": 20, "name": "c=1"},
      {"concurrency": 16, "count": 160, "name": "c=16"},
      {"concurrency": 64, "count": 640, "name": "c=64", "duration": 900}
  ]
  ```

- `capacity` (_optional_)

  How the HTML report reads the stages as capacity: `gpus`, the GPUs serving the model, divides the throughput, and `tiers` are the SLOs a stage is checked against, each `{"name", "first_token_ms", "tpot_ms"}` on the 99th percentile. The default tiers are MLPerf Inference's for Llama3.1-8B: conversational, TTFT 2,000 ms and TPOT 100 ms, and interactive, 500 ms and 30 ms. See [Exact input lengths and concurrency sweeps](docs/llm.md#exact-input-lengths-and-concurrency-sweeps).

  ```json
  "capacity": {"gpus": 1, "tiers": [{"name": "interactive", "first_token_ms": 500, "tpot_ms": 30}]}
  ```

- `load_spread` (_optional_)

  How iterations are placed in time. `front` (default) is the behaviour Ddosify always had: each second's share is started at the first ticks of that second, and with fewer iterations than seconds, in the first seconds of the test or `manual_load` stage. `even` starts them at even intervals across the stage instead: 30 iterations over 300 seconds start one every 10 seconds. Use `even` for a low arrival rate, such as an LLM endpoint at a few questions a minute. `poisson` starts exactly `count` iterations in each stage at random times, as independent users arrive: see [Poisson arrivals](docs/llm.md#poisson-arrivals-and-the-seed).

- `seed` (_optional_)

  An integer that seeds the `poisson` arrivals and the rows of a data set read in `random` order, so a run can be repeated. Default `1`.

- `stop_on_failure` (_optional_)

  `true` ends an iteration at its first step whose outcome is not `ok`: its remaining steps are not sent and not counted. Off by default.

- `request_log` (_optional_)

  `true` writes a `request` line per sent step with `-o stdout-json`, before the report line. See [Per-request records](docs/llm.md#per-request-records).

- `slo` (_optional_)

  Thresholds in milliseconds for the goodput, the share of iterations that succeeded and whose streams met every threshold. Keys: `ttfb_ms`, `first_event_ms`, `first_token_ms`, `total_ms`, `max_gap_ms`, and `goodput_perc`, below which the test fails.

- `insecure` (_optional_)

  `true` skips TLS certificate verification for every step; a step can set it alone. Off by default.

- `presets` (_optional_)

  A preset name, or a list of them, registered by a feature package, applied to this config before it is read. `--preset` adds more. See [Goodput and SLO](#goodput-and-slo).

- `proxy` (_optional_)

  This is the equivalent of the `-P` flag.

- `output` (_optional_)

  This is the equivalent of the `-o` flag.

- `engine_mode` (_optional_)

  Can be one of `distinct-user`, `repeated-user`, or default mode `ddosify`.

  - `distinct-user` mode simulates a new user for every iteration.
  - `repeated-user` mode can use pre-used user in subsequent iterations.
  - `ddosify` mode is default mode of the engine. In this mode engine runs in its max capacity, and does not show user simulation behaviour.

- `env` (_optional_)

  Scenario-scoped global variables. Note that dynamic variables changes every iteration.

  ```json
  "env": {
          "COMPANY_NAME" :"Ddosify",
          "randomCountry" : "{{_randomCountry}}"
  }
  ```

- `data` (_optional_)

  Config for loading test data from a CSV file.
  [CSV data](https://github.com/getanteon/anteon/blob/master/ddosify_engine/config/config_testdata/test.csv) used in below config.

  ```json
  "data":{
      "info": {
          "path" : "config/config_testdata/test.csv",
          "delimiter": ";",
          "vars": {
                  "0":{"tag":"name"},
                  "1":{"tag":"city"},
                  "2":{"tag":"team"},
                  "3":{"tag":"payload", "type":"json"},
                  "4":{"tag":"age", "type":"int"}
                  },
          "allow_quota" : true,
          "order": "sequential",
          "skip_first_line" : true,
          "skip_empty_line" : true
      }
  }
  ```

  | Field             | Description                                                                                                                                                                               | Type     | Default  | Required? |
  | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | -------- | --------- |
  | `path`            | Local path or remote url for your CSV file                                                                                                                                                | `string` | -        | Yes       |
  | `delimiter`       | Delimiter for reading CSV                                                                                                                                                                 | `string` | `,`      | No        |
  | `vars`            | Tag columns using column index as key, use `type` field if you want to cast a column to a specific type, default is `string`, can be one of the following: `json`, `int`, `float`,`bool`. | `map`    | -        | Yes       |
  | `allow_quota`     | If set to true, a quote may appear in an unquoted field and a non-doubled quote may appear in a quoted field                                                                              | `bool`   | `false`  | No        |
  | `order`           | Order of reading records from CSV. Can be `random` or `sequential`                                                                                                                        | `string` | `random` | No        |
  | `skip_first_line` | Skips first line while reading records from CSV.                                                                                                                                          | `bool`   | `false`  | No        |
  | `skip_empty_line` | Skips empty lines while reading records from CSV.                                                                                                                                         | `bool`   | `true`   | No        |

- `success_criterias` (_optional_)

  Config for pass fail logic for the test. _abort_ and _delay_ fields can be used to adjust the abort behaviour in case of failure. If abort is true for a rule and rules fails at certain point, engine will decide to abort test immediately if delay is 0 or not given. If delay is given, it will wait for delay seconds and reassert the rule.

  **Example:** Check _90th percentile_ and _fail_count_;

  ```json
  {
   "duration": 10,
   <other_global_configurations>,
   "success_criterias": [
    {
      "rule" : "p90(iteration_duration) < 220",
      "abort" : false
    },
    {
      "rule" : "fail_count_perc < 0.1",
      "abort" : true,
      "delay" : 1
    },
    {
      "rule" : "fail_count < 100",
      "abort" : true,
      "delay" : 0
    }
  ],
   "steps": [....]
  }
  ```

- `steps` (_required_)

  This parameter lets you create your scenario. Ddosify runs the provided steps, respectively. For the given example file step id: 2 will be executed immediately after the response of step id: 1 is received. The order of the execution is the same as the order of the steps in the config file.

  **Details of each parameter for a step;**

  - `id` (_required_)

    Each step must have a unique integer id.

  - `url` (_required_)

    This is the equivalent of the `-t` flag.

  - `name` (_optional_) <a name="#step-name"></a>

    Name of the step.

  - `method` (_optional_)

    This is the equivalent of the `-m` flag.

  - `headers` (_optional_)

    List of headers with key:value format.

  - `payload` (_optional_)

    Body or payload. This is the equivalent of the `-b` flag.

    _Note:_ If you want to use `x-www-form-urlencoded`, set Content-Type header to `application/x-www-form-urlencoded`.

    **Example:** send `x-www-form-urlencoded` data;

    ```json
    {
      "headers": {
        "Content-Type": "application/x-www-form-urlencoded"
      },
      "payload": "key1=value1&key2=value2"
    }
    ```

  - `payload_file` (_optional_)

    If you need a long payload, we suggest using this parameter instead of `payload`.

  - `payload_multipart` (_optional_) <a name="#payload_multipart"></a>

    Use this for `multipart/form-data` Content-Type.

    Accepts list of `form-field` objects, structured as below;

    ```json
    {
        "name": [field-name],
        "value": [field-value|file-path|url],
        "type": <text|file>,    // Default "text"
        "src": <local|remote>   // Default "local"
    }
    ```

    **Example:** Sending form name-value pairs;

    ```json
    "payload_multipart": [
        {
            "name": "[field-name]",
            "value": "[field-value]"
        }
    ]
    ```

    **Example:** Sending form name-value pairs and a local file;

    ```json
    "payload_multipart": [
        {
            "name": "[field-name]",
            "value": "[field-value]",
        },
        {
            "name": "[field-name]",
            "value": "./test.png",
            "type": "file"
        }
    ]
    ```

    **Example:** Sending form name-value pairs and a local file and a remote file;

    ```json
    "payload_multipart": [
        {
            "name": "[field-name]",
            "value": "[field-value]",
        },
        {
            "name": "[field-name]",
            "value": "./test.png",
            "type": "file"
        },
        {
            "name": "[field-name]",
            "value": "http://getanteon.com/test.png",
            "type": "file",
            "src": "remote"
        }
    ]
    ```

    _Note:_ Ddosify adds `Content-Type: multipart/form-data; boundary=[generated-boundary-value]` header to the request when using `payload_multipart`.

  - `timeout` (_optional_)

    This is the equivalent of the `-T` flag.

  - `capture_env` (_optional_)

    Config for extraction of variables to use them in next steps.
    **Example:** Capture _NUM_ variable from steps response body;

    ```json
    "steps": [
        {
            "id": 1,
            "url": "http://getanteon.com/endpoint1",
            "capture_env": {
                 "NUM" :{"from":"body","json_path":"num"},
            }
        },
    ]
    ```

  - `assertion` (_optional_)

    The response from this step will be subject to the assertion rules. If one of the provided rules fails, step is considered as failure.
    **Example:** Check _status code_ and _content-length_ header values;

    ```json
    "steps": [
        {
            "id": 1,
            "url": "http://getanteon.com/endpoint1",
            "assertion": [
                "equals(status_code,200)",
                "in(headers.content-length,[2000,3000])"
            ]
        },
    ]
    ```

  - `stream` (_optional_)

    Reads the response as server-sent events instead of reading it whole, and measures the stream. A `text/event-stream` response is read this way even without it. See [Stream config](#stream-config).

    ```json
    "stream": {"type": "sse"}
    ```

  - `sleep` (_optional_) <a name="#sleep"></a>

    Sleep duration(ms) before executing the next step. Can be an exact duration or a range.

    **Example:** Sleep 1000ms after step-1;

    ```json
    "steps": [
        {
            "id": 1,
            "url": "http://getanteon.com/endpoint1",
            "sleep": "1000"
        },
        {
            "id": 2,
            "url": "http://getanteon.com/endpoint2",
        }
    ]
    ```

    **Example:** Sleep between 300ms-500ms after step-1;

    ```json
    "steps": [
        {
            "id": 1,
            "url": "http://getanteon.com/endpoint1",
            "sleep": "300-500"
        },
        {
            "id": 2,
            "url": "http://getanteon.com/endpoint2",
        }
    ]
    ```

    The sleep is applied only when a later step of the iteration is sent: never after the last one, nor before steps that `skip_if_empty` or `stop_on_failure` leave out.

  - `skip_if_empty` (_optional_)

    A template rendered with the iteration's variables, like `url` and `payload`. When the result is blank the step is not sent, its `sleep` is not applied, and it is counted nowhere; the next steps still run. A variable that is not defined, such as a capture that failed, counts as blank.

    ```json
    "skip_if_empty": "{{data.sessions.turn2}}"
    ```

  - `auth` (_optional_)

    Basic authentication.

    ```json
    "auth": {
        "username": "test_user",
        "password": "12345"
    }
    ```

  - `others` (_optional_)

    This parameter accepts dynamic _key: value_ pairs to configure connection details of the protocol in use.

    ```json
    "others": {
        "disable-compression": false,    // Default true
        "h2": true,                      // Enables HTTP/2. Default false.
        "disable-redirect": true         // Default false
    }
    ```

## Parameterization (Dynamic Variables)

Just like the Postman, Ddosify supports parameterization (dynamic variables) on _URL_, _headers_, _payload (body)_ and _basic authentication_. Actually, we support all the random methods Postman supports. If you use `{{$randomVariable}}` on Postman you can use it as `{{_randomVariable}}` on Ddosify. Just change `$` to `_` and you will be fine. To simulate a realistic load test on your system, Ddosify can send every request with dynamic variables.

The full list of dynamic variables can be found in the [documentation](https://getanteon.com/docs/performance-testing/dynamic-variables-parametrization/).

### Parameterization on URL

Ddosify sends _100_ GET requests in _10_ seconds with random string `key` parameter. This approach can be also used in cache bypass.

```bash
ddosify run https://getanteon.com/?key={{_randomString}} -d 10 -n 100
```

### Parameterization on Headers

Ddosify sends _100_ GET requests in _10_ seconds with random `Transaction-Type` and `Country` headers.

```bash
ddosify run https://getanteon.com -d 10 -n 100 -H 'Transaction-Type: {{_randomTransactionType}}' -H 'Country: {{_randomCountry}}'
```

### Parameterization on Payload (Body)

Ddosify sends _100_ GET requests in _10_ seconds with random `latitude` and `longitude` values in body.

```bash
ddosify run https://getanteon.com -d 10 -n 100 -b '{"latitude": "{{_randomLatitude}}", "longitude": "{{_randomLongitude}}"}'
```

### Parameterization on Basic Authentication

Ddosify sends _100_ GET requests in _10_ seconds with random `username` and `password` with basic authentication.

```bash
ddosify run https://getanteon.com -d 10 -n 100 -a '{{_randomUserName}}:{{_randomPassword}}'
```

### Parameterization on Config File

Dynamic variables can be used on config file as well. Ddosify sends _100_ GET requests in _10_ seconds with random string `key` parameter in URL and random `User-Key` header.

```bash
ddosify run --config ddosify_config_dynamic.json
```

```json
{
  "iteration_count": 100,
  "load_type": "linear",
  "duration": 10,
  "steps": [
    {
      "id": 1,
      "url": "https://getanteon.com/?key={{_randomString}}",
      "method": "POST",
      "headers": {
        "User-Key": "{{_randomInt}}"
      }
    }
  ]
}
```

### Environment Variables

In addition, you can also use operating system environment variables. To access these variables, simply add the `$` prefix followed by the variable name wrapped in double curly braces. The syntax for this is `{{$OS_ENV_VARIABLE}}` within the **config file**.

For instance, to use the `USER` environment variable from your operating system, simply input `{{$USER}}`. You can use operating system environment variables in `URL`, `Headers`, `Body (Payload)`, and `Basic Authentication`.

Here is an example of using operating system environment variables in the config file. `TARGET_SITE` operating system environment variable is used in `URL` and `USER` environment variable is used in `Headers`.

```bash
export TARGET_SITE="https://getanteon.com"
ddosify run --config ddosify_config_os_env.json
```

```json
{
  "iteration_count": 100,
  "load_type": "linear",
  "duration": 10,
  "steps": [
    {
      "id": 1,
      "url": "{{$TARGET_SITE}}",
      "method": "POST",
      "headers": {
        "os-env-user": "{{$USER}}"
      }
    }
  ]
}
```

## Assertion

By default, Ddosify marks a step result as successful if it sends the request and receives the response without any network errors. Status code or body type (or content) does not affect the success/failure criteria. However, this may not provide a good test result for your use case, and you may want to create your own success/fail logic. That's where Assertions come in.

Ddosify supports assertions on `status code`, `response body`, `response size`, `response time`, `headers`, and `variables`. You can use the `assertion` parameter in the config file to check if the response matches the given condition per step. If the condition is not met, Ddosify will fail the step. Check the [example config](https://github.com/getanteon/anteon/blob/master/ddosify_engine/config_examples/config.json) to see how it looks.

As shown in the related table, the first five keywords store different data related to the response. The last keyword, `variables`, stores the current state of environment variables for the step. You can use [Functions](#functions) or [Operators](#operators) to build conditional expressions based on these keywords.

You can write multiple assertions for a step. If any assertion fails, the step is marked as failed.

If Ddosify can't receive the response for a request, that step is marked as failed without processing the assertions. You will see a **Server Error** as the failure reason in the test result instead of an **Assertion Error**.

### Keywords

| Keyword         | Description                   | Usage              |
| --------------- | ----------------------------- | ------------------ |
| `status_code`   | Status code                   | -                  |
| `body`          | Response body                 | -                  |
| `response_size` | Response size in bytes        | -                  |
| `response_time` | Response time in ms           | -                  |
| `headers`       | Response headers              | headers.header-key |
| `variables`     | Global and captured variables | variables.VarName  |

### Functions

| Function         | Parameters                                      | Description                                                                     |
| ---------------- | ----------------------------------------------- | ------------------------------------------------------------------------------- |
| `less_than`      | ( param `int`, limit `int` )                    | checks if param is less than limit                                              |
| `greater_than`   | ( param `int`, limit `int` )                    | checks if param is greater than limit                                           |
| `exists`         | ( param `any` )                                 | checks if variable exists                                                       |
| `equals`         | ( param1 `any`, param2 `any` )                  | checks if given parameters are equal                                            |
| `equals_on_file` | ( param `any`, file_path `string` )             | reads from given file path and checks if it equals to given parameter           |
| `in`             | ( param `any`, array_param `array` )            | checks if expression is in given array                                          |
| `contains`       | ( param1 `any`, param2 `any` )                  | makes substring with param1 inside param2                                       |
| `not`            | ( param `bool` )                                | returns converse of given param                                                 |
| `range`          | ( param `int`, low `int`,high `int` )           | returns param is in range of [low,high): low is included, high is not included. |
| `json_path`      | ( json_path `string`)                           | extracts from response body using given json path                               |
| `xpath`          | ( xpath `string` )                              | extracts from response body using given xml path                                |
| `html_path`      | ( html `string` )                               | extracts from response body using given html path                               |
| `regexp`         | ( param `any`, regexp `string`, matchNo `int` ) | extracts from given value in the first parameter using given regular expression |

### Operators

| Operator | Description  |
| -------- | ------------ |
| `==`     | equals       |
| `!=`     | not equals   |
| `>`      | greater than |
| `<`      | less than    |
| `!`      | not          |
| `&&`     | and          |
| `\|\|`   | or           |

### Assertion Examples

| Expression                                         | Description                                                                     |
| -------------------------------------------------- | ------------------------------------------------------------------------------- |
| `less_than(status_code,201)`                       | checks if status code is less than 201                                          |
| `equals(status_code,200)`                          | checks if status code equals to 200                                             |
| `status_code == 200`                               | same as preceding one                                                           |
| `not(status_code == 500)`                          | checks if status code not equals to 500                                         |
| `status_code != 500`                               | same as preceding one                                                           |
| `equals(json_path(\"employees.0.name\"),\"Name\")` | checks if json extracted value is equal to "Name"                               |
| `equals(xpath(\"//item/title\"),\"ABC\")`          | checks if xml extracted value is equal to "ABC"                                 |
| `equals(html_path(\"//body/h1\"),\"ABC\")`         | checks if html extracted value is equal to "ABC"                                |
| `equals(variables.x,100)`                          | checks if `x` variable coming from global or captured variables is equal to 100 |
| `equals(variables.x,variables.y)`                  | checks if variables `x` and `y` are equal to each other                         |
| `equals_on_file(body,\"file.json\")`               | reads from file.json and compares response body with read file                  |
| `exists(headers.Content-Type)`                     | checks if content-type header exists in response headers                        |
| `contains(body,\"xyz\")`                           | checks if body contains "xyz" in it                                             |
| `range(headers.content-length,100,300)`            | checks if content-length header is in range [100,300)                           |
| `in(status_code,[200,201])`                        | checks if status code equal to 200 or 201                                       |
| `(status_code == 200) \|\| (status_code == 201)`   | same as preceding one                                                           |
| `regexp(body,\"[a-z]+_[0-9]+\",0) == \"messi_10\"` | checks if matched result from regex is equal to "messi_10"                      |

## Success Criteria (Pass / Fail)

Ddosify supports success criteria, allowing users to verify the success of their load tests based on response times and failure counts of iterations. With this feature, users can assert the percentile of response times and the failure counts of all iterations in a test.

Users can specify the required percentile of response times and failure counts in the configuration file, and the engine will compare the actual response times and failure counts to these values throughout the test continuously. According to the user's configuration, the test can be aborted or continue running until the end. Check the [example config](https://github.com/getanteon/anteon/blob/master/ddosify_engine/config_examples/config.json) to see how the `success_criterias` keyword looks.

Note that the functions and operators mentioned in the [Step Assertion](#assertion) section can also be utilized for the Success Criteria keywords listed below.

The [examples](#examples) below show both.

## Difference Between Success Criteria and Step Assertions

Unlike assertions focused on individual steps, which determine the success or failure of a step according to its response, Success Criteria create an abort/continue logic for the entire test, which is based on the accumulated data from all iterations.

### Keywords

| Keyword              | Description                           | Usage                                                             |
| -------------------- | ------------------------------------- | ----------------------------------------------------------------- |
| `fail_count`         | Failure count of iterations           | Used for aborting when test exceeds certain fail_count            |
| `iteration_duration` | Response times of iterations in ms    | Used for percentile functions                                     |
| `fail_count_perc`    | Fail count percentage, in range [0,1] | Used for aborting when test exceeds certain fail count percentage |

### Functions

| Function | Parameters          | Description                                       |
| -------- | ------------------- | ------------------------------------------------- |
| `p99`    | ( arr `int array` ) | 99th percentile, use as `p99(iteration_duration)` |
| `p98`    | ( arr `int array` ) | 98th percentile, use as `p98(iteration_duration)` |
| `p95`    | ( arr `int array`)  | 95th percentile, use as `p95(iteration_duration)` |
| `p90`    | ( arr `int array`)  | 90th percentile, use as `p90(iteration_duration)` |
| `p80`    | ( arr `int array`)  | 80th percentile, use as `p80(iteration_duration)` |
| `min`    | ( arr `int array`)  | returns minimum element                           |
| `max`    | ( arr `int array`)  | returns maximum element                           |
| `avg`    | ( arr `int array`)  | calculates and returns average                    |

### Examples

| Expression                        | Description                                  |
| --------------------------------- | -------------------------------------------- |
| `p95(iteration_duration) < 100`   | 95th percentile should be less than 100 ms   |
| `less_than(fail_count,120)`       | Total fail count should be less than 120     |
| `less_than(fail_count_perc,0.05)` | Fail count percentage should be less than 5% |

## Correlation

Ddosify enables you to capture variables from steps using **json_path**, **xpath**, **xpath_html**, or **regular expressions**. Later, in the subsequent steps, you can inject both the captured variables and the scenario-scoped global variables.

> **:warning: Points to keep in mind**
>
> - You must specify **'header_key'** when capturing from header.
> - For json_path syntax, please take a look at [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) doc.
> - Regular expression are expected in **'Golang'** style regex. For converting your existing regular expressions, you can use [regex101](https://regex101.com).
> - You can extract values from **headers**, **body**, and **cookies**.

You can use **debug** parameter to validate your config.

```bash
ddosify run --config ddosify_config_correlation.json -debug
```

### Capture with json_path

```json
{
  "steps": [
    {
      "capture_env": {
        "NUM": { "from": "body", "json_path": "num" },
        "NAME": { "from": "body", "json_path": "name" },
        "SQUAD": { "from": "body", "json_path": "squad" },
        "PLAYERS": { "from": "body", "json_path": "squad.players" },
        "MESSI": { "from": "body", "json_path": "squad.players.0" }
      }
    }
  ]
}
```

### Capture with XPath on XML

```json
{
  "steps": [
    {
      "capture_env": {
        "TITLE": { "from": "body", "xpath": "//item/title" }
      }
    }
  ]
}
```

### Capture with XPath on HTML

```json
{
  "steps": [
    {
      "capture_env": {
        "TITLE": { "from": "body", "xpath_html": "//body/h1" }
      }
    }
  ]
}
```

### Capture with Regular Expressions

```json
{
  "steps": [
    {
      "capture_env": {
        "CONTENT_TYPE": {
          "from": "header",
          "header_key": "Content-Type",
          "regexp": { "exp": "application/(\\w)+", "matchNo": 0 }
        },
        "REGEX_MATCH_ENV": {
          "from": "body",
          "regexp": { "exp": "[a-z]+_[0-9]+", "matchNo": 1 }
        }
      }
    }
  ]
}
```

### Capture Header Value

```json
{
  "steps": [
    {
      "capture_env": {
        "TOKEN": { "from": "header", "header_key": "Authorization" }
      }
    }
  ]
}
```

### Scenario-Scoped Variables

```json
{
  "env": {
    "TARGET_URL": "http://localhost:8084/hello",
    "USER_KEY": "ABC",
    "COMPANY_NAME": "Ddosify",
    "RANDOM_COUNTRY": "{{_randomCountry}}",
    "NUMBERS": [22, 33, 10, 52]
  }
}
```

### Overall Config and Injection

On array-like captured variables or environment vars, the **rand( )** function can be utilized.

```json
// ddosify_config_correlation.json
{
  "iteration_count": 100,
  "load_type": "linear",
  "duration": 10,
  "steps": [
    {
      "id": 1,
      "url": "{{TARGET_URL}}",
      "method": "POST",
      "headers": {
        "User-Key": "{{USER_KEY}}",
        "Rand-Selected-Num": "{{rand(NUMBERS)}}"
      },
      "payload": "{{COMPANY_NAME}}",
      "capture_env": {
        "NUM": { "from": "body", "json_path": "num" },
        "NAME": { "from": "body", "json_path": "name" },
        "SQUAD": { "from": "body", "json_path": "squad" },
        "PLAYERS": { "from": "body", "json_path": "squad.players" },
        "MESSI": { "from": "body", "json_path": "squad.players.0" },
        "TOKEN": { "from": "header", "header_key": "Authorization" },
        "CONTENT_TYPE": {
          "from": "header",
          "header_key": "Content-Type",
          "regexp": { "exp": "application/(\\w)+", "matchNo": 0 }
        }
      }
    },
    {
      "id": 2,
      "url": "{{TARGET_URL}}",
      "method": "POST",
      "headers": {
        "User-Key": "{{USER_KEY}}",
        "Authorization": "{{TOKEN}}",
        "Content-Type": "{{CONTENT_TYPE}}"
      },
      "payload_file": "payload.json",
      "capture_env": {
        "TITLE": { "from": "body", "xpath": "//item/title" },
        "REGEX_MATCH_ENV": {
          "from": "body",
          "regexp": { "exp": "[a-z]+_[0-9]+", "matchNo": 1 }
        }
      }
    }
  ],
  "env": {
    "TARGET_URL": "http://localhost:8084/hello",
    "USER_KEY": "ABC",
    "COMPANY_NAME": "Ddosify",
    "RANDOM_COUNTRY": "{{_randomCountry}}",
    "NUMBERS": [22, 33, 10, 52]
  }
}
```

```json
// payload.json
{
  "boolField": "{{_randomBoolean}}",
  "numField": "{{NUM}}",
  "strField": "{{NAME}}",
  "numArrayField": ["{{NUM}}", 34],
  "strArrayField": ["{{NAME}}", "hello"],
  "mixedArrayField": ["{{NUM}}", 34, "{{NAME}}", "{{SQUAD}}"],
  "{{NAME}}": "messi",
  "obj": {
    "numField": "{{NUM}}",
    "objectField": "{{SQUAD}}",
    "arrayField": "{{PLAYERS}}"
  }
}
```

## Test Data Set

Ddosify enables you to load test data from **CSV** files. Later, in your scenario, you can inject variables that you tagged.

We are using this [CSV data](https://github.com/getanteon/anteon/tree/master/ddosify_engine/config/config_testdata/test.csv) in config below.

```json
// config_data_csv.json
"data":{
      "csv_test": {
          "path" : "config/config_testdata/test.csv",
          "delimiter": ";",
          "vars": {
                  "0":{"tag":"name"},
                  "1":{"tag":"city"},
                  "2":{"tag":"team"},
                  "3":{"tag":"payload", "type":"json"},
                  "4":{"tag":"age", "type":"int"}
                },
          "allow_quota" : true,
          "order": "random",
          "skip_first_line" : true
      }
    }
```

You can refer to tagged variables in your request like below.

```json
// payload.json
{
  "name": "{{data.csv_test.name}}",
  "team": "{{data.csv_test.team}}",
  "city": "{{data.csv_test.city}}",
  "payload": "{{data.csv_test.payload}}",
  "age": "{{data.csv_test.age}}"
}
```

## Cookies

Ddosify supports cookies in the following engine modes: `distinct-user` and `repeated-user`. Cookies are not supported in the default `ddosify` mode.

In `repeated-user` mode, Ddosify uses the same cookie jar for all iterations executed by the same user. It sets cookies returned at the first successful iteration and does not change them afterward. This way, the same cookies are passed through steps in all iterations executed by the same user.

In `distinct-user` mode, Ddosify uses a different cookie jar for each iteration, so cookies are passed through steps in one iteration only.

The sections below show each case.

### Initial / Custom Cookies

You can set initial/custom cookies for your test scenario using `cookie_jar` field in the config file. You can enable/disable custom cookies with `enabled` key. Check the [example config](https://github.com/getanteon/anteon/blob/master/ddosify_engine/config/config_testdata/config_init_cookies.json).

| Key         | Description                                                                                                     | Example                                                           |
| ----------- | --------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `name`      | The name of the cookie. This field is used to identify the cookie.                                              | `platform`                                                        |
| `value`     | The value of the cookie. This field contains the data that the cookie stores.                                   | `web`                                                             |
| `domain`    | Domain or subdomain that can access the cookie.                                                                 | `app.getanteon.com`                                               |
| `path`      | Path within the domain that can access the cookie.                                                              | `/`                                                               |
| `expires`   | When the cookie should expire. The date format should be rfc2616.                                               | `Thu, 16 Mar 2023 09:24:02 GMT`                                   |
| `max_age`   | Number of seconds until the cookie expires.                                                                     | `5`                                                               |
| `http_only` | Whether the cookie should only be accessible through HTTP or HTTPS headers, and not through client-side scripts | `true`                                                            |
| `secure`    | Whether the cookie should only be sent over a secure (HTTPS) connection                                         | `false`                                                           |
| `raw`       | The raw format of the cookie. If it is used, the other keys are discarded.                                      | `myCookie=myValue; Expires=Wed, 21 Oct 2026 07:28:00 GMT; Path=/` |

### Cookie Capture

You can capture values from cookies from its name just like you do for headers and body and use them in your test scenario.

```json
{
    "iteration_count": 100,
    "load_type": "linear",
    "duration": 10,
    "steps": [
        {
          ...
          "capture_env": {
            "TEST" :{"from":"cookies","cookie_name":"test"}
          }
        }
    ]
}
```

### Cookie Assertion

You can refer to cookie values as `cookies.cookie_name` while you write assertions for your steps.

Following fields are available for cookie assertion:

- `name`: Name of the cookie
- `domain`: Domain of the cookie
- `path`: Path of the cookie
- `value`: Value of the cookie
- `expires`: Expiration date of the cookie
- `maxAge`: Max age of the cookie
- `secure`: Secure flag of the cookie
- `httpOnly`: Http only flag of the cookie
- `rawExpires`: Raw expiration date of the cookie

**Examples:**

- `cookies.test.expires < time(\"Thu, 01 Jan 1990 00:00:00 GMT\")` is a valid assertion expression. It checks if the cookie named `test` has an expiration date before `Thu, 01 Jan 1990 00:00:00 GMT`.
- `cookies.test.path == \"/login\"` is another valid assertion expression. It checks if the cookie named `test` has a path value equal to `/login`.

## Upgrading from v1

Ddosify v2.0.0 is a major version:

- The module path is `go.ddosify.com/ddosify/v2`.
- The CLI has commands (`ddosify run`); headers are `-H`, `-h` is help, long flags take two dashes (`--config`, `--cert-path`), and flags override the config file instead of being ignored. `ddosify -t URL` and `ddosify -config FILE` keep working.
- TLS certificates are verified. `--insecure`, or `"insecure": true` in the config or on a step, turns that off for a host with a self-signed certificate.
- A failed success criterion exits `3` instead of `1`.
- The Docker image has no shell: `docker run --rm ddosify/ddosify run ...`.

## Verifying a release

The release's `checksums.txt` lists every archive and its SBOM, and carries a keyless cosign signature and a GitHub artifact attestation; so do the images.

```bash
cosign verify-blob --bundle checksums.txt.sigstore.json \
  --certificate-identity 'https://github.com/getanteon/anteon/.github/workflows/release.yml@refs/tags/v2.0.0' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
sha256sum --check --ignore-missing checksums.txt
gh attestation verify --owner getanteon ddosify_2.0.0_linux_amd64.tar.gz
gh attestation verify --owner getanteon oci://docker.io/ddosify/ddosify:v2.0.0
```

## Streaming (SSE) and LLM Endpoints

An LLM API answers with a stream: the headers come at once, the first word comes seconds later, and the answer ends tens of seconds after that. A plain response time says none of this. A step with `stream` reads the body as [server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html) as it arrives and records, per request, when each event came.

### Stream config

```json
"stream": {
    "type": "sse",
    "first_token": {"json_path": "type", "equals": "text-delta"},
    "error": {"json_path": "type", "equals": "error"},
    "done": "[DONE]",
    "max_body_bytes": 1048576
}
```

| Field            | Description                                                                                                                                                                                                                          | Default                           |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------- |
| `type`           | `sse`, the only stream type.                                                                                                                                                                                                        | `sse`                             |
| `first_token`    | The event that counts as the first token. Fields: `event` (the SSE `event:` name), `json_path` (a [gjson](https://github.com/tidwall/gjson) path into the event's data), `equals` (the value at `json_path`), `regexp` (matched against the value at `json_path`, or the whole data). Every field given has to match. `json_path` alone needs a value that is not empty. `{}` makes the first event the first token. | `type` equals `text-delta`, the [AI SDK UI message stream](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol)'s first word |
| `error`          | An event that reports a failure inside a `200` stream. Same fields as `first_token`. A stream that carries one is not counted as goodput.                                                                                          | `type` equals `error`             |
| `done`           | The data that ends the stream. Reading stops there. `""` reads until the server closes.                                                                                                                                         | `[DONE]`                          |
| `max_body_bytes` | The body is kept only when an assertion, a capture or debug mode needs it, and then up to this many bytes. `0` keeps all of it.                                                                                                | `1048576`                         |

For an OpenAI compatible chat completions endpoint use `"first_token": {"json_path": "choices.0.delta.content"}`.

When a step has no `stream` but the response is `text/event-stream`, it is read as a stream with the defaults above and no body cap, so assertions and captures see the whole body as before. A step with `stream` that gets an error status, such as a `429` problem document, reads it whole and records no stream. A stream is read until `done`, the end of the body or the step's `timeout`, which covers the whole stream: set it above the longest answer.

### What is measured

Every time is from the moment the request was sent.

| Metric        | Meaning                                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------------------- |
| `ttfb`        | First byte of the response, the status line                                                             |
| `first_event` | First event dispatched. Comments such as `: keepalive` are not events                                   |
| `first_token` | First event matching `first_token`                                                                      |
| `total`       | End of the stream                                                                                       |
| `gap_max`, `gap_p50`, `gap_p95` | The largest, median and 95th percentile time between consecutive events of one stream |
| `events`      | Events in the stream, `done` excluded                                                                   |
| ends          | `done` (the done marker came), `closed` (the server closed between events), `cut` (the connection broke, or closed in the middle of an event), `timeout` |

The parser follows the WHATWG event stream rules: `LF`, `CR` and `CRLF` line ends, a line or a `CRLF` split across reads, multi-line `data:` joined with a newline, comments, a leading byte order mark. Memory per stream is bounded: one event's data is cut at 1 MiB, and at most 4,096 gaps are kept for its percentiles.

The report gives each metric's p50, p90, p95, p99, max and mean over the step's successful streams, those that ended cleanly with no error event, as vLLM aggregates successful requests only; cut, timed out and failed streams are in the ends and outcomes and counted as `excluded_from_latency`. It counts every step result as one outcome: `ok`, `rate_limited_429`, `unavailable_503`, `server_error_5xx`, `client_error_4xx`, `timeout`, `network_error`, `stream_cut`, `stream_error_event` or `interrupted`. With `-o stdout-json` they are under `steps.<id>.stream` and `steps.<id>.outcomes`. Percentiles are exact up to 100,000 samples per metric and from a uniform sample beyond. These sections appear only once a test has a stream, an `slo` or a named stage, so the output of other tests does not change. For a streamed step the duration table runs to the end of the stream: Response Read is the time from the headers to the last event, Total is the stream's total, and `response_time` in an assertion is the same total. When a server answers before the request body is fully written, as an early 503 or 413 to a large upload, Server Processing is 0 and Request Write runs to the first byte of the answer.

Run one iteration with `-debug` to see one stream's measurements next to the request and response.

### Goodput and SLO

```json
"slo": {"first_token_ms": 10000, "total_ms": 30000}
```

An iteration is good when every step's outcome is `ok`, no assertion failed, and every stream ended cleanly, carried no error event and met every threshold given. A threshold left out is not checked; a stream that never sent its first token misses `first_token_ms`. The report prints `Goodput: good / iterations (%)`, warm-up excluded, and `goodput` in the JSON output.

| Key              | Threshold on      |
| ---------------- | ----------------- |
| `ttfb_ms`        | `ttfb`            |
| `first_event_ms` | `first_event`     |
| `first_token_ms` | `first_token`     |
| `total_ms`       | `total`           |
| `max_gap_ms`     | `gap_max`         |
| `goodput_perc`   | not a threshold on a stream: the test fails, exit code `3`, when the goodput is below this percentage |

### Stepped arrival rate and stages

Ddosify starts iterations on a schedule, whatever the earlier ones are doing, so `manual_load` is an arrival rate. With `load_spread: even`, a stage of `count` iterations over `duration` seconds starts one every `duration / count` seconds. Two minutes of warm-up, then 6, 12, 24, 48 and 96 iterations a minute for five minutes each:

```json
"load_spread": "even",
"manual_load": [
    {"duration": 120, "count": 6,   "name": "warm-up", "warmup": true},
    {"duration": 300, "count": 30,  "name": "6/min"},
    {"duration": 300, "count": 60,  "name": "12/min"},
    {"duration": 300, "count": 120, "name": "24/min"},
    {"duration": 300, "count": 240, "name": "48/min"},
    {"duration": 300, "count": 480, "name": "96/min"}
]
```

An iteration belongs to the stage it started in. The report has one row per stage: iterations, iterations that succeeded, goodput, `429`, `503`, other `5xx`, timeouts, network errors and cut streams, and first token and total p50 and p95. The stream summary, the outcomes and the goodput of the whole test leave the warm-up out. With `-o stdout-json` each stage is in `stages`, with its full stream summary.

### Example: an AI SDK chat API

[config_examples/sse_llm.json](config_examples/sse_llm.json) creates a chat, captures its id and asks a question from [a CSV](config_examples/questions.csv) in the same iteration. Each iteration is one person: `"END_USER": "loadtest:{{_randomUUID}}"` in `env` is generated once per iteration and sent by both steps, so a per end user rate limit sees a new person every time, and `{{_randomUUID}}` in the `Idempotency-Key` header is new on every request. A question is put into the JSON body as `"{{data.questions.text}}"`, quoted, so Ddosify escapes it as a JSON string; UTF-8 text such as Turkish passes through unchanged.

```bash
export GURUBASE_URL=https://gurubase.example.internal GURU_ID=... GURUBASE_API_KEY=gbk_...
ddosify run --config config_examples/sse_llm.json -debug   # one iteration, verbose
ddosify run --config config_examples/sse_llm.json          # the whole run
```

Count every request against the API's rate limits: two requests an iteration at 96 iterations a minute is 192 requests a minute on one key.

The ask step counts the answer's tokens with `"llm": {"token_source": "tokenizer"}` and the served model's Hugging Face `tokenizer.json`, given in the config or with `--tokenizer` (`DDOSIFY_TOKENIZER`). The tokenizer is forced rather than left to `auto` because Gurubase's `finish` chunk carries the usage of the whole run, every model call that planned, wrote and judged the answer, not the tokens the stream delivered; TPOT and tokens per second from it would be wrong by a large factor. The server's numbers stay in each result as `usage_output` and `usage_input`. Without a tokenizer the output is counted in chunks, and the report says chunks.

```bash
ddosify run --config config_examples/sse_llm.json --tokenizer ~/models/Qwen3-8B/tokenizer.json --html report.html
```

### Tokens, the OpenAI preset, sessions and prefix caching

[docs/llm.md](docs/llm.md) covers output tokens (from the server's usage, a Hugging Face `tokenizer.json`, or labelled as chunks), TPOT, ITL and tokens per second, the OpenAI compatible preset that needs only a URL, a key and a dataset, multi-turn sessions, cache-busting and shared prefixes, and the scheduling lag that shows a saturated client.

## HTML report

One self-contained HTML file that opens offline and loads nothing from the network. It holds numbers and step names only: no URL, header, body, token, credential or test data, so it can be sent on. Light and dark follow the reader's system, and a button switches them.

Write it while the test runs, or afterwards from the `--json` output:

```bash
ddosify run --config config.json --html report.html

ddosify run --config config.json --json > run.ndjson
ddosify report run.ndjson -o report.html --title "Nightly run"

kubectl logs job/loadtest | ddosify report -o report.html  # a pod's log holds both streams

# in a container, with no volume: the log in on stdin, the page out on stdout
docker run -i --rm IMAGE report -o - < run.ndjson > report.html

# runs side by side: one chart, one capacity table
ddosify report a.ndjson b.ndjson --label "model A" --label "model B" --gpus 1 -o compare.html
```

What it shows:

- **Cards**: goodput, requests, 429 and 503, and the first token and total p95 of the streamed steps.
- **Arrival rate**: iterations started a second (a minute under one a second) against each stage's planned rate, and failed requests. **In flight**: requests sent and not yet ended.
- **Latency**: p50 as a line and p50 to p95 as a band, with the SLO, for the first token, the total time and the ITL; goodput over time. Hovering one chart shows the same moment on every chart.
- **Stages**: each stage's outcomes as one bar (within SLO, outside it, 503, 429, other), and its first token and total p50, p95 and p99.
- **Throughput and interactivity**, when the stages are closed-loop or there are two or more with counted output: each stage as a point, its output tokens a second per GPU against the median answer's tokens a second after its first token, with the TPOT limits of the SLO tiers; and the **capacity** each tier allows, the highest concurrency, or the highest arrival rate of open-loop stages with the requests in flight by Little's law, whose TTFT p99 and TPOT p99 are within it.
- Tables of the outcomes, the answers' tokens, the stages, every step's percentiles and the shape of the load.

With `request_log` on, `ddosify report` also reads the `request` lines and adds:

- **Latency by start time**: every quarter window, the answers that started over the last window, or longer when 30 requests took longer to start. A request is placed where it began, so a slow answer is charged to the load that caused it, not to the moment it ended.
- **Steady state**: each stage without its first moments, the iterations that began within the window or the previous stage's p95 total time, whichever is longer. Its p95 has a distribution-free 95% confidence interval. With more than one measured step, each step's percentiles per stage.

The window is half the shortest stage past warm-up, from 10 to 60 seconds. When a step streams, the streamed steps are measured; otherwise every step. Memory stays bounded on a long run: the time series keeps about 1,500 points, and above 25,000 requests a step is measured from a seeded uniform sample, which the report says.

`ddosify report` exits `2` when the log can not be read, has no `report` line, or the output can not be written.

## Common Issues

### macOS Security Issue

```
"ddosify" can’t be opened because Apple cannot check it for malicious software.
```

- Open `/usr/local/bin`
- Right click `ddosify` and select Open
- Select Open
- Close the opened terminal

### OS Limit - Too Many Open Files

If you create large load tests, you may encounter the following errors:

```
Server Error Distribution (Count:Reason):
  199      :Get "https://getanteon.com": dial tcp 188.114.96.3:443: socket: too many open files
  159      :Get "https://getanteon.com": dial tcp 188.114.97.3:443: socket: too many open files
```

This is because the OS limits the number of open files. You can check the current limit by running `ulimit -n` command. You can increase this limit to 50000 by running the following command on both Linux and macOS.

```bash
ulimit -n 50000
```

But this will only increase the limit for the current session. To increase the limit permanently, you can change the shell configuration file. For example, if you are using bash, you can add the following lines to `~/.bashrc` file. If you are using zsh, you can add the following lines to `~/.zshrc` file.

```bash
# For .bashrc
echo "ulimit -n 50000" >> ~/.bashrc

# For .zshrc
echo "ulimit -n 50000" >> ~/.zshrc
```

## Contributing

See our [Contribution Guide](../CONTRIBUTING.md) and please follow the [Code of Conduct](../CODE_OF_CONDUCT.md) in all your interactions with the project.

## Communication

You can join our [Discord Server](https://discord.com/invite/9KdnrSUZQg) for issues, feature requests, feedbacks or anything else.

## Disclaimer

Ddosify is created for testing the performance of web applications. Users must be the owner of the target system. Using it for harmful purposes is extremely forbidden. Ddosify team & company is not responsible for its’ usages and consequences.

## License

Licensed under the [AGPLv3](../LICENSE)
