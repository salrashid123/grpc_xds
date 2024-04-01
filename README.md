# gRPC xDS Loadbalancing

Sample gRPC client/server application using  [xDS-Based Global Load Balancing](https://godoc.org/google.golang.org/grpc/xds)


The reason i wrote this was to understand what is going on and to dig into the bit left unanswered in [gRPC xDS example](https://github.com/grpc/grpc-go/blob/master/examples/features/xds/README.md)

_"This example doesn't include instructions to setup xDS environment. Please refer to documentation specific for your xDS management server."_


This sample app does really nothing special:

You run three gRPC servers on the same host on different named ports

You start an xDS server and specify each host:port of the backends clients should connect to (i.e, the three grpc server)

The xDS server will iterate over the list of backends services every 10 seconds and add the next server as a valid target

When the client first bootstraps to the xDS server, it sends down instructions to connect directly to one gRPC server instance.

The xDS server will iterate every 10 seconds and add new backends so your client would see a new backend that handled the request.


thats it.


ref:

- [Example xDS Server](https://github.com/envoyproxy/go-control-plane/tree/master/internal/example)

---

## Setup

edit `/etc/hosts`

```bash
127.0.0.1 be.cluster.local xds.domain.com
```

## gRPC Server

Start three gRPC Servers in new windows

```bash
cd app/
go run src/grpc_server.go --grpcport :50051 --servername server1
go run src/grpc_server.go --grpcport :50052 --servername server2
go run src/grpc_server.go --grpcport :50053 --servername server3
```

> **NOTE** If you change the ports or run more servers, ensure you update the list when you start the xDS server

## gRPC Client (DNS)

(optionally) Enable debug tracing on the client; the whole intent is to see all the details...but its a lot of logs


```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

Use DNS as the bootstrap mechanism to connect to the server:

```bash
$ go run src/grpc_client.go --host dns:///be.cluster.local:50051

2022/06/04 08:02:40 INFO: [core] [Channel #1] Channel created
2022/06/04 08:02:40 INFO: [core] [Channel #1] original dial target is: "dns:///be.cluster.local:50051"
2022/06/04 08:02:40 INFO: [core] [Channel #1] parsed dial target is: {Scheme:dns Authority: Endpoint:be.cluster.local:50051 URL:{Scheme:dns Opaque: User: Host: Path:/be.cluster.local:50051 RawPath: ForceQuery:false RawQuery: Fragment: RawFragment:}}
2022/06/04 08:02:40 INFO: [core] [Channel #1] Channel authority set to "be.cluster.local:50051"
2022/06/04 08:02:40 INFO: [core] [Server #2] Server created
2022/06/04 08:02:40 WARNING: [xds] failed to create xds client: xds: failed to read bootstrap file: xds: Failed to read bootstrap config: none of the bootstrap environment variables ("GRPC_XDS_BOOTSTRAP" or "GRPC_XDS_BOOTSTRAP_CONFIG") defined
2022/06/04 08:02:40 Admin port listen on :[::]:19000
2022/06/04 08:02:40 INFO: [core] [Server #2 ListenSocket #3] ListenSocket created
2022/06/04 08:02:40 INFO: [core] [Channel #1] Resolver state updated: {
  "Addresses": [
    {
      "Addr": "127.0.0.1:50051",
      "ServerName": "",
      "Attributes": null,
      "BalancerAttributes": null,
      "Type": 0,
      "Metadata": null
    }
  ],
  "ServiceConfig": null,
  "Attributes": null
} (resolver returned new addresses)
2022/06/04 08:02:40 INFO: [core] [Channel #1] Channel switches to new LB policy "pick_first"
2022/06/04 08:02:40 INFO: [core] [Channel #1 SubChannel #4] Subchannel created
2022/06/04 08:02:40 INFO: [core] [Channel #1 SubChannel #4] Subchannel Connectivity change to CONNECTING
2022/06/04 08:02:40 INFO: [core] [Channel #1 SubChannel #4] Subchannel picks a new address "127.0.0.1:50051" to connect
2022/06/04 08:02:40 INFO: [core] pickfirstBalancer: UpdateSubConnState: 0xc0002faaf0, {CONNECTING <nil>}
2022/06/04 08:02:40 INFO: [core] [Channel #1] Channel Connectivity change to CONNECTING
2022/06/04 08:02:40 INFO: [core] blockingPicker: the picked transport is not ready, loop back to repick
2022/06/04 08:02:40 INFO: [core] [Channel #1 SubChannel #4] Subchannel Connectivity change to READY
2022/06/04 08:02:40 INFO: [core] pickfirstBalancer: UpdateSubConnState: 0xc0002faaf0, {READY <nil>}
2022/06/04 08:02:40 INFO: [core] [Channel #1] Channel Connectivity change to READY
2022/06/04 08:02:40 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:02:42 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:02:44 RPC Response: 2 message:"Hello unary RPC msg   from server1"
```

Ignore the warning about xds...we're not even using it yet and that warning shows up because we've loaded the xds resolver on import

```golang
import (
	_ "google.golang.org/grpc/resolver" // use for "dns:///be.cluster.local:50051"
	_ "google.golang.org/grpc/xds"      // use for xds-experimental:///be-srv
)
```


## xDS Server

Now start the xDS server:

```bash
cd xds
go run main.go --upstream_port=50051 --upstream_port=50052 --upstream_port=50053

INFO[0000] Starting control plane                       
INFO[0000] management server listening                   port=18000
```

> **NOTE** Update the list of `--upstream_port` flags to reflect the list of ports for the gRPC servers that you started


## gRPC Client (xDS)

Ensure the xds Bootstrap file specifies the correct `server_url`

The grpc client will connect to this as the the xDS server.  The gRPC client library looks for a specific env-var (`GRPC_XDS_BOOTSTRAP`) that points to the file 

- `xds_bootstrap.json`:
```json
{
  "xds_servers": [
    {
      "server_uri": "xds.domain.com:18000",
      "channel_creds": [
        {
          "type": "insecure"
        }
      ],
      "server_features": ["xds_v3"]     
    }
  ],
  "node": {
    "id": "b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1",
    "metadata": {
      "R_GCP_PROJECT_NUMBER": "123456789012"
    },
    "locality": {
      "zone": "us-central1-a"
    }
  }
}
```

Then:

```bash
cd app/

export GRPC_XDS_BOOTSTRAP=`pwd`/xds_bootstrap.json

go run src/grpc_client.go --host xds:///be-srv
```

in the debug logs that it connected to port `:50051`

```console
INFO: 2020/04/21 16:14:42 Subchannel picks a new address "be.cluster.local:50051" to connect
```

The grpc client will issue grpc requests every 5 seconds using the list of backend services it gets from the xds server.
Since the xds server will abruptly rotate the grpc backend servers, the client will suddenly connect to `server2`

```console
INFO: 2020/04/21 16:16:08 Subchannel picks a new address "be.cluster.local:50052" to connect
```

The port it connected to is `:50052`

Right, that's it!

---

```log
unset GRPC_GO_LOG_VERBOSITY_LEVEL
unset GRPC_GO_LOG_SEVERITY_LEVEL

 go run src/grpc_client.go --host xds:///be-srv


2022/06/04 08:09:37 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:39 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:41 RPC Response: 2 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:43 RPC Response: 3 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:45 RPC Response: 4 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:47 RPC Response: 5 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:49 RPC Response: 6 message:"Hello unary RPC msg   from server2"    /// ** new server added
2022/06/04 08:09:51 RPC Response: 7 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:53 RPC Response: 8 message:"Hello unary RPC msg   from server2" 
2022/06/04 08:09:55 RPC Response: 9 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:09:57 RPC Response: 10 message:"Hello unary RPC msg   from server2" 
2022/06/04 08:09:59 RPC Response: 11 message:"Hello unary RPC msg   from server3"  /// ** new server added
2022/06/04 08:10:01 RPC Response: 12 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:10:03 RPC Response: 13 message:"Hello unary RPC msg   from server2" 
2022/06/04 08:10:05 RPC Response: 14 message:"Hello unary RPC msg   from server3" 
2022/06/04 08:10:07 RPC Response: 15 message:"Hello unary RPC msg   from server1" 
2022/06/04 08:10:09 RPC Response: 16 message:"Hello unary RPC msg   from server2" 
2022/06/04 08:10:11 RPC Response: 17 message:"Hello unary RPC msg   from server3" 
```

---

If you want more details...

### xDS Server start

The xDS Server simply logs the snapshots that are rotated

```log
INFO[0000] [UpstreamPorts] 50051                        
INFO[0000] [UpstreamPorts] 50052                        
INFO[0000] [UpstreamPorts] 50053                        
INFO[0000] Starting control plane                       
INFO[0000] management server listening                   port=18000
INFO[0004] OnStreamOpen 1 open for Type []              
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0004] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0004] >>>>>>>>>>>>>>>>>>> creating NodeID b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1 
INFO[0004] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50051 
INFO[0004] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0004] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0004] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0004] >>>>>>>>>>>>>>>>>>> creating snapshot Version 1 
INFO[0004] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.listener.v3.Listener],  Response[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0004] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0004] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.route.v3.RouteConfiguration],  Response[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0004] cb.Report()  callbacks                        fetches=0 requests=3
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0004] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.cluster.v3.Cluster],  Response[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0004] cb.Report()  callbacks                        fetches=0 requests=5
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0004] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0004] cb.Report()  callbacks                        fetches=0 requests=7
INFO[0004] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0014] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50052 
INFO[0014] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0014] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0014] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0014] >>>>>>>>>>>>>>>>>>> creating snapshot Version 2 
INFO[0014] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.route.v3.RouteConfiguration],  Response[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0014] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0014] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.cluster.v3.Cluster],  Response[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0014] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0014] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.listener.v3.Listener],  Response[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0014] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0014] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0014] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0014] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0014] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0014] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0014] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0024] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50053 
INFO[0024] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0024] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0024] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0024] >>>>>>>>>>>>>>>>>>> creating snapshot Version 3 
INFO[0024] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.route.v3.RouteConfiguration],  Response[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0024] cb.Report()  callbacks                        fetches=0 requests=12
INFO[0024] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0024] cb.Report()  callbacks                        fetches=0 requests=12
INFO[0024] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.cluster.v3.Cluster],  Response[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0024] cb.Report()  callbacks                        fetches=0 requests=12
INFO[0024] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.listener.v3.Listener],  Response[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0024] cb.Report()  callbacks                        fetches=0 requests=12
INFO[0024] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0024] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0024] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0024] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.listener.v3.Listener] 
2022/06/04 08:10:27 INFO: [transport] transport: loopyWriter.run returning. connection error: desc = "transport is closing"
INFO[0054] OnStreamClosed 1 closed 
```

### gRPC Client Call 


The client verbose logs shows all the details

[grpc_client log](https://gist.github.com/salrashid123/f266c1b5cee60bc9be9e602c97595151)

### xDS debug client

`grpc_client.go` also includes a debug listener that is started as its own go routine on port `:19000`.

- [https://github.com/grpc-ecosystem/grpcdebug#debug-xds](https://github.com/grpc-ecosystem/grpcdebug#debug-xds)

Which means that while the xDS client is running, you can interrogate it for the configuration and other statistics

```
$ grpcdebug localhost:19000 xds status
Name             Status    Version   Type                                                                 LastUpdated
be-srv           ACKED     1         type.googleapis.com/envoy.config.listener.v3.Listener                16 seconds ago   
be-srv-route     ACKED     1         type.googleapis.com/envoy.config.route.v3.RouteConfiguration         16 seconds ago   
be-srv-cluster   ACKED     1         type.googleapis.com/envoy.config.cluster.v3.Cluster                  16 seconds ago   
be-srv-cluster   ACKED     1         type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment   16 seconds ago   
```
