<h1 align="center">
    <img src="assets/ddosify-logo.svg"
        width="336px" /><br />
    Ddosify - High-performance load testing tool
</h1>

<p align="center">
    <a href="https://github.com/create-go-app/cli/releases" target="_blank"><img src="https://img.shields.io/badge/version-v0.0.1-orange?style=for-the-badge&logo=none" alt="ddosify version" /></a>&nbsp;
    <a href="https://pkg.go.dev/github.com/create-go-app/cli/v3?tab=doc" target="_blank"><img src="https://img.shields.io/badge/Go-1.17+-00ADD8?style=for-the-badge&logo=go" alt="go version" /></a>&nbsp;
    <a href="https://goreportcard.com/report/github.com/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/badge/Go_report-A+-success?style=for-the-badge&logo=none" alt="go report" /></a>&nbsp;
    <img src="https://img.shields.io/github/license/ddosify/ddosify?style=for-the-badge&logo=none" alt="license" />
</p>

<img src="assets/ddosify-quick-start.gif" alt="Ddosify - High-performance load testing tool quick start"/>

## Features
:heavy_check_mark: Protocol Agnostic - Currently supporting *HTTP, HTTPS, HTTP/2*. Other protocols are on the way.

:heavy_check_mark: Scenario-Based - Create your flow in a JSON file. Without a line of code!

:heavy_check_mark: Different Load Types - Test your system's limits across different load types.


## Installation

