# gRPC xDS Loadbalancing

Sample gRPC client/server application using the **Experimental** [xDS-Based Global Load Balancing](https://github.com/grpc/proposal/blob/master/A27-xds-global-load-balancing.md)

As it is experimental, any of this can change at anytime and break...

_caveat emptor_

>> ..this repo and code is not supported by google..

The reason i wrote this was to understand what is going on and to dig into the bit left unanswered in [gRPC xDS example](https://github.com/grpc/grpc-go/blob/master/examples/features/xds/README.md)

_"This example doesn't include instructions to setup xDS environment. Please refer to documentation specific for your xDS management server."_


This sample app does really nothing special:

You run two gRPC servers on the same host on two different ports

You start an xDS server in go that replays the protocol to let the gRPC clients know where to connect to

When the client first bootstraps to the xDS server, it sends down instructions to connect directly to one gRPC server instance.

Then wait a minute (really)

The xDS server will rotate the valid backend endpoint targets 60 seconds after it starts up (a trivial example, truly).  The second target is the second port where the second gRPC endpoint is running.


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

Start gRPC Servers

```bash
cd app/
go run src/grpc_server.go --grpcport :50051 --servername server1
go run src/grpc_server.go --grpcport :50052 --servername server2
```

> **NOTE** If you change the ports or run more servers, ensure you update the list when you start the xDS server

## gRPC Client (DNS)

Enable debug tracing on the client; the whole intent is to see all the details!

```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

Use DNS as the bootstrap mechanism to connec to the server:

```bash
$ go run src/grpc_client.go --host dns:///be.cluster.local:50051

INFO: 2020/11/11 08:24:42 [core] parsed scheme: "dns"
INFO: 2020/11/11 08:24:42 [core] ccResolverWrapper: sending update to cc: {[{127.0.0.1:50051  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2020/11/11 08:24:42 [core] ClientConn switching balancer to "pick_first"
INFO: 2020/11/11 08:24:42 [core] Channel switches to new LB policy "pick_first"
INFO: 2020/11/11 08:24:42 [core] Subchannel Connectivity change to CONNECTING
INFO: 2020/11/11 08:24:42 [core] blockingPicker: the picked transport is not ready, loop back to repick
INFO: 2020/11/11 08:24:42 [core] pickfirstBalancer: UpdateSubConnState: 0xc0003edcc0, {CONNECTING <nil>}
INFO: 2020/11/11 08:24:42 [core] Channel Connectivity change to CONNECTING
INFO: 2020/11/11 08:24:42 [core] Subchannel picks a new address "127.0.0.1:50051" to connect
INFO: 2020/11/11 08:24:42 [core] Subchannel Connectivity change to READY
INFO: 2020/11/11 08:24:42 [core] pickfirstBalancer: UpdateSubConnState: 0xc0003edcc0, {READY <nil>}
INFO: 2020/11/11 08:24:42 [core] Channel Connectivity change to READY
2020/11/11 08:24:42 RPC Response: 0 message:"Hello unary RPC msg   from server1"
```

## xDS Server

Now start the xDS server:

```bash
cd xds
go run main.go --upstream_port=50051 --upstream_port=50052

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
      ]      
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
export GRPC_XDS_BOOTSTRAP=`pwd`/xds_bootstrap.json
```

Then:

```bash
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

```
$ go run src/grpc_client.go --host xds:///be-srv
2020/11/09 08:28:59 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:04 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:09 RPC Response: 2 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:14 RPC Response: 3 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:19 RPC Response: 4 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:24 RPC Response: 5 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:29 RPC Response: 6 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:34 RPC Response: 7 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:39 RPC Response: 8 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:44 RPC Response: 9 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:49 RPC Response: 10 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:54 RPC Response: 11 message:"Hello unary RPC msg   from server1" 
2020/11/09 08:29:59 RPC Response: 12 message:"Hello unary RPC msg   from server2"   <<<<<<<<
2020/11/09 08:30:04 RPC Response: 13 message:"Hello unary RPC msg   from server2" 
2020/11/09 08:30:09 RPC Response: 14 message:"Hello unary RPC msg   from server2" 

```

---

If you want more details...

## References

- [Envoy Listener proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto)
- [Envoy Cluster proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto)
- [Envoy Endpoint proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto)


### xDS Server start

```bash
$ go run main.go --upstream_port=50051 --upstream_port=50052
INFO[0000] [UpstreamPorts] 50051                        
INFO[0000] [UpstreamPorts] 50052                        
INFO[0000] Starting control plane                       
INFO[0000] management server listening                   port=18000
INFO[0070] OnStreamOpen 1 open for Type []              
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0070] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0070] >>>>>>>>>>>>>>>>>>> creating NodeID b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50051 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0070] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating snapshot Version 1 
INFO[0070] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Listener],  Response[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0070] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0070] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.RouteConfiguration],  Response[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0070] cb.Report()  callbacks                        fetches=0 requests=3
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0070] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Cluster],  Response[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0070] cb.Report()  callbacks                        fetches=0 requests=5
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0070] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0070] cb.Report()  callbacks                        fetches=0 requests=7
INFO[0070] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 



///  Backend service ROTATION here 




INFO[0130] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50052 
INFO[0130] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0130] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0130] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0130] >>>>>>>>>>>>>>>>>>> creating snapshot Version 2 
INFO[0130] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Listener],  Response[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0130] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0130] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0130] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0130] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Cluster],  Response[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0130] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0130] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.RouteConfiguration],  Response[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0130] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0130] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0130] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0130] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0130] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
```

### gRPC Client Call 

```log
$ $ go run src/grpc_client.go --host xds:///be-srv
INFO: 2020/11/11 08:26:53 [core] parsed scheme: "xds"
INFO: 2020/11/11 08:26:53 [xds] [xds-bootstrap] Got bootstrap file location from GRPC_XDS_BOOTSTRAP environment variable: /home/srashid/Desktop/grpc_xds/app/xds_bootstrap.json
INFO: 2020/11/11 08:26:53 [xds] [xds-bootstrap] Bootstrap content: {
  "xds_servers": [
    {
      "server_uri": "xds.domain.com:18000",
      "channel_creds": [
        {
          "type": "insecure"
        }
      ]      
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

INFO: 2020/11/11 08:26:53 [xds] [xds-bootstrap] Bootstrap config for creating xds-client: &{BalancerName:xds.domain.com:18000 Creds:0xc000534d28 TransportAPI:0 NodeProto:id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning"  CertProviderConfigs:map[]}
INFO: 2020/11/11 08:26:53 [xds] [xds-resolver 0xc000182b80] Creating resolver for target: {Scheme:xds Authority: Endpoint:be-srv}
INFO: 2020/11/11 08:26:53 [core] parsed scheme: ""
INFO: 2020/11/11 08:26:53 [core] scheme "" not registered, fallback to default scheme
INFO: 2020/11/11 08:26:53 [core] ccResolverWrapper: sending update to cc: {[{xds.domain.com:18000  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2020/11/11 08:26:53 [core] ClientConn switching balancer to "pick_first"
INFO: 2020/11/11 08:26:53 [core] Channel switches to new LB policy "pick_first"
INFO: 2020/11/11 08:26:53 [core] Subchannel Connectivity change to CONNECTING
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Created ClientConn to xDS server: xds.domain.com:18000
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Created
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] new watch for type ListenerResource, resource name be-srv
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] first watch for type ListenerResource, resource name be-srv, will send a new xDS request
INFO: 2020/11/11 08:26:53 [xds] [xds-resolver 0xc000182b80] Watch started on resource name be-srv with xds-client 0xc000790000
INFO: 2020/11/11 08:26:53 [core] pickfirstBalancer: UpdateSubConnState: 0xc000780230, {CONNECTING <nil>}
INFO: 2020/11/11 08:26:53 [core] Channel Connectivity change to CONNECTING
INFO: 2020/11/11 08:26:53 [core] Subchannel picks a new address "xds.domain.com:18000" to connect
INFO: 2020/11/11 08:26:53 [core] Subchannel Connectivity change to READY
INFO: 2020/11/11 08:26:53 [core] pickfirstBalancer: UpdateSubConnState: 0xc000780230, {READY <nil>}
INFO: 2020/11/11 08:26:53 [core] Channel Connectivity change to READY
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS stream created
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.Listener
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.Listener" value:"\n\006be-srv\232\001z\nx\n`type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager\022\024\032\022\n\002\032\000\022\014be-srv-route" > type_url:"type.googleapis.com/envoy.api.v2.Listener" nonce:"1" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv, type: *envoy_config_listener_v3.Listener, contains: name:"be-srv" api_listener:<api_listener:<type_url:"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager" value:"\032\022\n\002\032\000\022\014be-srv-route" > > 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with type *envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager, contains rds:<config_source:<ads:<> > route_config_name:"be-srv-route" > 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] LDS resource with name be-srv, value {RouteConfigName:be-srv-route} added to cache
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: ListenerResource, version: 1, nonce: 1
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" response_nonce:"1" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] xds: client received LDS update: {RouteConfigName:be-srv-route}, err: <nil>
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] new watch for type RouteConfigResource, resource name be-srv-route
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] first watch for type RouteConfigResource, resource name be-srv-route, will send a new xDS request
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.RouteConfiguration
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" value:"\n\014be-srv-route\022+\n\tbe-srv-vs\022\006be-srv\032\026\n\002\n\000\022\020\n\016be-srv-cluster" > type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" nonce:"2" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-route, type: *envoy_config_route_v3.RouteConfiguration, contains: name:"be-srv-route" virtual_hosts:<name:"be-srv-vs" domains:"be-srv" routes:<match:<prefix:"" > route:<cluster:"be-srv-cluster" > > > . Picking routes for current watching hostname be-srv
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] RDS resource with name be-srv-route, value {Routes:[0xc000331240]} added to cache
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: RouteConfigResource, version: 1, nonce: 2
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" response_nonce:"2" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] xds: client received RDS update: {Routes:[0xc000331240]}, err: <nil>
INFO: 2020/11/11 08:26:53 [xds] [xds-resolver 0xc000182b80] Received update on resource be-srv from xds-client 0xc000790000, generated service config: {"loadBalancingConfig":[{"xds_routing_experimental":{"action":{"be-srv-cluster_1589047803":{"childPolicy":[{"weighted_target_experimental":{"targets":{"be-srv-cluster":{"weight":1,"childPolicy":[{"cds_experimental":{"cluster":"be-srv-cluster"}}]}}}}]}},"route":[{"prefix":"","action":"be-srv-cluster_1589047803"}]}}]}
INFO: 2020/11/11 08:26:53 [core] ccResolverWrapper: sending update to cc: {[] 0xc000418540 0xc000806da8}
INFO: 2020/11/11 08:26:53 [core] ClientConn switching balancer to "xds_routing_experimental"
INFO: 2020/11/11 08:26:53 [core] Channel switches to new LB policy "xds_routing_experimental"
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Created
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] update with config &{LoadBalancingConfig:<nil> routes:[{path: prefix: regex: headers:[] fraction:<nil> action:be-srv-cluster_1589047803}] actions:map[be-srv-cluster_1589047803:{ChildPolicy:0xc000418160}]}, resolver state {Addresses:[] ServiceConfig:0xc000418540 Attributes:0xc000806da8}
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Created
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Created child policy 0xc0004186e0 of type weighted_target_experimental
INFO: 2020/11/11 08:26:53 [xds] [cds-lb 0xc00031e780] Created
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Created child policy 0xc00031e780 of type cds_experimental
INFO: 2020/11/11 08:26:53 [xds] [cds-lb 0xc00031e780] Received update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc0003f65c0,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:26:53 [core] Channel Connectivity change to CONNECTING
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] new watch for type ClusterResource, resource name be-srv-cluster
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] first watch for type ClusterResource, resource name be-srv-cluster, will send a new xDS request
INFO: 2020/11/11 08:26:53 [xds] [cds-lb 0xc00031e780] Watch started on resource name be-srv-cluster with xds-client 0xc000790000
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.Cluster
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.Cluster" value:"\n\016be-srv-cluster\032\004\n\002\032\000\020\003" > type_url:"type.googleapis.com/envoy.api.v2.Cluster" nonce:"3" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-cluster, type: *envoy_config_cluster_v3.Cluster, contains: name:"be-srv-cluster" type:EDS eds_cluster_config:<eds_config:<ads:<> > > 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with name be-srv-cluster, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] CDS resource with name be-srv-cluster, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: ClusterResource, version: 1, nonce: 3
INFO: 2020/11/11 08:26:53 [xds] [cds-lb 0xc00031e780] Watch update from xds-client 0xc000790000, content: {ServiceName:be-srv-cluster EnableLRS:false}
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Created
INFO: 2020/11/11 08:26:53 [xds] [cds-lb 0xc00031e780] Created child policy 0xc00031e840 of type eds_experimental
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" response_nonce:"3" 
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> BalancerName: ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster LrsLoadReportingServerName:<nil>}
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] new watch for type EndpointsResource, resource name be-srv-cluster
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] first watch for type EndpointsResource, resource name be-srv-cluster, will send a new xDS request
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Watch started on resource name be-srv-cluster with xds-client 0xc000790000
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" value:"\n\016be-srv-cluster\022C\n\034\n\013us-central1\022\rus-central1-a\022\036\020\001\n\032\n\030\n\026\022\020be.cluster.local\030\203\207\003\032\003\010\350\007" > type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" nonce:"4" 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-cluster, type: *envoy_config_endpoint_v3.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster" endpoints:<locality:<region:"us-central1" zone:"us-central1-a" > lb_endpoints:<endpoint:<address:<socket_address:<address:"be.cluster.local" port_value:50051 > > > health_status:HEALTHY > load_balancing_weight:<value:1000 > > 
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] EDS resource with name be-srv-cluster, value {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50051 HealthStatus:1 Weight:0}] ID:us-central1:us-central1-a: Priority:0 Weight:1000}]} added to cache
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: EndpointsResource, version: 1, nonce: 4
INFO: 2020/11/11 08:26:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" response_nonce:"4" 
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Watch update from xds-client 0xc000790000, content: {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50051 HealthStatus:1 Weight:0}] ID:us-central1:us-central1-a: Priority:0 Weight:1000}]}
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] New priority 0 added
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] New locality us-central1:us-central1-a: added
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Switching priority from unset to 0
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Created child policy 0xc00042a780 of type round_robin
INFO: 2020/11/11 08:26:53 [balancer] base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50051  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2020/11/11 08:26:53 [core] Subchannel Connectivity change to CONNECTING
INFO: 2020/11/11 08:26:53 [core] Subchannel picks a new address "be.cluster.local:50051" to connect
INFO: 2020/11/11 08:26:53 [balancer] base.baseBalancer: handle SubConn state change: 0xc00043a470, CONNECTING
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Balancer state update from locality us-central1:us-central1-a:, new state: {ConnectivityState:CONNECTING Picker:0xc00043a3c0}
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Child pickers with config: map[us-central1:us-central1-a::weight:1000,picker:0xc000808720,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:CONNECTING Picker:0xc000088cc0}
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Child pickers with config: map[be-srv-cluster:weight:1,picker:0xc000088cc0,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Balancer state update from locality be-srv-cluster_1589047803, new state: {ConnectivityState:CONNECTING Picker:0xc0004f2f30}
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc0004f2f30,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:26:53 [core] Subchannel Connectivity change to READY
INFO: 2020/11/11 08:26:53 [balancer] base.baseBalancer: handle SubConn state change: 0xc00043a470, READY
INFO: 2020/11/11 08:26:53 [roundrobin] roundrobinPicker: newPicker called with info: {map[0xc00043a470:{{be.cluster.local:50051  <nil> 0 <nil>}}]}
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Balancer state update from locality us-central1:us-central1-a:, new state: {ConnectivityState:READY Picker:0xc0004a70e0}
INFO: 2020/11/11 08:26:53 [xds] [eds-lb 0xc00031e840] Child pickers with config: map[us-central1:us-central1-a::weight:1000,picker:0xc0004a7110,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc00050ebc0}
INFO: 2020/11/11 08:26:53 [xds] [weighted-target-lb 0xc0004186e0] Child pickers with config: map[be-srv-cluster:weight:1,picker:0xc00050ebc0,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Balancer state update from locality be-srv-cluster_1589047803, new state: {ConnectivityState:READY Picker:0xc0004f31f0}
INFO: 2020/11/11 08:26:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc0004f31f0,state:READY,stateToAggregate:READY]



INFO: 2020/11/11 08:26:53 [core] Channel Connectivity change to READY
2020/11/11 08:26:53 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:26:58 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:03 RPC Response: 2 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:08 RPC Response: 3 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:13 RPC Response: 4 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:18 RPC Response: 5 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:23 RPC Response: 6 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:28 RPC Response: 7 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:33 RPC Response: 8 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:38 RPC Response: 9 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:43 RPC Response: 10 message:"Hello unary RPC msg   from server1" 
2020/11/11 08:27:48 RPC Response: 11 message:"Hello unary RPC msg   from server1" 



///  Backend service ROTATION here 



INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.Listener
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.Listener" value:"\n\006be-srv\232\001z\nx\n`type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager\022\024\032\022\n\002\032\000\022\014be-srv-route" > type_url:"type.googleapis.com/envoy.api.v2.Listener" nonce:"5" 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv, type: *envoy_config_listener_v3.Listener, contains: name:"be-srv" api_listener:<api_listener:<type_url:"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager" value:"\032\022\n\002\032\000\022\014be-srv-route" > > 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with type *envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager, contains rds:<config_source:<ads:<> > route_config_name:"be-srv-route" > 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] LDS resource with name be-srv, value {RouteConfigName:be-srv-route} added to cache
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: ListenerResource, version: 2, nonce: 5
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] xds: client received LDS update: {RouteConfigName:be-srv-route}, err: <nil>
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" value:"\n\016be-srv-cluster\022C\n\034\n\013us-central1\022\rus-central1-a\022\036\020\001\n\032\n\030\n\026\022\020be.cluster.local\030\204\207\003\032\003\010\350\007" > type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" nonce:"6" 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" response_nonce:"5" 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-cluster, type: *envoy_config_endpoint_v3.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster" endpoints:<locality:<region:"us-central1" zone:"us-central1-a" > lb_endpoints:<endpoint:<address:<socket_address:<address:"be.cluster.local" port_value:50052 > > > health_status:HEALTHY > load_balancing_weight:<value:1000 > > 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] EDS resource with name be-srv-cluster, value {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50052 HealthStatus:1 Weight:0}] ID:us-central1:us-central1-a: Priority:0 Weight:1000}]} added to cache
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: EndpointsResource, version: 2, nonce: 6
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.Cluster
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.Cluster" value:"\n\016be-srv-cluster\032\004\n\002\032\000\020\003" > type_url:"type.googleapis.com/envoy.api.v2.Cluster" nonce:"7" 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-cluster, type: *envoy_config_cluster_v3.Cluster, contains: name:"be-srv-cluster" type:EDS eds_cluster_config:<eds_config:<ads:<> > > 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with name be-srv-cluster, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] CDS resource with name be-srv-cluster, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Watch update from xds-client 0xc000790000, content: {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50052 HealthStatus:1 Weight:0}] ID:us-central1:us-central1-a: Priority:0 Weight:1000}]}
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: ClusterResource, version: 2, nonce: 7
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" response_nonce:"6" 
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received, type: type.googleapis.com/envoy.api.v2.RouteConfiguration
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" value:"\n\014be-srv-route\022+\n\tbe-srv-vs\022\006be-srv\032\026\n\002\n\000\022\020\n\016be-srv-cluster" > type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" nonce:"8" 
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Locality us-central1:us-central1-a: updated, weightedChanged: false, addrsChanged: true
INFO: 2020/11/11 08:27:53 [xds] [cds-lb 0xc00031e780] Watch update from xds-client 0xc000790000, content: {ServiceName:be-srv-cluster EnableLRS:false}
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Resource with name: be-srv-route, type: *envoy_config_route_v3.RouteConfiguration, contains: name:"be-srv-route" virtual_hosts:<name:"be-srv-vs" domains:"be-srv" routes:<match:<prefix:"" > route:<cluster:"be-srv-cluster" > > > . Picking routes for current watching hostname be-srv
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] RDS resource with name be-srv-route, value {Routes:[0xc000089040]} added to cache
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] Sending ACK for response type: RouteConfigResource, version: 2, nonce: 8
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] xds: client received RDS update: {Routes:[0xc000089040]}, err: <nil>
INFO: 2020/11/11 08:27:53 [xds] [xds-resolver 0xc000182b80] Received update on resource be-srv from xds-client 0xc000790000, generated service config: {"loadBalancingConfig":[{"xds_routing_experimental":{"action":{"be-srv-cluster_1589047803":{"childPolicy":[{"weighted_target_experimental":{"targets":{"be-srv-cluster":{"weight":1,"childPolicy":[{"cds_experimental":{"cluster":"be-srv-cluster"}}]}}}}]}},"route":[{"prefix":"","action":"be-srv-cluster_1589047803"}]}}]}
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" response_nonce:"7" 
INFO: 2020/11/11 08:27:53 [core] ccResolverWrapper: sending update to cc: {[] 0xc00000fe20 0xc0007840b8}
INFO: 2020/11/11 08:27:53 [balancer] base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50052  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] update with config &{LoadBalancingConfig:<nil> routes:[{path: prefix: regex: headers:[] fraction:<nil> action:be-srv-cluster_1589047803}] actions:map[be-srv-cluster_1589047803:{ChildPolicy:0xc00000fb00}]}, resolver state {Addresses:[] ServiceConfig:0xc00000fe20 Attributes:0xc0007840b8}
INFO: 2020/11/11 08:27:53 [xds] [xds-client 0xc000790000] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.33.2" user_agent_name:"gRPC Go" user_agent_version:"1.33.2" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" response_nonce:"8" 
INFO: 2020/11/11 08:27:53 [core] Subchannel Connectivity change to CONNECTING
INFO: 2020/11/11 08:27:53 [xds] [cds-lb 0xc00031e780] Received update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2020/11/11 08:27:53 [core] Subchannel Connectivity change to SHUTDOWN
INFO: 2020/11/11 08:27:53 [core] Subchannel picks a new address "be.cluster.local:50052" to connect
INFO: 2020/11/11 08:27:53 [transport] transport: loopyWriter.run returning. connection error: desc = "transport is closing"
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> BalancerName: ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster LrsLoadReportingServerName:<nil>}
INFO: 2020/11/11 08:27:53 [balancer] base.baseBalancer: handle SubConn state change: 0xc0004f3c00, CONNECTING
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Balancer state update from locality us-central1:us-central1-a:, new state: {ConnectivityState:READY Picker:0xc0004a70e0}
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Child pickers with config: map[us-central1:us-central1-a::weight:1000,picker:0xc000596f90,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc00050f940}
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Child pickers with config: map[be-srv-cluster:weight:1,picker:0xc00050f940,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Balancer state update from locality be-srv-cluster_1589047803, new state: {ConnectivityState:READY Picker:0xc0004f3ea0}
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc0004f3ea0,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:27:53 [balancer] base.baseBalancer: handle SubConn state change: 0xc00043a470, SHUTDOWN
INFO: 2020/11/11 08:27:53 [roundrobin] roundrobinPicker: newPicker called with info: {map[]}
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Balancer state update from locality us-central1:us-central1-a:, new state: {ConnectivityState:CONNECTING Picker:0xc0004f3fe0}
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Child pickers with config: map[us-central1:us-central1-a::weight:1000,picker:0xc000597140,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:CONNECTING Picker:0xc00050f9c0}
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Child pickers with config: map[be-srv-cluster:weight:1,picker:0xc00050f9c0,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Balancer state update from locality be-srv-cluster_1589047803, new state: {ConnectivityState:CONNECTING Picker:0xc000644140}
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc000644140,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2020/11/11 08:27:53 [core] Channel Connectivity change to CONNECTING
INFO: 2020/11/11 08:27:53 [core] Subchannel Connectivity change to READY
INFO: 2020/11/11 08:27:53 [balancer] base.baseBalancer: handle SubConn state change: 0xc0004f3c00, READY
INFO: 2020/11/11 08:27:53 [roundrobin] roundrobinPicker: newPicker called with info: {map[0xc0004f3c00:{{be.cluster.local:50052  <nil> 0 <nil>}}]}
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Balancer state update from locality us-central1:us-central1-a:, new state: {ConnectivityState:READY Picker:0xc00078c6c0}
INFO: 2020/11/11 08:27:53 [xds] [eds-lb 0xc00031e840] Child pickers with config: map[us-central1:us-central1-a::weight:1000,picker:0xc00078c720,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc000430c80}
INFO: 2020/11/11 08:27:53 [xds] [weighted-target-lb 0xc0004186e0] Child pickers with config: map[be-srv-cluster:weight:1,picker:0xc000430c80,state:READY,stateToAggregate:READY]
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Balancer state update from locality be-srv-cluster_1589047803, new state: {ConnectivityState:READY Picker:0xc00043a800}
INFO: 2020/11/11 08:27:53 [xds] [xds-routing-lb 0xc000331500] Child pickers with routes: [pathPrefix:->be-srv-cluster_1589047803], actions: map[be-srv-cluster_1589047803:picker:0xc00043a800,state:READY,stateToAggregate:READY]


INFO: 2020/11/11 08:27:53 [core] Channel Connectivity change to READY
2020/11/11 08:27:53 RPC Response: 12 message:"Hello unary RPC msg   from server2" 
2020/11/11 08:27:58 RPC Response: 13 message:"Hello unary RPC msg   from server2" 
2020/11/11 08:28:03 RPC Response: 14 message:"Hello unary RPC msg   from server2" 
```

