<h1 align="center">
    <img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-logo.svg" alt="Ddosify logo" width="336px" /><br />
    Ddosify - High-performance load testing tool
</h1>

<p align="center">
    <a href="https://github.com/ddosify/ddosify/releases" target="_blank"><img src="https://img.shields.io/github/v/release/ddosify/ddosify?style=for-the-badge&logo=github&color=orange" alt="ddosify latest version" /></a>&nbsp;
    <a href="https://github.com/ddosify/ddosify/actions/workflows/test.yml" target="_blank"><img src="https://img.shields.io/github/workflow/status/ddosify/ddosify/Test?style=for-the-badge&logo=github" alt="ddosify build result" /></a>&nbsp;
    <a href="https://pkg.go.dev/go.ddosify.com/ddosify" target="_blank"><img src="https://img.shields.io/github/go-mod/go-version/ddosify/ddosify?style=for-the-badge&logo=go" alt="golang version" /></a>&nbsp;
    <a href="https://app.codecov.io/gh/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/codecov/c/github/ddosify/ddosify?style=for-the-badge&logo=codecov" alt="go coverage" /></a>&nbsp;
    <a href="https://goreportcard.com/report/github.com/ddosify/ddosify" target="_blank"><img src="https://goreportcard.com/badge/github.com/ddosify/ddosify?style=for-the-badge&logo=go" alt="go report" /></a>&nbsp;
    <a href="https://github.com/ddosify/ddosify/blob/master/LICENSE" target="_blank"><img src="https://img.shields.io/badge/LICENSE-AGPL--3.0-orange?style=for-the-badge&logo=none" alt="ddosify license" /></a>
    <a href="https://discord.gg/9KdnrSUZQg" target="_blank"><img src="https://img.shields.io/discord/898523141788287017?style=for-the-badge&logo=discord&label=DISCORD" alt="ddosify discord server" /></a>
    <a href="https://hub.docker.com/r/ddosify/ddosify" target="_blank"><img src="https://img.shields.io/docker/v/ddosify/ddosify?style=for-the-badge&color=orange&logo=docker&label=docker" alt="ddosify docker image" /></a>
    
</p>


<p align="center">
<img src="https://raw.githubusercontent.com/ddosify/ddosify/master/assets/ddosify-quick-start.gif" alt="Ddosify - High-performance load testing tool quick start"  width="900px" />
</p>


## Features
:heavy_check_mark: Protocol Agnostic - Currently supporting *HTTP, HTTPS, HTTP/2*. Other protocols are on the way.

:heavy_check_mark: Scenario-Based - Create your flow in a JSON file. Without a line of code!

:heavy_check_mark: Different Load Types - Test your system's limits across different load types.

## Installation