`ddosify` is available via [Docker](https://hub.docker.com/ddosify/ddosify and downloadable pre-compiled binaries from the [releases page](https://github.com/ddosify/ddosify/releases/latest).

### Docker

```bash
docker run -it --rm --name ddosify ddosify/ddosify
```
### Using the convenience script (macOS and Linux)

- The script requires root or sudo privileges to move ddosify binary to `/usr/local/bin`.
- The script attempts to detect your operating system (macOS or Linux) and architecture (arm64, x86, x8664, i386) to download the appropriate binary from the [releases page](https://github.com/ddosify/ddosify/releases/latest).
- By default, the script installs the latest version of `ddosify`.

```bash
curl -sSfL https://raw.githubusercontent.com/ddosify/ddosify/main/scripts/install.sh | sh
```

### Go install from source (macOS, Linux, Windows)

```bash
go install -v github.com/ddosify/ddosify@latest
```

## Easy Start
This section aims to show you how to use Ddosify without deep dive into its details easily.
    
1. ### Simple load test

		ddosify -t target_site.com

    The above command runs a load test with the default value that is 200 requests in 10 seconds.

2. ### Using some of the features

		ddosify -t target_site.com -n 1000 -d 20 -p HTTPS -m PUT -T 7 -P http://proxy_server.com:80

    Ddosify sends a total of *1000* *PUT* requests to *https://target_site.com* over proxy *http://proxy_server.com:80* in *20* seconds with a timeout of *7* seconds per request.

3. ### Scenario based load test

		ddosify -config config_examples/config.json
    
    Ddosify first sends *HTTP/2 POST* request to *https://test_site1.com/endpoint_1* using basic auth credentials *test_user:12345* over proxy *http://proxy_host.com:proxy_port*  and with a timeout of *3* seconds. Once the response is received, HTTPS GET request will be sent to *https://test_site1.com/endpoint_2* along with the payload included in *config_examples/payload.txt* file with a timeout of 2 seconds. This flow will be repeated *20* times in *5* seconds and response will be written to *stdout*.

		
## Details

You can configure your load test by the CLI options or a config file. Config file supports more features than the CLI. For example, you can't create a scenario-based load test with CLI options.
### CLI Flags

```bash
ddosify [FLAG]
```

| Flag | Description                                              | Type     | Default | Required?  |
| ------ | -------------------------------------------------------- | ------   | ------- | ---------  |
| `-t`   | Target website URL. Example: https://ddosify.com         | `string` | - | Yes        |
| `-n`   | Total request count                                      | `int`    | `200`   | No         |
| `-d`   | Test duration in seconds.                                | `int`    | `10`    | No         |
| `-p`   | Protocol of the request. Supported protocols are *HTTP, HTTPS*. HTTP/2 support is only available by using a config file as described. More protocols will be added.                                | `string`    | `HTTPS`    | No         |
| `-m`   | Request method. Available methods for HTTP(s) are *GET, POST, PUT, DELETE, UPDATE, PATCH* | `string`    | `GET`    | No  |
| `-b`   | The payload of the network packet. AKA body for the HTTP.  | `string`    | -    | No         |
| `-a`   | Basic authentication. Usage: `-a username:password`        | `string`    | -    | No         |
| `-h`   | Headers of the request. You can provide multiple headers with multiple `-h` flag.  | `string`| -    | No         |
| `-T`   | Timeout of the request in seconds.                       | `int`    | `5`    | No         |
| `-P`   | Proxy address as host:port. `-P http://user:pass@proxy_host.com:port'` | `string`    | -    | No |
| `-o`   | Test result output destination. Other output types will be added. | `string`    | `stdout`    | No |
| `-l`   | [Type](#load-types) of the load test. Ddosify supports 3 load types. | `string`    | `linear`    | No |
| `-config`   | [Config File](#config-file) of the load test. | `string`    | -    | No |


### Load Types

#### Linear

```bash
ddosify -t target_site.com -l linear
```

Result:

![linear load](assets/linear.gif)

*Note:* If the request count is too low for the given duration, the test might be finished earlier than you expect.

#### Incremental

```bash
ddosify -t target_site.com -l incremental
```

Result:

![incremental load](assets/incremental.gif)


#### Waved

```bash
ddosify -t target_site.com -l waved
```

Result:

![waved load](assets/waved.gif)


### Config File

Config file lets you use all capabilities of Ddosify. 

The features you can use by config file;
- Scenario creation
- Payload from a file
- Extra connection configuration, like *keep-alive* enable/disable logic
- HTTP2 support

Usage;

    ddosify -config <json_config_path>


There is an example config file at [config_examples/config.json](/config_examples/config.json). This file contains all of the parameters you can use. Details of each parameter;

- `request_count` *optional*

    This is the equivalent of the `-n` flag. The difference is that if you have multiple steps in your scenario, this value represents the iteration count of the steps.

- `load_type` *optional*

    This is the equivalent of the `-l` flag.

- `duration` *optional*

    This is the equivalent of the `-d` flag.

- `proxy` *optional*

    This is the equivalent of the `-P` flag.

- `output` *optional*

    This is the equivalent of the `-o` flag.

- `steps` *mandatory*

    This parameter lets you create your scenario. Ddosify runs the provided steps, respectively. For the given example file step id: 2 will be executed immediately after the response of step id: 1 is received. The order of the execution is the same as the order of the steps in the config file.
    
        Details of each parameter for a step;
    - `id` *mandatory*
    
        Each step must have a unique integer id.

    - `url` *mandatory*

        This is the equivalent of the `-t` flag.
    
    - `protocol` *optional*

        This is the equivalent of the `-p` flag.

    - `method` *optional*

        This is the equivalent of the `-m` flag.

    - `headers` *optional*

        List of headers with key:value format.

    - `payload` *optional*

        This is the equivalent of the `-b` flag.

    - `payload_file` *optional*

        If you need a long payload, we suggest using this parameter instead of `payload`.  

    - `auth` *optional*
        
        Basic authentication.
        ```json
        "auth": {
            "username": "test_user",
            "password": "12345"
        }
        ```
    - `others` *optional*

        This parameter accepts dynamic *key: value* pairs to configure connection details of the protocol in use.

        ```json
        "others": {
            "keep-alive": true,              // Default false
            "disable-compression": false,    // Default true
            "h2": true,                      // Enables HTTP/2. Default false.
            "disable-redirect": true         // Default false
        }
        ```

## License

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
