package main

import (
	"flag"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daseinio/dasein-p2p/crypto/ed25519"
	"github.com/daseinio/dasein-p2p/examples/request_benchmark/messages"
	"github.com/daseinio/dasein-p2p/network"
	"github.com/daseinio/dasein-p2p/network/discovery"
	"github.com/daseinio/dasein-p2p/network/rpc"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	defaultNumNodes      = 5
	defaultNumReqPerNode = 50
	host                 = "localhost"
	startPort            = 23000
)

func main() {
	// send glog to the terminal instead of a file
	flag.Set("logtostderr", "true")

	fmt.Print(run())
}

func run() string {

	runtime.GOMAXPROCS(runtime.NumCPU())

	numReqPerNodeFlag := flag.Int("r", defaultNumReqPerNode, "Number of requests per node")
	numNodesFlag := flag.Int("n", defaultNumNodes, "Number of nodes")

	flag.Parse()

	numNodes := *numNodesFlag
	numReqPerNode := *numReqPerNodeFlag

	nets := setupNetworks(host, startPort, numNodes)
	expectedTotalResp := numReqPerNode * numNodes * (numNodes - 1)
	var totalPos uint32

	startTime := time.Now()

	wg := &sync.WaitGroup{}

	// sending to all nodes concurrently
	for r := 0; r < numReqPerNode; r++ {
		for n, nt := range nets {
			wg.Add(1)
			go func(net *network.Network, idx int) {
				defer wg.Done()
				positive := sendMsg(net, idx)
				atomic.AddUint32(&totalPos, positive)
			}(nt, n+numNodes*r)
		}
	}
	wg.Wait()

	totalTime := time.Since(startTime)
	reqPerSec := float64(totalPos) / totalTime.Seconds()

	return fmt.Sprintf("Test completed in %s, num nodes = %d, successful requests = %d / %d, requestsPerSec = %f\n",
		totalTime, numNodes, totalPos, expectedTotalResp, reqPerSec)
}

func setupNetworks(host string, startPort int, numNodes int) []*network.Network {
	var nodes []*network.Network

	for i := 0; i < numNodes; i++ {
		builder := network.NewBuilder()
		builder.SetKeys(ed25519.RandomKeyPair())
		builder.SetAddress(network.FormatAddress("tcp", host, uint16(startPort+i)))

		builder.AddComponent(new(discovery.Component))
		builder.AddComponent(new(loadTestComponent))

		node, err := builder.Build()
		if err != nil {
			fmt.Println(err)
		}

		go node.Listen()

		nodes = append(nodes, node)
	}

	// Make sure all nodes are listening for incoming peers.
	for _, node := range nodes {
		node.BlockUntilListening()
	}

	// Bootstrap to Node 0.
	for i, node := range nodes {
		if i != 0 {
			node.Bootstrap(nodes[0].Address)
		}
	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1 * time.Second)

	return nodes
}

func sendMsg(net *network.Network, idx int) uint32 {
	var positiveResponses uint32

	Component, registered := net.Component(discovery.ComponentID)
	if !registered {
		return 0
	}

	routes := Component.(*discovery.Component).Routes

	addresses := routes.GetPeerAddresses()

	errs := make(chan error, len(addresses))

	wg := &sync.WaitGroup{}

	for _, address := range addresses {
		wg.Add(1)

		go func(address string) {
			defer wg.Done()

			expectedID := fmt.Sprintf("%s:%d->%s", net.Address, idx, address)
			request := &rpc.Request{}
			request.SetTimeout(3 * time.Second)
			request.SetMessage(&messages.LoadRequest{Id: expectedID})

			client, err := net.Client(address)
			if err != nil {
				errs <- errors.Wrapf(err, "client error for req id %s", expectedID)
				return
			}

			response, err := client.Request(request)
			if err != nil {
				errs <- errors.Wrapf(err, "request error for req id %s", expectedID)
				return
			}

			if reply, ok := response.(*messages.LoadReply); ok {
				if reply.Id == expectedID {
					atomic.AddUint32(&positiveResponses, 1)
				} else {
					errs <- errors.Errorf("expected ID=%s got %s\n", expectedID, reply.Id)
				}
			} else {
				errs <- errors.Errorf("expected messages.LoadReply but got %v\n", response)
			}

		}(address)
	}

	wg.Wait()

	close(errs)

	for err := range errs {
		glog.Error(err)
	}

	return atomic.LoadUint32(&positiveResponses)
}

type loadTestComponent struct {
	*network.Component
}

// Receive takes in *messages.ProxyMessage and replies with *messages.ID
func (p *loadTestComponent) Receive(ctx *network.ComponentContext) error {
	switch msg := ctx.Message().(type) {
	case *messages.LoadRequest:
		response := &messages.LoadReply{Id: msg.Id}
		ctx.Reply(response)
	}

	return nil
}
