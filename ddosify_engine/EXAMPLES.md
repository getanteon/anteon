# Examples of how to use the Ddosify Engine

## Example 1: Cookie Support

Ddosify Engine supports cookies. If the engine mode is `distict-user` or `repeated-user` the engine will store the cookies in a cookie jar and use them in the next request. If the engine mode is `ddosify` which is default, the engine will not use the cookies in the cookie jar. You can enable and disable custom / initial cookies with `enabled` key in the `cookie_jar` section of the config file. For more: [Cookies](./README.md#cookies).

- Create a Ddosify configuration file called: `config.json`
```json
{
    "iteration_count": 100,
    "load_type": "waved",
    "engine_mode": "distinct-user",
    "duration": 5,
    "steps": [
        {
            "id": 1,
            "name": "Get Servers",
            "url": "https://app.servdown.com/servers/",
            "method": "GET",
            "timeout": 10,
            "others": {
            },
            "capture_env": {
                "CSRFTOKEN" :{"from":"cookies","cookie_name":"csrftoken"}
              }
        },
        {
            "id": 2,
            "name": "Login",
            "url": "https://app.servdown.com/accounts/login/?next=/",
            "method": "POST",
            "timeout": 10,
            "headers": {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            "payload": "email=user@gmail.com&password=PasssChange&csrfmiddlewaretoken={{CSRFTOKEN}}",
            "others": {
                "disable-redirect": false
            }
        }
    ],
    "output": "stdout",
    "cookie_jar":{
        "enabled" : true,
        "cookies" :[
            {
                "name": "platform",
                "value": "web",
                "domain": "httpbin.ddosify.com",
                "path": "/",
                "expires": "Thu, 16 Mar 2023 09:24:02 GMT",
                "http_only": true,
                "secure": false
            }
        ]
    } 
}
```

- Run the engine with the following command:

```bash
ddosify -config config.json --debug
```

In the first step we are getting the CSRF token from the server response cookie and storing it in the environment variable `CSRFTOKEN`.

In the second step we are using the `CSRFTOKEN` variable to replace the CSRF token in the payload. In this example, we are using `x-www-form-urlencoded` as the content type, so the we set `email`, `password` and `csrfmiddlewaretoken` fields in the payload. The `csrfmiddlewaretoken` field is replaced with the `CSRFTOKEN` captured variable.

We also set an initial cookie (`platform`) in the `cookie_jar` section of the config file. The engine will also store the cookies in the cookie jar and use them in the requests.


## Example 2: Success Criteria

Ddosify engine supports success criteria. You can set a success criteria for the entire test based on server or assertion errors. If the success criteria is not met the engine will stop the test based on `abort` and `delay` keywords. If you set `abort` to `true` the engine will stop the test immediately after `delay` seconds. If you set `abort` to `false` the engine will not stop the test. For more: [Success Criteria](./README.md#success-criteria-pass--fail).

- Create a Ddosify configuration file called: `config.json`
```json
{
    "iteration_count": 20,
    "debug": false,
    "load_type": "linear",
    "duration": 2,
    "output": "stdout",
    "success_criterias": [
        {
            "rule": "fail_count < 5",
            "abort": false,
            "delay": 0
        },
        {
            "rule": "fail_count_perc < 0.1",
            "abort": true,
            "delay": 1
        },
        {
            "rule": "p90(iteration_duration) < 220",
            "abort": false
        }
    ],
    "steps": [
        {
            "id": 1,
            "url": "https://httpbin.ddosify.com/json",
            "method": "GET",
            "timeout": 5,
            "assertion": [
                "equals(status_code, 201)",
                "equals(json_path(\"quoteResponse.result.0.ask\"), 130.74)"
            ]
        },
        {
            "id": 2,
            "url": "https://httpbin.ddosify.com/status/500",
            "method": "GET",
            "timeout": 5,
            "assertion": [
                "in(status_code, [413,403])",
                "less_than(response_time, 100)"
            ]
        }
    ]
}
```

- Run the engine with the following command:

```bash
ddosify -config config.json
```

<details>
    <summary> Example Output</summary>
    
```
⚙️  Initializing... 
🔥 Engine fired. 

🛑 CTRL+C to gracefully stop.
✔️  Successful Run: 0        0%       ❌ Failed Run: 12     100%       ⏱️  Avg. Duration: 0.00000s
✔️  Successful Run: 0        0%       ❌ Failed Run: 20     100%       ⏱️  Avg. Duration: 0.00000s


RESULT
-------------------------------------

1. Step 1
---------------------------------
Success Count:    0     (0%)
Failed Count:     20    (100%)

Durations (Avg):
  DNS                  :0.0029s
  Connection           :0.0389s
  TLS                  :0.0269s
  Request Write        :0.0000s
  Server Processing    :0.1304s
  Response Read        :0.0001s
  Total                :0.1992s

Status Code (Message) :Count
  200 (OK)    :20

Assertion Error Distribution:
    Condition: equals(status_code, 201)
        Fail Count: 20
        Received: 
             equals(status_code,201) : [false]
             status_code : [200]
        Reason: expression evaluated to false 



2. Step 2
---------------------------------
Success Count:    0     (0%)
Failed Count:     20    (100%)

Durations (Avg):
  DNS                  :0.0016s
  Connection           :0.0578s
  TLS                  :0.0470s
  Request Write        :0.0000s
  Server Processing    :0.1297s
  Response Read        :0.0000s
  Total                :0.2361s

Status Code (Message) :Count
  500 (Internal Server Error)    :20

Assertion Error Distribution:
    Condition: in(status_code, [413,403])
        Fail Count: 20
        Received: 
             status_code : [500]
             in(status_code,[413,403]) : [false]
        Reason: expression evaluated to false 

    Condition: less_than(response_time, 100)
        Fail Count: 20
        Received: 
             less_than(response_time,100) : [false]
             response_time : [397 392 399 129 130]
        Reason: expression evaluated to false 



Test Status: Failed

Rule: fail_count < 5
Received: 
    fail_count: 20

Rule: fail_count_perc < 0.1
Received: 
    fail_count_perc: 1

Rule: p90(iteration_duration) < 220
Received: 
    iteration_duration: [257 258 258 258 259 259 260 260 261 262 263 388 390 526 654 662 793 795 800 822]
    p90(iteration_duration): 795
exit status 1
```
</details>

In this example, we are using the `fail_count`,  `fail_count_perc` and `p90(iteration_duration)` success criteria. The engine will stop the test if any of the success criteria is not met. In this example, the engine will stop the test because the `fail_count` is `20` which is greater than `5`, `fail_count_perc` is `1` which is greater than `0.1` and `p90(iteration_duration)` is `795` which is greater than `220`.

