package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher"
	"github.com/perlin-network/noise/handshake"
	"github.com/perlin-network/noise/skademlia"
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
	"net"
	"os"
	"strconv"
	"time"
)

type chatHandler struct{}

func (chatHandler) Stream(stream Chat_StreamServer) error {
	for {
		txt, err := stream.Recv()

		if err != nil {
			return err
		}

		p, ok := peer.FromContext(stream.Context())

		if !ok {
			panic("cannot get peer from context")
		}

		info := noise.InfoFromPeer(p)

		if info == nil {
			panic("cannot get info from peer")
		}

		id := info.Get(skademlia.KeyID)

		if id == nil {
			panic("cannot get id from peer")
		}

		fmt.Printf("%s> %s\n", id, txt.Message)
	}
}

const (
	C1 = 1
	C2 = 1
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening for peers on port:", listener.Addr().(*net.TCPAddr).Port)

	keys, err := skademlia.NewKeys(C1, C2)
	if err != nil {
		panic(err)
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))

	client := skademlia.NewClient(addr, keys, skademlia.WithC1(C1), skademlia.WithC2(C2))
	client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))

	go func() {
		server := client.Listen()
		RegisterChatServer(server, &chatHandler{})

		if err := server.Serve(listener); err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	for _, addr := range flag.Args() {
		if _, err := client.Dial(addr); err != nil {
			panic(err)
		}
	}

	client.Bootstrap()

	reader := bufio.NewReader(os.Stdin)

	for {
		line, _, err := reader.ReadLine()

		if err != nil {
			panic(err)
		}

		conns := client.ClosestPeers()

		for _, conn := range conns {
			chat := NewChatClient(conn)

			stream, err := chat.Stream(context.Background())
			if err != nil {
				continue
			}

			if err := stream.Send(&Text{Message: string(line)}); err != nil {
				continue
			}
		}

	}
}
