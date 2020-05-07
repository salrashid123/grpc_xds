## gRPC xDS Loadbalancing

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

The xDS server will rotate the valid endpoint targets it has (which is really one; a trivial example, truly).  The second target is the second port where the second gRPC endpoint is running.

The next time you run the gRPC client, it will automatically received endpoint instructions from the xDS server and connect to the second gRPC server.


thats it.


---

### Setup

edit `/etc/hosts`

```bash
127.0.0.1 be.cluster.local xds.domain.com
```

### gRPC Server

Start gRPC Servers

```bash
cd app/
go run src/grpc_server.go --grpcport :50051
go run src/grpc_server.go --grpcport :50052
```

### gRPC Client (DNS)

Enable debug tracing on the client; the whole intent is to see all the details!

```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

Use DNS as the bootstrap mechanism to connec to the server:

```bash
$ go run src/grpc_client.go --host dns:///be.cluster.local:50051

INFO: 2020/04/21 15:25:11 parsed scheme: "dns"
INFO: 2020/04/21 15:25:11 ccResolverWrapper: sending update to cc: {[{127.0.0.1:50051  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2020/04/21 15:25:11 ClientConn switching balancer to "pick_first"
INFO: 2020/04/21 15:25:11 Channel switches to new LB policy "pick_first"
INFO: 2020/04/21 15:25:11 Subchannel Connectivity change to CONNECTING
INFO: 2020/04/21 15:25:11 blockingPicker: the picked transport is not ready, loop back to repick
INFO: 2020/04/21 15:25:11 Subchannel picks a new address "127.0.0.1:50051" to connect
INFO: 2020/04/21 15:25:11 pickfirstBalancer: HandleSubConnStateChange: 0xc00015bfa0, {CONNECTING <nil>}
INFO: 2020/04/21 15:25:11 Channel Connectivity change to CONNECTING
INFO: 2020/04/21 15:25:11 Subchannel Connectivity change to READY
INFO: 2020/04/21 15:25:11 pickfirstBalancer: HandleSubConnStateChange: 0xc00015bfa0, {READY <nil>}
INFO: 2020/04/21 15:25:11 Channel Connectivity change to READY

2020/04/21 15:25:12 RPC Response: 0 message:"Hello unary RPC msg   from hostname srashid1" 

INFO: 2020/04/21 15:25:12 Channel Connectivity change to SHUTDOWN
INFO: 2020/04/21 15:25:12 Subchannel Connectivity change to SHUTDOWN
```

### xDS Server

Now start the xDS server:

```bash
cd xds
go run main.go

INFO[0000] Starting control plane                       
INFO[0000] gateway listening HTTP/1.1                    port=18001
INFO[0000] management server listening                   port=18000
```


### gRPC Client (xDS)

Edit the xds Bootstrap file and specify the `server_url`

The grpc client will connect to this as the the xDS server.  The gRPC client library looks for a specific env-var (`GRPC_XDS_BOOTSTRAP`) that points to the file 

- `xds_bootstrap.json`:
```json
{
  "xds_servers": [
    {
      "server_uri": "xds.domain.com:18000"
    }
  ],
  "node": {
    "id": "b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1",
    "metadata": {
      "R_PROJECT_NUMBER": "123456789012"
    },
    "locality": {
      "zone": "us-central1-a"
    }
  }
}
```

```bash
export GRPC_XDS_BOOTSTRAP=`pwd`/xds_bootstrap.json
```

* Run the client

```
go run src/grpc_client.go --host xds-experimental:///be-srv
```

in the debug logs that it connected to port `:50051`
```
INFO: 2020/04/21 16:14:42 Subchannel picks a new address "be.cluster.local:50051" to connect
```


Then wait one minute and rerun the client:

```
INFO: 2020/04/21 16:16:08 Subchannel picks a new address "be.cluster.local:50052" to connect
```

The port it connected to is `:50052`


right, thats it

---


if you want more details...

### References

- [Envoy Listener proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto)
- [Envoy Cluster proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto)
- [Envoy Endpoint proto](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto)


#### xDS Server start

```bash
INFO[0000] Starting control plane                       
INFO[0000] gateway listening HTTP/1.1                    port=18001
INFO[0000] management server listening                   port=18000
INFO[0010] OnStreamOpen 1 open for Type []              
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0010] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0010] >>>>>>>>>>>>>>>>>>> creating NodeID b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1 
INFO[0010] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50051 
INFO[0010] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0010] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0010] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0010] >>>>>>>>>>>>>>>>>>> creating snapshot Version 1 
INFO[0010] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Listener],  Response[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0010] cb.Report()  callbacks                        fetches=0 requests=1
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Listener] 
INFO[0010] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.RouteConfiguration],  Response[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0010] cb.Report()  callbacks                        fetches=0 requests=3
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.RouteConfiguration] 
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0010] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.Cluster],  Response[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0010] cb.Report()  callbacks                        fetches=0 requests=5
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.Cluster] 
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0010] OnStreamResponse... 1   Request [type.googleapis.com/envoy.api.v2.ClusterLoadAssignment],  Response[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0010] cb.Report()  callbacks                        fetches=0 requests=7
INFO[0010] OnStreamRequest 1  Request[type.googleapis.com/envoy.api.v2.ClusterLoadAssignment] 
INFO[0011] OnStreamClosed 1 closed
```


#### gRPC Client Call #1

```
$ go run src/grpc_client.go --host xds-experimental:///be-srv

