package main

import (
	"echo"
	"flag"

	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	//	"google.golang.org/grpc/resolver"
	_ "google.golang.org/grpc/xds"
)

const ()

var (
	conn *grpc.ClientConn
)

func main() {

	address := flag.String("host", "dns:///be1.cluster.local:50051", "dns:///be1.cluster.local:50051 or xds-experimental:///be-srv")
	flag.Parse()

	//*address = fmt.Sprintf("dns:///be-srv-lb.default.svc.cluster.local:50051")

	//address = fmt.Sprintf("xds-experimental:///be-srv")

	conn, err := grpc.Dial(*address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := echo.NewEchoServerClient(conn)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		r, err := c.SayHello(ctx, &echo.EchoRequest{Name: "unary RPC msg "})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("RPC Response: %v %v", i, r)
		time.Sleep(5 * time.Second)
	}

}
