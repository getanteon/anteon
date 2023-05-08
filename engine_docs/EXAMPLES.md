# Examples of how to use the Ddosify Engine

## Example 1: Cookie Support

Ddosify Engine supports cookies. If the engine mode is `distict-user` or `repeated-user` the engine will store the cookies in a cookie jar and use them in the next request. If the engine mode is `ddosify` which is default, the engine will not use the cookies in the cookie jar. You can enable and disable custom / initial cookies with `enabled` key in the `cookie_jar` section of the config file.

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
            "timeout": 3,
            "sleep": "1000",
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
            "timeout": 3,
            "sleep": "1000",
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

Run the engine with the following command:

```bash
ddosify -config config.json --debug
```

In the first step we are getting the CSRF token from the server response cookie and storing it in the environment variable `CSRFTOKEN`.

In the second step we are using the `{{CSRFTOKEN}}` variable to replace the CSRF token in the payload. In this example, we are using `x-www-form-urlencoded` as the content type, so the we set `email`, `password` and `csrfmiddlewaretoken` fields in the payload. The `csrfmiddlewaretoken` field is replaced with the `{{CSRFTOKEN}}` captured variable.

We also set an initial cookie (`platform`) in the `cookie_jar` section of the config file. The engine will also store the cookies in the cookie jar and use them in the requests.
