# Self hosted

## Add New Engine

The Ddosify Self-Hosted includes a default engine out of the box. To integrate additional engines, simply run a Docker container for each new engine. These engine containers will automatically register with the service and become available for use. Before adding new engines, ensure that you have enabled the distributed mode by clicking the `Unlock the Distributed Mode` button in the dashboard.

Make sure the new engine server can access the service server. Use the `DDOSIFY_SERVICE_ADDRESS` environment variable to specify the service server address where the install.sh script was executed.

The engine server must connect to the following ports on the `DDOSIFY_SERVICE_ADDRESS`:

- `8001`: Hammer Manager service. The service server utilizes this port to register the engine.
- `5672`: RabbitMQ server. The engine server connects to this port to send and receive messages to and from the service server.
- `8086`: InfluxDB server. The engine server accesses this port to transmit metrics to the backend.
- `9000`: Object storage server. The engine server uses this port to exchange files with the service server.

If you are adding the engine to the same server where the install.sh script was run, set `DDOSIFY_SERVICE_ADDRESS=host.docker.internal`

In case you have modified the default values like InfluxDB password in the `.env` file, utilize the `--env` flag in the docker run command to establish the necessary environment variables.

```bash
# Make sure to set the following environment variables
DDOSIFY_SERVICE_ADDRESS=SERVICE_ADDRESS
IP_ADDRESS=ENGINE_IP_ADDRESS
NAME=ddosify_hammer_1

docker run --name $NAME -dit \
    --env DDOSIFY_SERVICE_ADDRESS=$DDOSIFY_SERVICE_ADDRESS \
    --env IP_ADDRESS=$IP_ADDRESS \
    --network ddosify \
    --restart always \
    ddosify/selfhosted_hammer
```

You should see `mq_waiting_new_job` log in the engine container logs. This means that the engine is waiting for a job from the service server. After the engine is added, you can see it in the Engines page.
