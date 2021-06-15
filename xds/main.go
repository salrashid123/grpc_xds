package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"net"

	"sync"
	"sync/atomic"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	ep "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	lv2 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/golang/protobuf/ptypes"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// UpstreamPorts is a type that implements flag.Value interface
type UpstreamPorts []int

// String is a method that implements the flag.Value interface
func (u *UpstreamPorts) String() string {
	// See: https://stackoverflow.com/a/37533144/609290
	return strings.Join(strings.Fields(fmt.Sprint(*u)), ",")
}

// Set is a method that implements the flag.Value interface
func (u *UpstreamPorts) Set(port string) error {
	log.Printf("[UpstreamPorts] %s", port)
	i, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	*u = append(*u, i)
	return nil
}

var (
	debug       bool
	onlyLogging bool

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	config cachev3.SnapshotCache

	upstreamPorts UpstreamPorts
)

const (
	localhost       = "127.0.0.1"
	Ads             = "ads"
	backendHostName = "be.cluster.local"
	listenerName    = "be-srv"
	routeConfigName = "be-srv-route"
	clusterName     = "be-srv-cluster"
	virtualHostName = "be-srv-vs"
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&gatewayPort, "gateway", 18001, "Management server port for HTTP gateway")
	flag.StringVar(&mode, "ads", Ads, "Management server type (ads, xds, rest)")
	// Converts repeated flags (e.g. `--upstream_port=50051 --upstream_port=50052`) into a []int
	flag.Var(&upstreamPorts, "upstream_port", "list of upstream gRPC servers")
}

type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}
func (logger logger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}
func (cb *callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.WithFields(log.Fields{"fetches": cb.fetches, "requests": cb.requests}).Info("cb.Report()  callbacks")
}
func (cb *callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	log.Infof("OnStreamOpen %d open for Type [%s]", id, typ)
	return nil
}
func (cb *callbacks) OnStreamClosed(id int64) {
	log.Infof("OnStreamClosed %d closed", id)
}
func (cb *callbacks) OnStreamRequest(id int64, r *discovery.DiscoveryRequest) error {
	log.Infof("OnStreamRequest %d  Request[%v]", id, r.TypeUrl)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.requests++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnStreamResponse(id int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	log.Infof("OnStreamResponse... %d   Request [%v],  Response[%v]", id, req.TypeUrl, resp.TypeUrl)
	cb.Report()
}
func (cb *callbacks) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
	log.Infof("OnFetchRequest... Request [%v]", req.TypeUrl)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fetches++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	log.Infof("OnFetchResponse... Resquest[%v],  Response[%v]", req.TypeUrl, resp.TypeUrl)
}

func (cb *callbacks) OnDeltaStreamClosed(id int64) {
	log.Infof("OnDeltaStreamClosed... %v", id)
}

func (cb *callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	log.Infof("OnDeltaStreamOpen... %v  of type %s", id, typ)
	return nil
}

func (c *callbacks) OnStreamDeltaRequest(i int64, request *discovery.DeltaDiscoveryRequest) error {
	log.Infof("OnStreamDeltaRequest... %v  of type %s", i, request)
	return nil
}

func (c *callbacks) OnStreamDeltaResponse(i int64, request *discovery.DeltaDiscoveryRequest, response *discovery.DeltaDiscoveryResponse) {
	log.Infof("OnStreamDeltaResponse... %v  of type %s", i, request)
}

type callbacks struct {
	signal   chan struct{}
	fetches  int
	requests int
	mu       sync.Mutex
}

// Hasher returns node ID as an ID
type Hasher struct {
}

// ID function
func (h Hasher) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

const grpcMaxConcurrentStreams = 1000

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, server xds.Server, port uint) {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	// register services
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)

	log.WithFields(log.Fields{"port": port}).Info("management server listening")
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// 11/11/20 TODO:  optionally set this up
// // RunManagementGateway starts an HTTP gateway to an xDS server.
// func RunManagementGateway(ctx context.Context, srv xds.Server, port uint) {
// 	log.WithFields(log.Fields{"port": port}).Info("gateway listening HTTP/1.1")

