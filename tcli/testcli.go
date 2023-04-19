package main

import (
	"context"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/benchmark/latency"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/features/proto/echo"
)

var addr = flag.String("addr", "localhost:50052", "the address to connect to")

func main() {
	flag.Parse()

	//const maxWindowSize int32 = (1 << 20) * 16
	//const maxWindowSize2 int32 = 935420

	conn, err := grpc.Dial(*addr,
		//grpc.WithInitialWindowSize(maxWindowSize2),
		//grpc.WithInitialConnWindowSize(maxWindowSize2),
		grpc.WithWriteBufferSize(10*1024*1024),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			dialer := latency.Longhaul.Dialer(net.Dial)
			sConn, err := dialer("tcp4", addr)
			if err != nil {
				return nil, err
			}
			return sConn, nil
		}))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewEchoClient(conn)
	for {
		log.Print("start RPC")
		_, err = c.UnaryEcho(context.Background(), &pb.EchoRequest{Message: "keepalive demo"})
		if err != nil {
			log.Fatalf("unexpected error from UnaryEcho: %v", err)
		}
		log.Print("finish RPC")
	}
	select {}
}
