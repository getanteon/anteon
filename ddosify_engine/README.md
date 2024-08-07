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

## Features

- ✅ **[Scenario-Based](#config-file)** - Create your flow in a JSON file. Without a line of code!

- ✅ **[Different Load Types](#load-types)** - Test your system's limits across different load types.

- ✅ **[Parameterization](#parameterization-dynamic-variables)** - Use dynamic variables just like on Postman.

- ✅ **[Correlation](#correlation)** - Extract variables from earlier phases and pass them on to the following ones.

- ✅ **[Test Data](#test-data-set)** - Import test data from CSV and use it in the scenario.

- ✅ **[Assertion](#assertion)** - Verify that the response matches your expectations.

- ✅ **[Success Criteria](#success-criteria-pass--fail)** - Set the success criteria for your test.

- ✅ **[Cookies](#cookies)** - Pass cookies through steps and set initial cookies if you want.

- ✅ **Widely Used Protocols** - Currently supporting _HTTP, HTTPS, HTTP/2_. Other protocols are on the way.

## Tutorials / Blog Posts

- [Testing the Performance of User Authentication Flow](https://getanteon.com/blog/testing-the-performance-of-user-authentication-flow)
- [Load Testing a Fintech API with CSV Test Data Import](https://getanteon.com/blog/load-testing-a-fintech-exchange-api-with-csv-test-data-import)

## Installation

`ddosify` is available via [Docker](https://hub.docker.com/r/ddosify/ddosify), [Docker Extension](https://hub.docker.com/extensions/ddosify/ddosify-docker-extension), [Homebrew Tap](#homebrew-tap-macos-and-linux), and downloadable as pre-compiled binaries from the [releases page](https://github.com/getanteon/anteon/releases/tag/v1.0.6) for macOS, Linux and Windows.

For shell auto completions, see [Ddosify Completions](https://github.com/getanteon/anteon/tree/master/ddosify_engine/completions).

### Docker

```bash
docker run -it --rm ddosify/ddosify
```

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

- Download zip file for your architecture from the [releases page](https://github.com/ddosify/ddosify/releases/tag/v1.0.6).
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

_Minimum supported Go version is 1.18_

```bash
go install -v go.ddosify.com/ddosify@latest
```

## Quick Start

This section aims to show you how to use Ddosify easily without deep dive into its details.

1.  ### Simple load test

    `ddosify -t https://getanteon.com`

    The above command runs a load test with the default value that is 100 requests in 10 seconds.

2.  ### Using some of the features

    `ddosify -t https://getanteon.com -n 1000 -d 20 -m PUT -T 7 -P http://proxy_server.com:80`

    Ddosify sends a total of _1000_ _PUT_ requests to *https://getanteon.com* over proxy _http://proxy_server.com:80_ in _20_ seconds with a timeout of _7_ seconds per request.

3.  ### Usage for CI/CD pipelines (JSON output)

    `ddosify -t https://getanteon.com -o stdout-json | jq .avg_duration`

    Ddosify outputs the result in JSON format. Then `jq` (or any other command-line JSON processor) fetches the `avg_duration`. The rest depends on your CI/CD flow logic.

4.  ### Scenario based load test

    `ddosify -config config_examples/config.json`

    Ddosify first sends _HTTP/2 POST_ request to *https://getanteon.com/endpoint_1* using basic auth credentials _test_user:12345_ over proxy _http://proxy_host.com:proxy_port_ and with a timeout of _3_ seconds. Once the response is received, HTTPS GET request will be sent to *https://getanteon.com/endpoint_2* along with the payload included in _config_examples/payload.txt_ file with a timeout of 2 seconds. This flow will be repeated _20_ times in _5_ seconds and response will be written to _stdout_.

5.  ### Load test with Dynamic Variables (Parameterization)

    `ddosify -t https://getanteon.com/{{_randomInt}} -d 10 -n 100 -h 'User-Agent: {{_randomUserAgent}}' -b '{"city": "{{_randomCity}}"}'`

    Ddosify sends a total of _100_ _GET_ requests to *https://getanteon.com/{{_randomInt}}* in _10_ seconds. `{{_randomInt}}` path generates random integers between 1 and 1000 in every request. Dynamic variables can be used in _URL_, _headers_, _payload (body)_ and _basic authentication_. In this example, Ddosify generates a random user agent in the header and a random city in the body. The full list of the dynamic variables can be found in the [docs](https://getanteon.com/docs/performance-testing/dynamic-variables-parametrization/).

6.  ### Correlation (Captured Variables)

    `ddosify -config ddosify_config_correlation.json`

    Ddosify allows you to specify variables at the global level and use them throughout the scenario, as well as extract variables from previous steps and inject them to the next steps in each iteration individually. You can inject those variables in requests _url_, _headers_ and _payload(body)_. The example config can be found in [correlation-config-example](#Correlation).

7.  ### Test Data

    `ddosify -config ddosify_data_csv.json`

    Ddosify allows you to load test data from a file, tag specific columns for later use. You can inject those variables in requests _url_, _headers_ and _payload (body)_. The example config can be found in [test-data-example](#test-data-set).

## Advanced Usage

You can configure your load test by the CLI options or a config file. Config file supports more features than the CLI. For example, you can't create a scenario-based load test with CLI options.

### CLI Flags

```bash
ddosify [FLAG]
```

| Flag                                                        | Description                                                                                                       | Type     | Default  | Required |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | -------- | -------- | -------- |
| `-t`                                                        | Target website URL. Example: https://getanteon.com                                                                | `string` | -        | Yes      |
| `-n`                                                        | Total iteration count                                                                                             | `int`    | `100`    | No       |
| `-d`                                                        | Test duration in seconds.                                                                                         | `int`    | `10`     | No       |
| `-m`                                                        | Request method. Available methods for HTTP(s) are _GET, POST, PUT, DELETE, HEAD, PATCH, OPTIONS_                  | `string` | `GET`    | No       |
| `-b`                                                        | The payload of the network packet. AKA body for the HTTP.                                                         | `string` | -        | No       |
| `-a`                                                        | Basic authentication. Usage: `-a username:password`                                                               | `string` | -        | No       |
| `-h`                                                        | Headers of the request. You can provide multiple headers with multiple `-h` flag. Usage: `-h 'Accept: text/html'` | `string` | -        | No       |
| `-T`                                                        | Timeout of the request in seconds.                                                                                | `int`    | `5`      | No       |
| `-P`                                                        | Proxy address as host:port. `-P 'http://user:pass@proxy_host.com:port'`                                           | `string` | -        | No       |
| `-o`                                                        | Test result output destination. Supported outputs are [*stdout, stdout-json*] Other output types will be added.   | `string` | `stdout` | No       |
| `-l`                                                        | [Type](#load-types) of the load test. Ddosify supports 3 load types.                                              | `string` | `linear` | No       |
| <span style="white-space: nowrap;">`--config`</span>        | [Config File](#config-file) of the load test.                                                                     | `string` | -        | No       |
| <span style="white-space: nowrap;">`--version`</span>       | Prints version, git commit, built date (utc), go information and quit                                             | -        | -        | No       |
| <span style="white-space: nowrap;">`--cert_path`</span>     | A path to a certificate file (usually called 'cert.pem')                                                          | -        | -        | No       |
| <span style="white-space: nowrap;">`--cert_key_path`</span> | A path to a certificate key file (usually called 'key.pem')                                                       | -        | -        | No       |
| <span style="white-space: nowrap;">`--debug`</span>         | Iterates the scenario once and prints curl-like verbose result. Note that this flag overrides json config.        | `bool`   | `false`  | No       |

### Load Types

#### Linear

```bash
ddosify -t https://getanteon.com -l linear
```

Result:

![linear load](https://raw.githubusercontent.com/getanteon/anteon/master/assets/linear.gif)

_Note:_ If the iteration count is too low for the given duration, the test might be finished earlier than you expect.

#### Incremental

```bash
ddosify -t https://getanteon.com -l incremental
```

Result:

![incremental load](https://raw.githubusercontent.com/getanteon/anteon/master/assets/incremental.gif)

#### Waved

```bash
ddosify -t https://getanteon.com -l waved
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
- Payload from a file
- Multipart/form-data payload
- Extra connection configuration
- HTTP2 support

Usage:

```bash
ddosify -config <json_config_path>
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
  Also `manual_load` overrides `load_type` if you provide both of them. As a result, you don't need to provide these 3 parameters when using `manual_load`.

  ```json
  "manual_load": [
      {"duration": 5, "count": 5},
      {"duration": 6, "count": 10},
      {"duration": 7, "count": 20}
  ]
  ```

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
ddosify -t https://getanteon.com/?key={{_randomString}} -d 10 -n 100
```

### Parameterization on Headers

Ddosify sends _100_ GET requests in _10_ seconds with random `Transaction-Type` and `Country` headers.

```bash
ddosify -t https://getanteon.com -d 10 -n 100 -h 'Transaction-Type: {{_randomTransactionType}}' -h 'Country: {{_randomCountry}}'
```

### Parameterization on Payload (Body)

Ddosify sends _100_ GET requests in _10_ seconds with random `latitude` and `longitude` values in body.

```bash
ddosify -t https://getanteon.com -d 10 -n 100 -b '{"latitude": "{{_randomLatitude}}", "longitude": "{{_randomLongitude}}"}'
```

### Parameterization on Basic Authentication

Ddosify sends _100_ GET requests in _10_ seconds with random `username` and `password` with basic authentication.

```bash
ddosify -t https://getanteon.com -d 10 -n 100 -a '{{_randomUserName}}:{{_randomPassword}}'
```

### Parameterization on Config File

Dynamic variables can be used on config file as well. Ddosify sends _100_ GET requests in _10_ seconds with random string `key` parameter in URL and random `User-Key` header.

```bash
ddosify -config ddosify_config_dynamic.json
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
ddosify -config ddosify_config_os_env.json
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

You can see a success criteria example in the [EXAMPLES](https://github.com/getanteon/anteon/blob/master/ddosify_engine/EXAMPLES.md#example-2-success-criteria) file.

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
ddosify -config ddosify_config_correlation.json -debug
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

You can see a cookie example in the [EXAMPLES](https://github.com/getanteon/anteon/blob/master/ddosify_engine/EXAMPLES.md#example-1-cookie-support) file.

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

## More

This repository includes the single-node version of the Ddosify Loader. For distributed and Geo-targeted Load Testing you can use [Anteon Cloud](https://getanteon.com)

## Disclaimer

Ddosify is created for testing the performance of web applications. Users must be the owner of the target system. Using it for harmful purposes is extremely forbidden. Ddosify team & company is not responsible for its’ usages and consequences.

## License

Licensed under the [AGPLv3](../LICENSE)