// 	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: &xds.HTTPGateway{Server: srv}}
// 	go func() {
// 		if err := server.ListenAndServe(); err != nil {
// 			log.Error(err)
// 		}
// 	}()
// }

func main() {
	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	ctx := context.Background()

	log.Printf("Starting control plane")

	signal := make(chan struct{})
	cb := &callbacks{
		signal:   signal,
		fetches:  0,
		requests: 0,
	}
	config = cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil)

	srv := xds.NewServer(ctx, config, cb)

	go RunManagementServer(ctx, srv, port)
	//go RunManagementGateway(ctx, srv, gatewayPort)

	<-signal

	cb.Report()

	nodeId := config.GetStatusKeys()[0]
	log.Infof(">>>>>>>>>>>>>>>>>>> creating NodeID %s", nodeId)

	for _, v := range upstreamPorts {

		// ENDPOINT
		log.Infof(">>>>>>>>>>>>>>>>>>> creating ENDPOINT for remoteHost:port %s:%d", backendHostName, v)
		hst := &core.Address{Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Address:  backendHostName,
				Protocol: core.SocketAddress_TCP,
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: uint32(v),
				},
			},
		}}

		//eds := []cache.Resource{
		eds := []types.Resource{
			&endpoint.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints: []*ep.LocalityLbEndpoints{{
					Locality: &core.Locality{
						Region: "us-central1",
						Zone:   "us-central1-a",
					},
					Priority:            0,
					LoadBalancingWeight: &wrapperspb.UInt32Value{Value: uint32(1000)},
					LbEndpoints: []*ep.LbEndpoint{
						{
							HostIdentifier: &ep.LbEndpoint_Endpoint{
								Endpoint: &ep.Endpoint{
									Address: hst,
								}},
							HealthStatus: core.HealthStatus_HEALTHY,
						},
					},
				}},
			},
		}

		// CLUSTER
		log.Infof(">>>>>>>>>>>>>>>>>>> creating CLUSTER " + clusterName)
		cls := []types.Resource{
			&cluster.Cluster{
				Name:                 clusterName,
				LbPolicy:             cluster.Cluster_ROUND_ROBIN,
				ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
				EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
					EdsConfig: &core.ConfigSource{
						ConfigSourceSpecifier: &core.ConfigSource_Ads{},
					},
				},
			},
		}

		// RDS
		log.Infof(">>>>>>>>>>>>>>>>>>> creating RDS " + virtualHostName)
		vh := &route.VirtualHost{
			Name:    virtualHostName,
			Domains: []string{listenerName}, //******************* >> must match what is specified at xds:/// //

			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterName,
						},
					},
				},
			}}}

		rds := []types.Resource{
			&route.RouteConfiguration{
				Name:         routeConfigName,
				VirtualHosts: []*route.VirtualHost{vh},
			},
		}

		// LISTENER
		log.Infof(">>>>>>>>>>>>>>>>>>> creating LISTENER " + listenerName)
		hcRds := &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: routeConfigName,
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
		}

		manager := &hcm.HttpConnectionManager{
			CodecType:      hcm.HttpConnectionManager_AUTO,
			RouteSpecifier: hcRds,
		}

		pbst, err := ptypes.MarshalAny(manager)
		if err != nil {
			panic(err)
		}

		l := []types.Resource{
			&listener.Listener{
				Name: listenerName,
				ApiListener: &lv2.ApiListener{
					ApiListener: pbst,
				},
			}}

		rt := []types.Resource{}
		sec := []types.Resource{}

		// =================================================================================
		atomic.AddInt32(&version, 1)
		log.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(version))

		snap := cachev3.NewSnapshot(fmt.Sprint(version), eds, cls, rds, l, rt, sec)

		config.SetSnapshot(nodeId, snap)

		time.Sleep(60 * time.Second)

	}

}
