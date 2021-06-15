# gRPC xDS Loadbalancing

Sample gRPC client/server application using  [xDS-Based Global Load Balancing](https://godoc.org/google.golang.org/grpc/xds)

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

edit `src/grpc_client.go`, set to use the default dns resolver

```golang
import (
	_ "google.golang.org/grpc/resolver" // use for "dns:///be.cluster.local:50051"
	//_ "google.golang.org/grpc/xds"      // use for xds-experimental:///be-srv
)
```

Use DNS as the bootstrap mechanism to connect to the server:

```bash
$ go run src/grpc_client.go --host dns:///be.cluster.local:50051

INFO: 2021/06/15 10:36:07 [core] Channel Created
INFO: 2021/06/15 10:36:07 [core] parsed scheme: "dns"
INFO: 2021/06/15 10:36:07 [core] ccResolverWrapper: sending update to cc: {[{127.0.0.1:50051  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2021/06/15 10:36:07 [core] Resolver state updated: {Addresses:[{Addr:127.0.0.1:50051 ServerName: Attributes:<nil> Type:0 Metadata:<nil>}] ServiceConfig:<nil> Attributes:<nil>} (resolver returned new addresses)
INFO: 2021/06/15 10:36:07 [core] ClientConn switching balancer to "pick_first"
INFO: 2021/06/15 10:36:07 [core] Channel switches to new LB policy "pick_first"
INFO: 2021/06/15 10:36:07 [core] Subchannel Created
INFO: 2021/06/15 10:36:07 [core] Subchannel(id:4) created
INFO: 2021/06/15 10:36:07 [core] Subchannel Connectivity change to CONNECTING
INFO: 2021/06/15 10:36:07 [core] blockingPicker: the picked transport is not ready, loop back to repick
INFO: 2021/06/15 10:36:07 [core] pickfirstBalancer: UpdateSubConnState: 0xc000099fc0, {CONNECTING <nil>}
INFO: 2021/06/15 10:36:07 [core] Channel Connectivity change to CONNECTING
INFO: 2021/06/15 10:36:07 [core] Subchannel picks a new address "127.0.0.1:50051" to connect
INFO: 2021/06/15 10:36:07 [core] Subchannel Connectivity change to READY
INFO: 2021/06/15 10:36:07 [core] pickfirstBalancer: UpdateSubConnState: 0xc000099fc0, {READY <nil>}
INFO: 2021/06/15 10:36:07 [core] Channel Connectivity change to READY
2021/06/15 10:36:07 RPC Response: 0 message:"Hello unary RPC msg   from server1"
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
export GRPC_XDS_BOOTSTRAP=`pwd`/xds_bootstrap.json
```

edit `src/grpc_client.go`, set to use the xds resolver

```golang
import (
	//_ "google.golang.org/grpc/resolver" // use for "dns:///be.cluster.local:50051"
	_ "google.golang.org/grpc/xds"      // use for xds-experimental:///be-srv
)
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

```log
$ go run src/grpc_client.go --host xds:///be-srv

2021/06/15 10:47:26 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:31 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:36 RPC Response: 2 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:41 RPC Response: 3 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:46 RPC Response: 4 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:51 RPC Response: 5 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:47:56 RPC Response: 6 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:01 RPC Response: 7 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:06 RPC Response: 8 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:11 RPC Response: 9 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:16 RPC Response: 10 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:21 RPC Response: 11 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:48:26 RPC Response: 12 message:"Hello unary RPC msg   from server2"   <<<<<<<<<<<
2021/06/15 10:48:31 RPC Response: 13 message:"Hello unary RPC msg   from server2" 
2021/06/15 10:48:36 RPC Response: 14 message:"Hello unary RPC msg   from server2" 
```

---

If you want more details...

## References

- [Envoy Listener proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto)
- [Envoy Cluster proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto)
- [Envoy Endpoint proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto)


### xDS Server start

```log
$ go run main.go --upstream_port=50051 --upstream_port=50052
INFO[0000] [UpstreamPorts] 50051                        
INFO[0000] [UpstreamPorts] 50052                        
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


///  Backend service ROTATION here 

INFO[0064] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50052 
INFO[0064] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0064] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0064] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0064] >>>>>>>>>>>>>>>>>>> creating snapshot Version 2 
INFO[0064] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.route.v3.RouteConfiguration],  Response[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0064] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0064] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.cluster.v3.Cluster],  Response[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0064] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0064] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0064] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0064] OnStreamResponse... 1   Request [type.googleapis.com/envoy.config.listener.v3.Listener],  Response[type.googleapis.com/envoy.config.listener.v3.Listener] 
INFO[0064] cb.Report()  callbacks                        fetches=0 requests=8
INFO[0064] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.route.v3.RouteConfiguration] 
INFO[0064] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.cluster.v3.Cluster] 
INFO[0064] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment] 
INFO[0064] OnStreamRequest 1  Request[type.googleapis.com/envoy.config.listener.v3.Listener] 
```

### gRPC Client Call 

```log
$ go run src/grpc_client.go --host xds:///be-srv

