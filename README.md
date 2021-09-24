# **Ddosify** - High-performance load testing tool

TODO: Logo
TODO: Badges

## Features
- :heavy_check_mark: Protocol Agnostic - Currently supporting HTTP, HTTPS, HTTP/2. Other protocols are on the way.
- :heavy_check_mark: Scenario Based - Create your flow with a Json file. Without a line of code!
- :heavy_check_mark: Different Load Types - Test your system's limits across different load types.

TODO: GIF KOY

## Installation

`ddosify` is available via [Docker](https://hub.docker.com/ddosify/ddosify), [Homebrew](https://formulae.brew.sh/formula/ddosify), [Homebrew Tap](), [Conda]() and as a downloadable pre-compiled binaries from the [releases page](https://github.com/ddosify/ddosify/releases/latest).

### Docker

```bash
docker run -it --rm --name ddosify ddosify/ddosify
```

### Homebrew (macOS and Linux)

```bash
brew install ddosify
```

### Homebrew Tap (macOS and Linux)

```bash
brew install ddosify/ddosify
```

### Conda (macOS, Linux and Windows)

```bash
conda install ddosify --channel conda-forge
```
### Using the convenience script (macOS and Linux)

- The script requires root or sudo privileges to move ddosify binary to `/usr/bin`.
- The script attempts to detect your operating system (macos or linux) and architecture (arm64, x86, x8664, i386) to download the appropriate binary from the [releases page](https://github.com/ddosify/ddosify/releases/latest).
- By default, the scripts installs the latest version of `ddosify`. 

TODO: Change URL
```bash
curl -sSfL https://raw.githubusercontent.com/ddosify/hammer/master/scripts/install.sh | sh
```

## Easy Start
This section aims to show how to use Ddosify without deepig dive into its details. TODO: 
    
1. ### Simple load test

		ddosify -t target_site.com

    Above command runs a load test with the default values that is 200 requests in 10 seconds.

2. ### Using some of the features.

		ddosify -t target_site.com -n 1000 -d 20 -p HTTPS -m PUT -T 7 -P http://proxy_server.com:80

    Above command tells Ddosify to send total *1000* PUT request to *https://target_site.com* over proxy *http://proxy_server.com:80* in *20* seconds with timeout *7* seconds. 

3. ### Scenario based load test

		ddosify -t config_examples/config.json
    
    Ddosify first send *HTTP/2 POST* request to *https://test_site1.com/endpoint_1* using basic auth credentials *test_user:12345* over proxy *http://proxy_host.com:proxy_port*  and with timeout *3* sconds. Once the response recieved, HTTPS GET request will be send to *https://test_site1.com/endpoint_2* along with the payload included in *config_examples/payload.txt* file with timeout 2 seconds. This flow will be repeated *20* times in *5* seconds and response will be written to *stdout*.

		
## Details

You can configure your load test by the CLI options or a config file. Config file supports more features than the CLI. For example you can't create scenario based load test with CLI options.

### CLI Flags

    ddosify -t <target_website> [options...]

1. `-n`

    Total request count. Default is 200.
2. `-d`

    Test duration in seconds. Default is 10 second.
3. `-l`

    Type of the load test. Default is "linear". Ddosify supports 3 load types;
    1. `-l linear`

        Example; 

            ddosify -t target_site.com -n 200 -d 10 -l linear

        Result;

        ![enter image description here](assets/linear.gif)

        *Note:* If the request count is too low for the given duration, the test would be finished earlier than you expect. 10 request per second is the lower limit to run this load type smoothly.

    2. `-l incremental`
    
        Example;

            ddosify -t target_site.com -n 200 -d 10 -l incremental

        Result;

        ![enter image description here](assets/incremental.gif)
        
    3. `-l waved`
        
        Example;

            ddosify -t target_site.com -n 400 -d 16 -l waved

        Result;

        ![enter image description here](assets/waved.gif)

        *Note:* Wave count equals to `log2(duration) / 2`.
4. `-p`

    Protocol of the request. Defaul is HTTPS. Supported protocols [HTTP, HTTPS]. HTTP/2 support only available by using config file as described here.(TODO: href).More protocols will be added.
    
    *Note:* If the target url passed with `-t` option includes protocol inside of it, then the value of the `-p` will be ignored.

5. `-m`

    Request method. Default is GET. For Http(s):[GET, POST, PUT, DELETE, UPDATE, PATCH]

6. `-b` 

    Payload of the network packet. AKA body for the HTTP.

7. `-a`

    Basic authentication. 

        ddosify -t target_site.com -a username:password

8. `-h`

    Request headers. You can provide multiple headers.

        ddosify -t target_site.com -h 'Accept: text/html' -h 'Content-Type: application/xml'

9. `-T`

    Request timeout in seconds. Default is 5 second.

10. `-P`

    Proxy address as host:port. 

        ddosify -t target_site.com -P http://proxy_host.com:port'

11. `-o`

    Test result output destination. Default is *stdout*. Other output types will be added.

### Config File

Config file  lets you to use all capabilities of the Ddosify. 

The features that can only usable by a config file;
- Scneario creation.
- Payload from a file.
- Extra connection configuration, like *keep-alive* enable/disable logic.
- HTTP2 support. 

Usage;

    ddosify -config <json_config_path>


Only Json formatted file is supported for now. There is an example config file at config_examples/config.json (TODO: link koy). This file contains all of the parameters you can use. Details of the each parameters;

1. `request_count` *optional*

    This is the equilevent of the `-n` flag. The difference is that if you have multiple steps in your scenario than this value represents the iteration count of your steps.

2. `load_type` *optional*

    This is the equilevent of the `-l` flag.

3. `duration` *optional*

    This is the equilevent of the `-d` flag.

4. `proxy` *optional*

    This is the equilevent of the `-P` flag.

5. `output` *optional*

    This is the equilevent of the `-o` flag.

6. `steps` *mandatory*

    This parameter lets you create your own scenario. Ddosify runs the provided steps sync. For the given example file step with ID:2 will be executed immediately after the response of step with ID:1 recieved. The order of the execution is same as the order of the steps in the config file.
    
    Details of the each parameters for a step;
    1. `id` *mandatory*
    
        Each step must have a unique integer id.

    2. `url` *mandatory*

        This is the equilevent of the `-t` flag.
    
    3. `protocol` *optional*

        This is the equilevent of the `-p` flag.

    4. `method` *optional*

        This is the equilevent of the `-m` flag.

    5. `headers` *optional*

        List of headers with key:value format.

    6. `payload` *optional*

        This is the equilevent of the `-b` flag.

    7. `payload_file` *optional*

        If you need a long payload, we suggests to use this parameter instead of `payload`.  

    7. `auth` *optional*
        
        Basic authentication.
        ```json
        "auth": {
            "username": "test_user",
            "password": "12345"
        }
        ```
    8. `others` *optional*

        This parameter accepts dynamic *key:value* pairs to configure connection details of the protocol in use.
        
        ```json
        "others": {
            "keep-alive": true,              // Default false
            "disable-compression": false,    // Default true
            "h2": true,                      // Enables HTTP/2. Default is false.
            "disable-redirect": true         // Default false
        }
        ```




    


  

## Future
TODO: Our motivation & future of the ddosify.