`ddosify` is available via [Docker](https://hub.docker.com/r/ddosify/ddosify), [Homebrew Tap](#homebrew-tap-macos-and-linux), and downloadable pre-compiled binaries from the [releases page](https://github.com/ddosify/ddosify/releases/latest) for macOS, Linux and Windows.

### Docker

```bash
docker run -it --rm ddosify/ddosify
```

### Homebrew Tap (macOS and Linux)

```bash
brew install ddosify/tap/ddosify
```

### apk, deb, rpm, Arch Linux packages

- For arm architectures change `ddosify_amd64` to `ddosify_arm64` or `ddosify_armv6`.
- Superuser privilege is required.

```bash
# For Redhat based (Fedora, CentOS, RHEL, etc.)
rpm -i https://github.com/ddosify/ddosify/releases/latest/download/ddosify_amd64.rpm

# For Debian based (Ubuntu, Linux Mint, etc.)
wget https://github.com/ddosify/ddosify/releases/latest/download/ddosify_amd64.deb
dpkg -i ddosify_amd64.deb

# For Alpine
wget https://github.com/ddosify/ddosify/releases/latest/download/ddosify_amd64.apk
apk add --allow-untrusted ddosify_amd64.apk

# For Arch Linux
git clone https://aur.archlinux.org/ddosify.git
cd ddosify
makepkg -sri
```

### Using the convenience script (macOS and Linux)

- The script requires root or sudo privileges to move ddosify binary to `/usr/local/bin`.
- The script attempts to detect your operating system (macOS or Linux) and architecture (arm64, x86, amd64) to download the appropriate binary from the [releases page](https://github.com/ddosify/ddosify/releases/latest).
- By default, the script installs the latest version of `ddosify`.
- If you have problems, check [common issues](#common-issues)
- Required packages: `curl` and `sudo`

```bash
curl -sSfL https://raw.githubusercontent.com/ddosify/ddosify/master/scripts/install.sh | sh
```

### Go install from source (macOS, Linux, Windows)

```bash
go install -v go.ddosify.com/ddosify@latest
```

## Easy Start
This section aims to show you how to use Ddosify without deep dive into its details easily.
    
1. ### Simple load test

		ddosify -t target_site.com

    The above command runs a load test with the default value that is 100 requests in 10 seconds.

2. ### Using some of the features

		ddosify -t target_site.com -n 1000 -d 20 -p HTTPS -m PUT -T 7 -P http://proxy_server.com:80

    Ddosify sends a total of *1000* *PUT* requests to *https://target_site.com* over proxy *http://proxy_server.com:80* in *20* seconds with a timeout of *7* seconds per request.

3. ### Usage for CI/CD pipelines (JSON output)

    	ddosify -t target_site.com -o stdout-json | jq .avg_duration

    Ddosify outputs the result in JSON format. Then `jq` (or any other command-line JSON processor) fetches the `avg_duration`. The rest depends on your CI/CD flow logic. 

4. ### Scenario based load test

		ddosify -config config_examples/config.json
    Ddosify first sends *HTTP/2 POST* request to *https://test_site1.com/endpoint_1* using basic auth credentials *test_user:12345* over proxy *http://proxy_host.com:proxy_port*  and with a timeout of *3* seconds. Once the response is received, HTTPS GET request will be sent to *https://test_site1.com/endpoint_2* along with the payload included in *config_examples/payload.txt* file with a timeout of 2 seconds. This flow will be repeated *20* times in *5* seconds and response will be written to *stdout*.

		
## Details

You can configure your load test by the CLI options or a config file. Config file supports more features than the CLI. For example, you can't create a scenario-based load test with CLI options.
### CLI Flags

```bash
ddosify [FLAG]
```

| <div style="width:90px">Flag</div> | Description                  | Type     | Default | Required?  |
| ------ | -------------------------------------------------------- | ------   | ------- | ---------  |
| `-t`   | Target website URL. Example: https://ddosify.com         | `string` | - | Yes        |
| `-n`   | Total request count                                      | `int`    | `100`   | No         |
| `-d`   | Test duration in seconds.                                | `int`    | `10`    | No         |
| `-p`   | Protocol of the request. Supported protocols are *HTTP, HTTPS*. HTTP/2 support is only available by using a config file as described. More protocols will be added.                                | `string`    | `HTTPS`    | No         |
| `-m`   | Request method. Available methods for HTTP(s) are *GET, POST, PUT, DELETE, UPDATE, PATCH* | `string`    | `GET`    | No  |
| `-b`   | The payload of the network packet. AKA body for the HTTP.  | `string`    | -    | No         |
| `-a`   | Basic authentication. Usage: `-a username:password`        | `string`    | -    | No         |
| `-h`   | Headers of the request. You can provide multiple headers with multiple `-h` flag.  | `string`| -    | No         |
| `-T`   | Timeout of the request in seconds.                       | `int`    | `5`    | No         |
| `-P`   | Proxy address as host:port. `-P http://user:pass@proxy_host.com:port'` | `string`    | -    | No |
| `-o`   | Test result output destination. Supported outputs are [*stdout, stdout-json*] Other output types will be added. | `string`    | `stdout`    | No |
| `-l`   | [Type](#load-types) of the load test. Ddosify supports 3 load types. | `string`    | `linear`    | No |
| `-config`   | [Config File](#config-file) of the load test. | `string`    | -    | No |
| `-version `   | Prints version, git commit, built date (utc), go information and quit | -    | -    | No |


### Load Types

#### Linear

```bash
ddosify -t target_site.com -l linear
```

Result:

![linear load](https://raw.githubusercontent.com/ddosify/ddosify/master/assets/linear.gif)

*Note:* If the request count is too low for the given duration, the test might be finished earlier than you expect.

#### Incremental

```bash
ddosify -t target_site.com -l incremental
```

Result:

![incremental load](https://raw.githubusercontent.com/ddosify/ddosify/master/assets/incremental.gif)


#### Waved

```bash
ddosify -t target_site.com -l waved
```

Result:

![waved load](https://raw.githubusercontent.com/ddosify/ddosify/master/assets/waved.gif)


### Config File

Config file lets you use all capabilities of Ddosify. 

The features you can use by config file;
- Scenario creation
- Custom load type creation
- Payload from a file
- Multipart/form-data payload
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

- `manual_load` *optional*

    If you are looking for creating your own custom load type, you can use this feature. The example below says that Ddosify will run the scenario 5 times, 10 times, and 20 times, respectively along with the provided durations. `request_count` and `duration` will be auto-filled by Ddosify according to `manual_load` configuration. In this example, `request_count` will be 35 and the `duration` will be 18 seconds.
    Also `manual_load` overrides `load_type` if you provide both of them. As a result, you don't need to provide these 3 parameters when using `manual_load`.
    ```json
    "manual_load": [
        {"duration": 5, "count": 5},
        {"duration": 6, "count": 10},
        {"duration": 7, "count": 20}
    ]
    ```

- `proxy` *optional*

    This is the equivalent of the `-P` flag.

- `output` *optional*

    This is the equivalent of the `-o` flag.

- `steps` *mandatory*

    This parameter lets you create your scenario. Ddosify runs the provided steps, respectively. For the given example file step id: 2 will be executed immediately after the response of step id: 1 is received. The order of the execution is the same as the order of the steps in the config file.
    
    **Details of each parameter for a step;**
    - `id` *mandatory*
    
        Each step must have a unique integer id.

    - `url` *mandatory*

        This is the equivalent of the `-t` flag.

    - `name` *optional* <a name="#step-name"></a>
    
        Name of the step.
    
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

    - `payload_multipart` *optional* <a name="#payload_multipart"></a>

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
                "value": "http://test.com/test.png",
                "type": "file",
                "src": "remote"
            }
        ]
        ```

        *Note:* Ddosify adds `Content-Type: multipart/form-data; boundary=[generated-boundary-value]` header to the request when using `payload_multipart`.

    - `timeout` *optional*

        This is the equivalent of the `-T` flag. 

    - `sleep` *optional* <a name="#sleep"></a>

        Sleep duration(ms) before executing the next step. Can be an exact duration or a range.

        **Example:** Sleep 1000ms after step-1;
        ```json
        "steps": [
            {
                "id": 1,
                "url": "target.com/endpoint1",
                "sleep": "1000"
            },
            {
                "id": 2,
                "url": "target.com/endpoint2",
            }
        ]
        ```

        **Example:** Sleep between 300ms-500ms after step-1;
        ```json
        "steps": [
            {
                "id": 1,
                "url": "target.com/endpoint1",
                "sleep": "300-500"
            },
            {
                "id": 2,
                "url": "target.com/endpoint2",
            }
        ]
        ```

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

## Common Issues

### macOS Security Issue

```
"ddosify" can’t be opened because Apple cannot check it for malicious software.
```

- Open `/usr/local/bin`
- Right click `ddosify` and select Open
- Select Open
- Close the opened terminal

## Communication

You can join our [Discord Server](https://discord.gg/9KdnrSUZQg) for issues, feature requests, feedbacks or anything else. 

## More

This repository includes the single-node version of the Ddosify Loader. Ddosify Cloud will be available soon. 
It will support multi-location based distributed load testing and more features. 

Join the waitlist: https://ddosify.com

## License

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
