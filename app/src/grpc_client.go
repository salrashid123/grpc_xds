package main

import (
	"flag"
	"net"

	"github.com/salrashid123/gcegrpc/app/echo"

	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"google.golang.org/grpc/admin"
	_ "google.golang.org/grpc/resolver" // use for "dns:///be.cluster.local:50051"
	_ "google.golang.org/grpc/xds"      // use for xds-experimental:///be-srv
)

const ()

var (
	conn *grpc.ClientConn
)

func main() {

	address := flag.String("host", "dns:///be.cluster.local:50051", "dns:///be.cluster.local:50051 or xds-experimental:///be-srv")
	flag.Parse()

	//address = fmt.Sprintf("xds-experimental:///be-srv")

	// (optional) start background grpc admin services to monitor client
	// "google.golang.org/grpc/admin"
	go func() {
		lis, err := net.Listen("tcp", ":19000")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		defer lis.Close()
		opts := []grpc.ServerOption{grpc.MaxConcurrentStreams(10)}
		grpcServer := grpc.NewServer(opts...)
		cleanup, err := admin.Register(grpcServer)
		if err != nil {
			log.Fatalf("failed to register admin services: %v", err)
		}
		defer cleanup()

		log.Printf("Admin port listen on :%s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	conn, err := grpc.Dial(*address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := echo.NewEchoServerClient(conn)
	ctx := context.Background()

	for i := 0; i < 30; i++ {
		r, err := c.SayHello(ctx, &echo.EchoRequest{Name: "unary RPC msg "})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("RPC Response: %v %v", i, r)
		time.Sleep(2 * time.Second)
	}

}
