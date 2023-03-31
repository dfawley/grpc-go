package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/benchmark/latency"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/keepalive"
)

var port = flag.Int("port", 50052, "port number")
var addr = flag.String("addr", "localhost:50052", "the address to connect to")

var kasp = keepalive.ServerParameters{
	MaxConnectionAge: 3 * time.Second,
}

// server implements EchoServer.
type server struct {
	pb.UnimplementedEchoServer
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var rs = randomString(630000)

func (s *server) UnaryEcho(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Message: rs}, nil
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	address := fmt.Sprintf("localhost:%v", *port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	sLis := latency.Longhaul.Listener(lis)

	s := grpc.NewServer(grpc.KeepaliveParams(kasp))
	pb.RegisterEchoServer(s, &server{})

	go func() {
		if err := s.Serve(sLis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	//const maxWindowSize int32 = (1 << 20) * 16

	if true {
		select {}
	} else {

		conn, err := grpc.Dial(*addr,
			//grpc.WithInitialWindowSize(maxWindowSize),
			//grpc.WithInitialConnWindowSize(maxWindowSize),
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
	}
}