INFO: 2021/06/15 10:50:43 [core] Channel Created
INFO: 2021/06/15 10:50:43 [core] parsed scheme: "xds"
INFO: 2021/06/15 10:50:43 [xds] [xds-resolver 0xc000118400] Creating resolver for target: {Scheme:xds Authority: Endpoint:be-srv}
INFO: 2021/06/15 10:50:43 [xds] [xds-bootstrap] xds: using bootstrap file with name "/home/srashid/Desktop/grpc_xds/app/xds_bootstrap.json"
INFO: 2021/06/15 10:50:43 [xds] [xds-bootstrap] Bootstrap content: {
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

INFO: 2021/06/15 10:50:43 [xds] [xds-bootstrap] Bootstrap config for creating xds-client: &{BalancerName:xds.domain.com:18000 Creds:0xc0004fc318 TransportAPI:1 NodeProto:id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning" CertProviderConfigs:map[] ServerListenerResourceNameTemplate:}
INFO: 2021/06/15 10:50:43 [core] Channel Created
INFO: 2021/06/15 10:50:43 [core] parsed scheme: ""
INFO: 2021/06/15 10:50:43 [core] scheme "" not registered, fallback to default scheme
INFO: 2021/06/15 10:50:43 [core] ccResolverWrapper: sending update to cc: {[{xds.domain.com:18000  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2021/06/15 10:50:43 [core] Resolver state updated: {Addresses:[{Addr:xds.domain.com:18000 ServerName: Attributes:<nil> Type:0 Metadata:<nil>}] ServiceConfig:<nil> Attributes:<nil>} (resolver returned new addresses)
INFO: 2021/06/15 10:50:43 [core] ClientConn switching balancer to "pick_first"
INFO: 2021/06/15 10:50:43 [core] Channel switches to new LB policy "pick_first"
INFO: 2021/06/15 10:50:43 [core] Subchannel Created
INFO: 2021/06/15 10:50:43 [core] Subchannel(id:4) created
INFO: 2021/06/15 10:50:43 [core] Subchannel Connectivity change to CONNECTING
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Created ClientConn to xDS management server: xds.domain.com:18000
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Created
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] new watch for type ListenerResource, resource name be-srv
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] first watch for type ListenerResource, resource name be-srv, will send a new xDS request
INFO: 2021/06/15 10:50:43 [xds] [xds-resolver 0xc000118400] Watch started on resource name be-srv with xds-client 0x17a3790
INFO: 2021/06/15 10:50:43 [core] Subchannel picks a new address "xds.domain.com:18000" to connect
INFO: 2021/06/15 10:50:43 [core] blockingPicker: the picked transport is not ready, loop back to repick
INFO: 2021/06/15 10:50:43 [core] pickfirstBalancer: UpdateSubConnState: 0xc00022a4f0, {CONNECTING <nil>}
INFO: 2021/06/15 10:50:43 [core] Channel Connectivity change to CONNECTING
INFO: 2021/06/15 10:50:43 [core] Subchannel Connectivity change to READY
INFO: 2021/06/15 10:50:43 [core] pickfirstBalancer: UpdateSubConnState: 0xc00022a4f0, {READY <nil>}
INFO: 2021/06/15 10:50:43 [core] Channel Connectivity change to READY
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS stream created
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv"  type_url:"type.googleapis.com/envoy.config.listener.v3.Listener"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.listener.v3.Listener
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"1"  resources:{[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}}  type_url:"type.googleapis.com/envoy.config.listener.v3.Listener"  nonce:"1"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv, type: *envoy_config_listener_v3.Listener, contains: name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] LDS resource with name be-srv, value {RouteConfigName:be-srv-route InlineRouteConfig:<nil> MaxStreamDuration:0s HTTPFilters:[] InboundListenerCfg:<nil> Raw:[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}} added to cache
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: ListenerResource, version: 1, nonce: 1
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"1"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv"  type_url:"type.googleapis.com/envoy.config.listener.v3.Listener"  response_nonce:"1"
INFO: 2021/06/15 10:50:43 [xds] [xds-resolver 0xc000118400] received LDS update: {RouteConfigName:be-srv-route InlineRouteConfig:<nil> MaxStreamDuration:0s HTTPFilters:[] InboundListenerCfg:<nil> Raw:[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}}, err: <nil>
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] new watch for type RouteConfigResource, resource name be-srv-route
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] first watch for type RouteConfigResource, resource name be-srv-route, will send a new xDS request
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-route"  type_url:"type.googleapis.com/envoy.config.route.v3.RouteConfiguration"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.route.v3.RouteConfiguration
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"1"  resources:{[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}}  type_url:"type.googleapis.com/envoy.config.route.v3.RouteConfiguration"  nonce:"2"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-route, type: *envoy_config_route_v3.RouteConfiguration, contains: name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}.
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] RDS resource with name be-srv-route, value {VirtualHosts:[0xc00032e480] Raw:[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}} added to cache
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: RouteConfigResource, version: 1, nonce: 2
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"1"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-route"  type_url:"type.googleapis.com/envoy.config.route.v3.RouteConfiguration"  response_nonce:"2"
INFO: 2021/06/15 10:50:43 [xds] [xds-resolver 0xc000118400] received RDS update: {VirtualHosts:[0xc00032e480] Raw:[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}}, err: <nil>
INFO: 2021/06/15 10:50:43 [xds] [xds-resolver 0xc000118400] Received update on resource be-srv from xds-client 0x17a3790, generated service config: {"loadBalancingConfig":[{"xds_cluster_manager_experimental":{"children":{"be-srv-cluster":{"childPolicy":[{"cds_experimental":{"cluster":"be-srv-cluster"}}]}}}}]}
INFO: 2021/06/15 10:50:43 [core] ccResolverWrapper: sending update to cc: {[] 0xc00050db40 0xc000010a98}
INFO: 2021/06/15 10:50:43 [core] Resolver state updated: {Addresses:[] ServiceConfig:0xc00050db40 Attributes:0xc000010a98} (service config updated)
INFO: 2021/06/15 10:50:43 [core] ClientConn switching balancer to "xds_cluster_manager_experimental"
INFO: 2021/06/15 10:50:43 [core] Channel switches to new LB policy "xds_cluster_manager_experimental"
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Created
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] update with config &{LoadBalancingConfig:<nil> Children:map[be-srv-cluster:{ChildPolicy:0xc00050d9c0}]}, resolver state {Addresses:[] ServiceConfig:0xc00050db40 Attributes:0xc000010a98}
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] Created
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] xDS credentials in use: false
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Created child policy 0xc0002b6200 of type cds_experimental
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] Received update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] new watch for type ClusterResource, resource name be-srv-cluster
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] first watch for type ClusterResource, resource name be-srv-cluster, will send a new xDS request
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] Watch started on resource name be-srv-cluster with xds-client 0x17a3790
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.cluster.v3.Cluster"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.cluster.v3.Cluster
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"1"  resources:{[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}}}  type_url:"type.googleapis.com/envoy.config.cluster.v3.Cluster"  nonce:"3"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-cluster, type: *envoy_config_cluster_v3.Cluster, contains: name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] CDS resource with name be-srv-cluster, value {ClusterType:0 ServiceName:be-srv-cluster EnableLRS:false SecurityCfg:<nil> MaxRequests:<nil> Raw:[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}} PrioritizedClusterNames:[]} added to cache
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: ClusterResource, version: 1, nonce: 3
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"1"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.cluster.v3.Cluster"  response_nonce:"3"
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] Watch update from xds-client 0x17a3790, content: {ClusterType:0 ServiceName:be-srv-cluster EnableLRS:false SecurityCfg:<nil> MaxRequests:<nil> Raw:[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}} PrioritizedClusterNames:[]}
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Created
INFO: 2021/06/15 10:50:43 [xds] [cds-lb 0xc0002b6200] Created child policy 0xc0005d03c0 of type eds_experimental
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster MaxConcurrentRequests:<nil> LrsLoadReportingServerName:<nil>}
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] new watch for type EndpointsResource, resource name be-srv-cluster
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] first watch for type EndpointsResource, resource name be-srv-cluster, will send a new xDS request
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Watch started on resource name be-srv-cluster with xds-client 0x17a3790
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"1"  resources:{[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50051}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}}  type_url:"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"  nonce:"4"
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-cluster, type: *envoy_config_endpoint_v3.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50051}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] EDS resource with name be-srv-cluster, value {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50051 HealthStatus:1 Weight:0}] ID:{Region:us-central1 Zone:us-central1-a SubZone:} Priority:0 Weight:1000}] Raw:[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50051}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}} added to cache
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: EndpointsResource, version: 1, nonce: 4
INFO: 2021/06/15 10:50:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"1"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"  response_nonce:"4"
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Watch update from xds-client 0x17a3790, content: {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50051 HealthStatus:1 Weight:0}] ID:{Region:us-central1 Zone:us-central1-a SubZone:} Priority:0 Weight:1000}] Raw:[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50051}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}}
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] New priority 0 added
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] New locality {us-central1 us-central1-a } added
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Switching priority from unset to 0
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Created child policy 0xc000172380 of type round_robin
INFO: 2021/06/15 10:50:43 [balancer] base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50051  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2021/06/15 10:50:43 [core] Subchannel Created
INFO: 2021/06/15 10:50:43 [core] Subchannel(id:7) created
INFO: 2021/06/15 10:50:43 [core] Subchannel Connectivity change to CONNECTING
INFO: 2021/06/15 10:50:43 [core] Subchannel picks a new address "be.cluster.local:50051" to connect
INFO: 2021/06/15 10:50:43 [balancer] base.baseBalancer: handle SubConn state change: 0xc000203d50, CONNECTING
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Balancer state update from locality {"region":"us-central1","zone":"us-central1-a"}, new state: {ConnectivityState:CONNECTING Picker:0xc000203c10}
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Child pickers with config: map[{"region":"us-central1","zone":"us-central1-a"}:weight:1000,picker:0xc000508db0,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:CONNECTING Picker:0xc0000930e0}
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Child pickers: map[be-srv-cluster:picker:0xc0000930e0,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2021/06/15 10:50:43 [core] Channel Connectivity change to CONNECTING
INFO: 2021/06/15 10:50:43 [core] Subchannel Connectivity change to READY
INFO: 2021/06/15 10:50:43 [balancer] base.baseBalancer: handle SubConn state change: 0xc000203d50, READY
INFO: 2021/06/15 10:50:43 [roundrobin] roundrobinPicker: newPicker called with info: {map[0xc000203d50:{{be.cluster.local:50051  <nil> 0 <nil>}}]}
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Balancer state update from locality {"region":"us-central1","zone":"us-central1-a"}, new state: {ConnectivityState:READY Picker:0xc0004881e0}
INFO: 2021/06/15 10:50:43 [xds] [eds-lb 0xc0005d03c0] Child pickers with config: map[{"region":"us-central1","zone":"us-central1-a"}:weight:1000,picker:0xc000488210,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc0004823c0}
INFO: 2021/06/15 10:50:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Child pickers: map[be-srv-cluster:picker:0xc0004823c0,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:50:43 [core] Channel Connectivity change to READY
2021/06/15 10:50:43 RPC Response: 0 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:50:48 RPC Response: 1 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:50:53 RPC Response: 2 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:50:58 RPC Response: 3 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:03 RPC Response: 4 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:08 RPC Response: 5 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:13 RPC Response: 6 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:18 RPC Response: 7 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:23 RPC Response: 8 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:28 RPC Response: 9 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:33 RPC Response: 10 message:"Hello unary RPC msg   from server1" 
2021/06/15 10:51:38 RPC Response: 11 message:"Hello unary RPC msg   from server1" 


*****************  ROTATION **********************


INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"2"  resources:{[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50052}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}}  type_url:"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"  nonce:"5"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-cluster, type: *envoy_config_endpoint_v3.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50052}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] EDS resource with name be-srv-cluster, value {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50052 HealthStatus:1 Weight:0}] ID:{Region:us-central1 Zone:us-central1-a SubZone:} Priority:0 Weight:1000}] Raw:[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50052}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}} added to cache
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: EndpointsResource, version: 2, nonce: 5
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.listener.v3.Listener
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Watch update from xds-client 0x17a3790, content: {Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50052 HealthStatus:1 Weight:0}] ID:{Region:us-central1 Zone:us-central1-a SubZone:} Priority:0 Weight:1000}] Raw:[type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment]:{cluster_name:"be-srv-cluster"  endpoints:{locality:{region:"us-central1"  zone:"us-central1-a"}  lb_endpoints:{endpoint:{address:{socket_address:{address:"be.cluster.local"  port_value:50052}}}  health_status:HEALTHY}  load_balancing_weight:{value:1000}}}}
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"2"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"  response_nonce:"5"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"2"  resources:{[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}}  type_url:"type.googleapis.com/envoy.config.listener.v3.Listener"  nonce:"6"
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Locality {us-central1 us-central1-a } updated, weightedChanged: false, addrsChanged: true
INFO: 2021/06/15 10:51:43 [balancer] base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50052  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2021/06/15 10:51:43 [core] Subchannel Created
INFO: 2021/06/15 10:51:43 [core] Subchannel(id:9) created
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv, type: *envoy_config_listener_v3.Listener, contains: name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] LDS resource with name be-srv, value {RouteConfigName:be-srv-route InlineRouteConfig:<nil> MaxStreamDuration:0s HTTPFilters:[] InboundListenerCfg:<nil> Raw:[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}} added to cache
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: ListenerResource, version: 2, nonce: 6
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.cluster.v3.Cluster
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"2"  resources:{[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}}}  type_url:"type.googleapis.com/envoy.config.cluster.v3.Cluster"  nonce:"7"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-cluster, type: *envoy_config_cluster_v3.Cluster, contains: name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] CDS resource with name be-srv-cluster, value {ClusterType:0 ServiceName:be-srv-cluster EnableLRS:false SecurityCfg:<nil> MaxRequests:<nil> Raw:[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}} PrioritizedClusterNames:[]} added to cache
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: ClusterResource, version: 2, nonce: 7
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received, type: type.googleapis.com/envoy.config.route.v3.RouteConfiguration
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS response received: version_info:"2"  resources:{[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}}  type_url:"type.googleapis.com/envoy.config.route.v3.RouteConfiguration"  nonce:"8"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Resource with name: be-srv-route, type: *envoy_config_route_v3.RouteConfiguration, contains: name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}.
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] RDS resource with name be-srv-route, value {VirtualHosts:[0xc00032f040] Raw:[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}} added to cache
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] Sending ACK for response type: RouteConfigResource, version: 2, nonce: 8
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"2"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv"  type_url:"type.googleapis.com/envoy.config.listener.v3.Listener"  response_nonce:"6"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"2"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-cluster"  type_url:"type.googleapis.com/envoy.config.cluster.v3.Cluster"  response_nonce:"7"
INFO: 2021/06/15 10:51:43 [xds] [xds-client 0xc0005c2800] ADS request sent: version_info:"2"  node:{id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1"  metadata:{fields:{key:"R_GCP_PROJECT_NUMBER"  value:{string_value:"123456789012"}}}  locality:{zone:"us-central1-a"}  user_agent_name:"gRPC Go"  user_agent_version:"1.38.0"  client_features:"envoy.lb.does_not_support_overprovisioning"}  resource_names:"be-srv-route"  type_url:"type.googleapis.com/envoy.config.route.v3.RouteConfiguration"  response_nonce:"8"
INFO: 2021/06/15 10:51:43 [core] Subchannel Connectivity change to CONNECTING
INFO: 2021/06/15 10:51:43 [core] Subchannel Connectivity change to SHUTDOWN
INFO: 2021/06/15 10:51:43 [core] Subchannel picks a new address "be.cluster.local:50052" to connect
INFO: 2021/06/15 10:51:43 [transport] transport: loopyWriter.run returning. connection error: desc = "transport is closing"
INFO: 2021/06/15 10:51:43 [xds] [xds-resolver 0xc000118400] received LDS update: {RouteConfigName:be-srv-route InlineRouteConfig:<nil> MaxStreamDuration:0s HTTPFilters:[] InboundListenerCfg:<nil> Raw:[type.googleapis.com/envoy.config.listener.v3.Listener]:{name:"be-srv"  api_listener:{api_listener:{[type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager]:{rds:{config_source:{ads:{}}  route_config_name:"be-srv-route"}}}}}}, err: <nil>
INFO: 2021/06/15 10:51:43 [core] Subchannel Deleted
INFO: 2021/06/15 10:51:43 [core] Subchanel(id:7) deleted
INFO: 2021/06/15 10:51:43 [xds] [cds-lb 0xc0002b6200] Watch update from xds-client 0x17a3790, content: {ClusterType:0 ServiceName:be-srv-cluster EnableLRS:false SecurityCfg:<nil> MaxRequests:<nil> Raw:[type.googleapis.com/envoy.config.cluster.v3.Cluster]:{name:"be-srv-cluster"  type:EDS  eds_cluster_config:{eds_config:{ads:{}}}} PrioritizedClusterNames:[]}
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster MaxConcurrentRequests:<nil> LrsLoadReportingServerName:<nil>}
INFO: 2021/06/15 10:51:43 [balancer] base.baseBalancer: handle SubConn state change: 0xc0005778b0, CONNECTING
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Balancer state update from locality {"region":"us-central1","zone":"us-central1-a"}, new state: {ConnectivityState:READY Picker:0xc0004881e0}
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Child pickers with config: map[{"region":"us-central1","zone":"us-central1-a"}:weight:1000,picker:0xc000489560,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:51:43 [xds] [xds-resolver 0xc000118400] received RDS update: {VirtualHosts:[0xc00032f040] Raw:[type.googleapis.com/envoy.config.route.v3.RouteConfiguration]:{name:"be-srv-route"  virtual_hosts:{name:"be-srv-vs"  domains:"be-srv"  routes:{match:{prefix:""}  route:{cluster:"be-srv-cluster"}}}}}, err: <nil>
INFO: 2021/06/15 10:51:43 [balancer] base.baseBalancer: handle SubConn state change: 0xc000203d50, SHUTDOWN
INFO: 2021/06/15 10:51:43 [xds] [xds-resolver 0xc000118400] Received update on resource be-srv from xds-client 0x17a3790, generated service config: {"loadBalancingConfig":[{"xds_cluster_manager_experimental":{"children":{"be-srv-cluster":{"childPolicy":[{"cds_experimental":{"cluster":"be-srv-cluster"}}]}}}}]}
INFO: 2021/06/15 10:51:43 [roundrobin] roundrobinPicker: newPicker called with info: {map[]}
INFO: 2021/06/15 10:51:43 [core] ccResolverWrapper: sending update to cc: {[] 0xc000494320 0xc000656008}
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Balancer state update from locality {"region":"us-central1","zone":"us-central1-a"}, new state: {ConnectivityState:CONNECTING Picker:0xc000576050}
INFO: 2021/06/15 10:51:43 [core] Resolver state updated: {Addresses:[] ServiceConfig:0xc000494320 Attributes:0xc000656008} ()
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Child pickers with config: map[{"region":"us-central1","zone":"us-central1-a"}:weight:1000,picker:0xc000508270,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] update with config &{LoadBalancingConfig:<nil> Children:map[be-srv-cluster:{ChildPolicy:0xc000494200}]}, resolver state {Addresses:[] ServiceConfig:0xc000494320 Attributes:0xc000656008}
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc000482000}
INFO: 2021/06/15 10:51:43 [xds] [cds-lb 0xc0002b6200] Received update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Child pickers: map[be-srv-cluster:picker:0xc000482000,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:51:43 [xds] [xds-resolver 0xc000118400] Received update on resource be-srv from xds-client 0x17a3790, generated service config: {"loadBalancingConfig":[{"xds_cluster_manager_experimental":{"children":{"be-srv-cluster":{"childPolicy":[{"cds_experimental":{"cluster":"be-srv-cluster"}}]}}}}]}
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:CONNECTING Picker:0xc000482230}
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Child pickers: map[be-srv-cluster:picker:0xc000482230,state:CONNECTING,stateToAggregate:CONNECTING]
INFO: 2021/06/15 10:51:43 [core] Channel Connectivity change to CONNECTING
INFO: 2021/06/15 10:51:43 [core] ccResolverWrapper: sending update to cc: {[] 0xc000494820 0xc000656048}
INFO: 2021/06/15 10:51:43 [core] Resolver state updated: {Addresses:[] ServiceConfig:0xc000494820 Attributes:0xc000656048} ()
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] update with config &{LoadBalancingConfig:<nil> Children:map[be-srv-cluster:{ChildPolicy:0xc000494700}]}, resolver state {Addresses:[] ServiceConfig:0xc000494820 Attributes:0xc000656048}
INFO: 2021/06/15 10:51:43 [xds] [cds-lb 0xc0002b6200] Received update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2021/06/15 10:51:43 [core] Subchannel Connectivity change to READY
INFO: 2021/06/15 10:51:43 [balancer] base.baseBalancer: handle SubConn state change: 0xc0005778b0, READY
INFO: 2021/06/15 10:51:43 [roundrobin] roundrobinPicker: newPicker called with info: {map[0xc0005778b0:{{be.cluster.local:50052  <nil> 0 <nil>}}]}
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Balancer state update from locality {"region":"us-central1","zone":"us-central1-a"}, new state: {ConnectivityState:READY Picker:0xc0002dade0}
INFO: 2021/06/15 10:51:43 [xds] [eds-lb 0xc0005d03c0] Child pickers with config: map[{"region":"us-central1","zone":"us-central1-a"}:weight:1000,picker:0xc0002dae10,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Balancer state update from locality be-srv-cluster, new state: {ConnectivityState:READY Picker:0xc0005325f0}
INFO: 2021/06/15 10:51:43 [xds] [xds-cluster-manager-lb 0xc00050dc00] Child pickers: map[be-srv-cluster:picker:0xc0005325f0,state:READY,stateToAggregate:READY]
INFO: 2021/06/15 10:51:43 [core] Channel Connectivity change to READY
2021/06/15 10:51:43 RPC Response: 12 message:"Hello unary RPC msg   from server2" 
2021/06/15 10:51:48 RPC Response: 13 message:"Hello unary RPC msg   from server2" 
2021/06/15 10:51:53 RPC Response: 14 message:"Hello unary RPC msg   from server2" 
INFO: 2021/06/15 10:51:58 [core] Channel Connectivity change to SHUTDOWN
INFO: 2021/06/15 10:51:58 [xds] [xds-client 0xc0005c2800] watch for type ListenerResource, resource name be-srv canceled
INFO: 2021/06/15 10:51:58 [xds] [xds-client 0xc0005c2800] last watch for type ListenerResource, resource name be-srv canceled, will send a new xDS request
INFO: 2021/06/15 10:51:58 [xds] [xds-client 0xc0005c2800] watch for type RouteConfigResource, resource name be-srv-route canceled
INFO: 2021/06/15 10:51:58 [xds] [xds-client 0xc0005c2800] last watch for type RouteConfigResource, resource name be-srv-route canceled, will send a new xDS request
INFO: 2021/06/15 10:51:58 [xds] [xds-resolver 0xc000118400] Watch cancel on resource name be-srv with xds-client 0x17a3790
INFO: 2021/06/15 10:51:58 [xds] [xds-resolver 0xc000118400] Shutdown
INFO: 2021/06/15 10:51:58 [core] Subchannel Connectivity change to SHUTDOWN
INFO: 2021/06/15 10:51:58 [core] Subchannel Deleted
INFO: 2021/06/15 10:51:58 [core] Subchanel(id:9) deleted
INFO: 2021/06/15 10:51:58 [core] Channel Deleted


```

### xDS debug client

`grpc_client.go` also includes a debug listener that is started as its own go routine on port `:50053`.

- [https://github.com/grpc-ecosystem/grpcdebug#debug-xds](https://github.com/grpc-ecosystem/grpcdebug#debug-xds)

Which means that while the xDS client is running, you can interrogate it for the configuration and other statistics

```
$ grpcdebug localhost:50053 xds status
Name             Status    Version   Type                                                                 LastUpdated
be-srv           ACKED     1         type.googleapis.com/envoy.config.listener.v3.Listener                16 seconds ago   
be-srv-route     ACKED     1         type.googleapis.com/envoy.config.route.v3.RouteConfiguration         16 seconds ago   
be-srv-cluster   ACKED     1         type.googleapis.com/envoy.config.cluster.v3.Cluster                  16 seconds ago   
be-srv-cluster   ACKED     1         type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment   16 seconds ago   
```