INFO: 2020/04/21 16:14:42 parsed scheme: "xds-experimental"
INFO: 2020/04/21 16:14:42 [xds-bootstrap] Got bootstrap file location from GRPC_XDS_BOOTSTRAP environment variable: /home/srashid/Desktop/xds_grpc/app/xds_bootstrap.json
INFO: 2020/04/21 16:14:42 [xds-bootstrap] Bootstrap content: {
  "xds_servers": [
    {
      "server_uri": "xds.domain.com:18000"
    }
  ],---

    "locality": {
      "zone": "us-central1-a"
    }
  }
}
INFO: 2020/04/21 16:14:42 [xds-bootstrap] Bootstrap config for creating xds-client: &{BalancerName:xds.domain.com:18000 Creds:<nil> NodeProto:id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" }
INFO: 2020/04/21 16:14:42 [xds-resolver 0xc000203900] Creating resolver for target: {Scheme:xds-experimental Authority: Endpoint:be-srv}
WARNING: 2020/04/21 16:14:42 [xds-resolver 0xc000203900] No credentials available, using Insecure
INFO: 2020/04/21 16:14:42 parsed scheme: ""
INFO: 2020/04/21 16:14:42 scheme "" not registered, fallback to default scheme
INFO: 2020/04/21 16:14:42 ccResolverWrapper: sending update to cc: {[{xds.domain.com:18000  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2020/04/21 16:14:42 ClientConn switching balancer to "pick_first"
INFO: 2020/04/21 16:14:42 Channel switches to new LB policy "pick_first"
INFO: 2020/04/21 16:14:42 Subchannel Connectivity change to CONNECTING
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Created ClientConn to xDS server: xds.domain.com:18000
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Created
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.Listener, resource names: [be-srv]
INFO: 2020/04/21 16:14:42 [xds-resolver 0xc000203900] Watch started on resource name be-srv with xds-client 0xc00011a870
INFO: 2020/04/21 16:14:42 Subchannel picks a new address "xds.domain.com:18000" to connect
INFO: 2020/04/21 16:14:42 pickfirstBalancer: HandleSubConnStateChange: 0xc00024e0f0, {CONNECTING <nil>}
INFO: 2020/04/21 16:14:42 Channel Connectivity change to CONNECTING
INFO: 2020/04/21 16:14:42 Subchannel Connectivity change to READY
INFO: 2020/04/21 16:14:42 pickfirstBalancer: HandleSubConnStateChange: 0xc00024e0f0, {READY <nil>}
INFO: 2020/04/21 16:14:42 Channel Connectivity change to READY
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS stream created
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received, type: type.googleapis.com/envoy.api.v2.Listener
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.Listener" value:"\n\006be-srv\232\001z\nx\n`type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager\022\024\032\022\n\002\032\000\022\014be-srv-route" > type_url:"type.googleapis.com/envoy.api.v2.Listener" nonce:"1" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name: be-srv, type: *envoy_api_v2.Listener, contains: name:"be-srv" api_listener:<api_listener:<type_url:"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager" value:"\032\022\n\002\032\000\022\014be-srv-route" > > 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with type *envoy_config_filter_network_http_connection_manager_v2.HttpConnectionManager, contains rds:<config_source:<ads:<> > route_config_name:"be-srv-route" > 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] xds: client received LDS update: {routeName:be-srv-route}, err: <nil>
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.RouteConfiguration, resource names: [be-srv-route]
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.Listener, version: 1, nonce: 1
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" response_nonce:"1" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received, type: type.googleapis.com/envoy.api.v2.RouteConfiguration
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" value:"\n\014be-srv-route\022+\n\tbe-srv-vs\022\006be-srv\032\026\n\002\n\000\022\020\n\016be-srv-cluster" > type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" nonce:"2" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name: be-srv-route, type: *envoy_api_v2.RouteConfiguration, contains: name:"be-srv-route" virtual_hosts:<name:"be-srv-vs" domains:"be-srv" routes:<match:<prefix:"" > route:<cluster:"be-srv-cluster" > > > 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name be-srv-route, type string, value be-srv-cluster added to cache
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] xds: client received RDS update: {clusterName:be-srv-cluster}, err: <nil>
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.RouteConfiguration, version: 1, nonce: 2
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" response_nonce:"2" 
INFO: 2020/04/21 16:14:42 [xds-resolver 0xc000203900] Received update on resource be-srv from xds-client 0xc00011a870, generated service config: {
    "loadBalancingConfig":[
      {
        "cds_experimental":{
          "Cluster": "be-srv-cluster"
        }
      }
    ]
  }
INFO: 2020/04/21 16:14:42 ccResolverWrapper: sending update to cc: {[] 0xc000256600 0xc0002e6050}
INFO: 2020/04/21 16:14:42 ClientConn switching balancer to "cds_experimental"
INFO: 2020/04/21 16:14:42 Channel switches to new LB policy "cds_experimental"
INFO: 2020/04/21 16:14:42 [cds-lb 0xc000258750] Created
INFO: 2020/04/21 16:14:42 [cds-lb 0xc000258750] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.Cluster, resource names: [be-srv-cluster]
INFO: 2020/04/21 16:14:42 [cds-lb 0xc000258750] Watch started on resource name be-srv-cluster with xds-client 0xc00011a870
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received, type: type.googleapis.com/envoy.api.v2.Cluster
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.Cluster" value:"\n\016be-srv-cluster\032\004\n\002\032\000\020\003" > type_url:"type.googleapis.com/envoy.api.v2.Cluster" nonce:"3" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name: be-srv-cluster, type: *envoy_api_v2.Cluster, contains: name:"be-srv-cluster" type:EDS eds_cluster_config:<eds_config:<ads:<> > > 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name be-srv-cluster, type client.CDSUpdate, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.Cluster, version: 1, nonce: 3
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" response_nonce:"3" 
INFO: 2020/04/21 16:14:42 [cds-lb 0xc000258750] Watch update from xds-client 0xc00011a870, content: {ServiceName:be-srv-cluster EnableLRS:false}
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Created
INFO: 2020/04/21 16:14:42 [cds-lb 0xc000258750] Created child policy 0xc000258820 of type eds_experimental
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> BalancerName: ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster LrsLoadReportingServerName:<nil>}
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment, resource names: [be-srv-cluster]
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Watch started on resource name be-srv-cluster with xds-client 0xc00011a870
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received, type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS response received: version_info:"1" resources:<type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" value:"\n\016be-srv-cluster\022C\n\034\n\013us-central1\022\rus-central1-a\022\036\020\001\n\032\n\030\n\026\022\020be.cluster.local\030\203\207\003\032\003\010\350\007" > type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" nonce:"4" 
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Resource with name: be-srv-cluster, type: *envoy_api_v2.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster" endpoints:<locality:<region:"us-central1" zone:"us-central1-a" > lb_endpoints:<endpoint:<address:<socket_address:<address:"be.cluster.local" port_value:50051 > > > health_status:HEALTHY > load_balancing_weight:<value:1000 > > 
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Watch update from xds-client 0xc00011a870, content: &{Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50051 HealthStatus:1 Weight:0}] ID:us-central1-us-central1-a- Priority:0 Weight:1000}]}
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment, version: 1, nonce: 4
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] New priority 0 added
INFO: 2020/04/21 16:14:42 [xds-client 0xc00011a870] ADS request sent: version_info:"1" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" response_nonce:"4" 
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] New locality us-central1-us-central1-a- added
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Switching priority from unset to 0
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Created child policy 0xc00042a0a0 of type round_robin
INFO: 2020/04/21 16:14:42 base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50051  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2020/04/21 16:14:42 Subchannel Connectivity change to CONNECTING
INFO: 2020/04/21 16:14:42 Subchannel picks a new address "be.cluster.local:50051" to connect
INFO: 2020/04/21 16:14:42 base.baseBalancer: handle SubConn state change: 0xc000266740, CONNECTING
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Balancer state update from locality us-central1-us-central1-a-, new state: {ConnectivityState:CONNECTING Picker:0xc0002666f0}
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Child pickers with weight: map[us-central1-us-central1-a-:weight:1000,picker:0xc0002556d0,state:CONNECTING]
INFO: 2020/04/21 16:14:42 Channel Connectivity change to CONNECTING
INFO: 2020/04/21 16:14:42 Subchannel Connectivity change to READY
INFO: 2020/04/21 16:14:42 base.baseBalancer: handle SubConn state change: 0xc000266740, READY
INFO: 2020/04/21 16:14:42 roundrobinPicker: newPicker called with info: {map[0xc000266740:{{be.cluster.local:50051  <nil> 0 <nil>}}]}
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Balancer state update from locality us-central1-us-central1-a-, new state: {ConnectivityState:READY Picker:0xc00031c4b0}
INFO: 2020/04/21 16:14:42 [eds-lb 0xc000258820] Child pickers with weight: map[us-central1-us-central1-a-:weight:1000,picker:0xc0002b21e0,state:READY]
INFO: 2020/04/21 16:14:42 Channel Connectivity change to READY
2020/04/21 16:14:43 RPC Response: 0 message:"Hello unary RPC msg   from hostname srashid1" 
INFO: 2020/04/21 16:14:43 Channel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:14:43 [xds-resolver 0xc000203900] Watch cancel on resource name be-srv with xds-client 0xc00011a870
INFO: 2020/04/21 16:14:43 Channel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:14:43 Subchannel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:14:43 [xds-client 0xc00011a870] Shutdown
INFO: 2020/04/21 16:14:43 [xds-resolver 0xc000203900] Shutdown
WARNING: 2020/04/21 16:14:43 [xds-client 0xc00011a870] ADS stream is closed with error: rpc error: code = Canceled desc = context canceled
INFO: 2020/04/21 16:14:43 transport: loopyWriter.run returning. connection error: desc = "transport is closing"
INFO: 2020/04/21 16:14:43 [cds-lb 0xc000258750] Watch cancelled on resource name be-srv-cluster with xds-client 0xc00011a870
INFO: 2020/04/21 16:14:43 Subchannel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:14:43 [eds-lb 0xc000258820] Watch cancelled on resource name be-srv-cluster with xds-client 0xc00011a870
```

#### xDS Server rotate endpoint

```bash
INFO[0070] >>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port be.cluster.local:50052 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating CLUSTER be-srv-cluster 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating RDS be-srv-vs   
INFO[0070] >>>>>>>>>>>>>>>>>>> creating LISTENER be-srv 
INFO[0070] >>>>>>>>>>>>>>>>>>> creating snapshot Version 2 
```

#### gRPC Server Call #2
```
$ go run src/grpc_client.go --host xds-experimental:///be-srv

INFO: 2020/04/21 16:16:08 parsed scheme: "xds-experimental"
INFO: 2020/04/21 16:16:08 [xds-bootstrap] Got bootstrap file location from GRPC_XDS_BOOTSTRAP environment variable: /home/srashid/Desktop/xds_grpc/app/xds_bootstrap.json
INFO: 2020/04/21 16:16:08 [xds-bootstrap] Bootstrap content: {
  "xds_servers": [
    {
      "server_uri": "xds.domain.com:18000"
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
INFO: 2020/04/21 16:16:08 [xds-bootstrap] Bootstrap config for creating xds-client: &{BalancerName:xds.domain.com:18000 Creds:<nil> NodeProto:id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" }
INFO: 2020/04/21 16:16:08 [xds-resolver 0xc00022b900] Creating resolver for target: {Scheme:xds-experimental Authority: Endpoint:be-srv}
WARNING: 2020/04/21 16:16:08 [xds-resolver 0xc00022b900] No credentials available, using Insecure
INFO: 2020/04/21 16:16:08 parsed scheme: ""
INFO: 2020/04/21 16:16:08 scheme "" not registered, fallback to default scheme
INFO: 2020/04/21 16:16:08 ccResolverWrapper: sending update to cc: {[{xds.domain.com:18000  <nil> 0 <nil>}] <nil> <nil>}
INFO: 2020/04/21 16:16:08 ClientConn switching balancer to "pick_first"
INFO: 2020/04/21 16:16:08 Channel switches to new LB policy "pick_first"
INFO: 2020/04/21 16:16:08 Subchannel Connectivity change to CONNECTING
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Created ClientConn to xDS server: xds.domain.com:18000
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Created
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.Listener, resource names: [be-srv]
INFO: 2020/04/21 16:16:08 [xds-resolver 0xc00022b900] Watch started on resource name be-srv with xds-client 0xc000136870
INFO: 2020/04/21 16:16:08 Subchannel picks a new address "xds.domain.com:18000" to connect
INFO: 2020/04/21 16:16:08 pickfirstBalancer: HandleSubConnStateChange: 0xc0002780f0, {CONNECTING <nil>}
INFO: 2020/04/21 16:16:08 Channel Connectivity change to CONNECTING
INFO: 2020/04/21 16:16:08 Subchannel Connectivity change to READY
INFO: 2020/04/21 16:16:08 pickfirstBalancer: HandleSubConnStateChange: 0xc0002780f0, {READY <nil>}
INFO: 2020/04/21 16:16:08 Channel Connectivity change to READY
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS stream created
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received, type: type.googleapis.com/envoy.api.v2.Listener
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.Listener" value:"\n\006be-srv\232\001z\nx\n`type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager\022\024\032\022\n\002\032\000\022\014be-srv-route" > type_url:"type.googleapis.com/envoy.api.v2.Listener" nonce:"1" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name: be-srv, type: *envoy_api_v2.Listener, contains: name:"be-srv" api_listener:<api_listener:<type_url:"type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager" value:"\032\022\n\002\032\000\022\014be-srv-route" > > 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with type *envoy_config_filter_network_http_connection_manager_v2.HttpConnectionManager, contains rds:<config_source:<ads:<> > route_config_name:"be-srv-route" > 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] xds: client received LDS update: {routeName:be-srv-route}, err: <nil>
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.RouteConfiguration, resource names: [be-srv-route]
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.Listener, version: 2, nonce: 1
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv" type_url:"type.googleapis.com/envoy.api.v2.Listener" response_nonce:"1" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received, type: type.googleapis.com/envoy.api.v2.RouteConfiguration
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" value:"\n\014be-srv-route\022+\n\tbe-srv-vs\022\006be-srv\032\026\n\002\n\000\022\020\n\016be-srv-cluster" > type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" nonce:"2" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name: be-srv-route, type: *envoy_api_v2.RouteConfiguration, contains: name:"be-srv-route" virtual_hosts:<name:"be-srv-vs" domains:"be-srv" routes:<match:<prefix:"" > route:<cluster:"be-srv-cluster" > > > 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name be-srv-route, type string, value be-srv-cluster added to cache
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] xds: client received RDS update: {clusterName:be-srv-cluster}, err: <nil>
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.RouteConfiguration, version: 2, nonce: 2
INFO: 2020/04/21 16:16:08 [xds-resolver 0xc00022b900] Received update on resource be-srv from xds-client 0xc000136870, generated service config: {
    "loadBalancingConfig":[
      {
        "cds_experimental":{
          "Cluster": "be-srv-cluster"
        }
      }
    ]
  }
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-route" type_url:"type.googleapis.com/envoy.api.v2.RouteConfiguration" response_nonce:"2" 
INFO: 2020/04/21 16:16:08 ccResolverWrapper: sending update to cc: {[] 0xc0003d2160 0xc0003de018}
INFO: 2020/04/21 16:16:08 ClientConn switching balancer to "cds_experimental"
INFO: 2020/04/21 16:16:08 Channel switches to new LB policy "cds_experimental"
INFO: 2020/04/21 16:16:08 [cds-lb 0xc0003e00d0] Created
INFO: 2020/04/21 16:16:08 [cds-lb 0xc0003e00d0] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> ClusterName:be-srv-cluster}
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.Cluster, resource names: [be-srv-cluster]
INFO: 2020/04/21 16:16:08 [cds-lb 0xc0003e00d0] Watch started on resource name be-srv-cluster with xds-client 0xc000136870
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received, type: type.googleapis.com/envoy.api.v2.Cluster
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.Cluster" value:"\n\016be-srv-cluster\032\004\n\002\032\000\020\003" > type_url:"type.googleapis.com/envoy.api.v2.Cluster" nonce:"3" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name: be-srv-cluster, type: *envoy_api_v2.Cluster, contains: name:"be-srv-cluster" type:EDS eds_cluster_config:<eds_config:<ads:<> > > 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name be-srv-cluster, type client.CDSUpdate, value {ServiceName:be-srv-cluster EnableLRS:false} added to cache
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.Cluster, version: 2, nonce: 3
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.Cluster" response_nonce:"3" 
INFO: 2020/04/21 16:16:08 [cds-lb 0xc0003e00d0] Watch update from xds-client 0xc000136870, content: {ServiceName:be-srv-cluster EnableLRS:false}
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Created
INFO: 2020/04/21 16:16:08 [cds-lb 0xc0003e00d0] Created child policy 0xc000143ee0 of type eds_experimental
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Receive update from resolver, balancer config: &{LoadBalancingConfig:<nil> BalancerName: ChildPolicy:<nil> FallBackPolicy:<nil> EDSServiceName:be-srv-cluster LrsLoadReportingServerName:<nil>}
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ADS request for new watch of type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment, resource names: [be-srv-cluster]
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Watch started on resource name be-srv-cluster with xds-client 0xc000136870
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received, type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS response received: version_info:"2" resources:<type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" value:"\n\016be-srv-cluster\022C\n\034\n\013us-central1\022\rus-central1-a\022\036\020\001\n\032\n\030\n\026\022\020be.cluster.local\030\204\207\003\032\003\010\350\007" > type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" nonce:"4" 
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Resource with name: be-srv-cluster, type: *envoy_api_v2.ClusterLoadAssignment, contains: cluster_name:"be-srv-cluster" endpoints:<locality:<region:"us-central1" zone:"us-central1-a" > lb_endpoints:<endpoint:<address:<socket_address:<address:"be.cluster.local" port_value:50052 > > > health_status:HEALTHY > load_balancing_weight:<value:1000 > > 
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Watch update from xds-client 0xc000136870, content: &{Drops:[] Localities:[{Endpoints:[{Address:be.cluster.local:50052 HealthStatus:1 Weight:0}] ID:us-central1-us-central1-a- Priority:0 Weight:1000}]}
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] Sending ACK for response type: type.googleapis.com/envoy.api.v2.ClusterLoadAssignment, version: 2, nonce: 4
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] New priority 0 added
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] New locality us-central1-us-central1-a- added
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Switching priority from unset to 0
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Created child policy 0xc000248960 of type round_robin
INFO: 2020/04/21 16:16:08 base.baseBalancer: got new ClientConn state:  {{[{be.cluster.local:50052  <nil> 0 <nil>}] <nil> <nil>} <nil>}
INFO: 2020/04/21 16:16:08 [xds-client 0xc000136870] ADS request sent: version_info:"2" node:<id:"b7f9c818-fb46-43ca-8662-d3bdbcf7ec18~10.0.0.1" metadata:<fields:<key:"R_GCP_PROJECT_NUMBER" value:<string_value:"123456789012" > > > locality:<zone:"us-central1-a" > build_version:"gRPC Go 1.28.1" user_agent_name:"gRPC Go" user_agent_version:"1.28.1" client_features:"envoy.lb.does_not_support_overprovisioning" > resource_names:"be-srv-cluster" type_url:"type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" response_nonce:"4" 
INFO: 2020/04/21 16:16:08 Subchannel Connectivity change to CONNECTING
INFO: 2020/04/21 16:16:08 Subchannel picks a new address "be.cluster.local:50052" to connect
INFO: 2020/04/21 16:16:08 base.baseBalancer: handle SubConn state change: 0xc000278ad0, CONNECTING
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Balancer state update from locality us-central1-us-central1-a-, new state: {ConnectivityState:CONNECTING Picker:0xc000278a80}
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Child pickers with weight: map[us-central1-us-central1-a-:weight:1000,picker:0xc0003d9e00,state:CONNECTING]
INFO: 2020/04/21 16:16:08 Channel Connectivity change to CONNECTING
INFO: 2020/04/21 16:16:08 Subchannel Connectivity change to READY
INFO: 2020/04/21 16:16:08 base.baseBalancer: handle SubConn state change: 0xc000278ad0, READY
INFO: 2020/04/21 16:16:08 roundrobinPicker: newPicker called with info: {map[0xc000278ad0:{{be.cluster.local:50052  <nil> 0 <nil>}}]}
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Balancer state update from locality us-central1-us-central1-a-, new state: {ConnectivityState:READY Picker:0xc00027b5f0}
INFO: 2020/04/21 16:16:08 [eds-lb 0xc000143ee0] Child pickers with weight: map[us-central1-us-central1-a-:weight:1000,picker:0xc00027f220,state:READY]
INFO: 2020/04/21 16:16:08 Channel Connectivity change to READY
2020/04/21 16:16:09 RPC Response: 0 message:"Hello unary RPC msg   from hostname srashid1" 
INFO: 2020/04/21 16:16:09 Channel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:16:09 [xds-resolver 0xc00022b900] Watch cancel on resource name be-srv with xds-client 0xc000136870
INFO: 2020/04/21 16:16:09 Channel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:16:09 Subchannel Connectivity change to SHUTDOWN
INFO: 2020/04/21 16:16:09 [xds-client 0xc000136870] Shutdown
INFO: 2020/04/21 16:16:09 [xds-resolver 0xc00022b900] Shutdown
WARNING: 2020/04/21 16:16:09 [xds-client 0xc000136870] ADS stream is closed with error: rpc error: code = Canceled desc = context canceled
INFO: 2020/04/21 16:16:09 Subchannel Connectivity change to SHUTDOWN
```

