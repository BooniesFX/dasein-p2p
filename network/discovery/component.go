package discovery

import (
	"strings"

	"github.com/daseinio/dasein-p2p/dht"
	"github.com/daseinio/dasein-p2p/internal/protobuf"
	"github.com/daseinio/dasein-p2p/network"
	"github.com/daseinio/dasein-p2p/peer"
	"github.com/golang/glog"
)

type Component struct {
	*network.Component

	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes *dht.RoutingTable
}

var (
	ComponentID                            = (*Component)(nil)
	_           network.ComponentInterface = (*Component)(nil)
)

func (state *Component) Startup(net *network.Network) {
	// Create routing table.
	state.Routes = dht.CreateRoutingTable(net.ID)
}

func (state *Component) Receive(ctx *network.ComponentContext) error {
	// Update routing for every incoming message.
	state.Routes.Update(ctx.Sender())

	// Handle RPC.
	switch msg := ctx.Message().(type) {
	case *protobuf.Ping:
		if state.DisablePing {
			break
		}

		// Send pong to peer.
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if state.DisablePong {
			break
		}

		peers := FindNode(ctx.Network(), ctx.Sender(), dht.BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			state.Routes.Update(peerID)
		}

		glog.Infof("bootstrapped w/ peer(s): %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	case *protobuf.LookupNodeRequest:
		if state.DisableLookup {
			break
		}

		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range state.Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := ctx.Reply(response)
		if err != nil {
			return err
		}

		glog.Infof("connected peers: %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	}

	return nil
}

func (state *Component) Cleanup(net *network.Network) {
	// TODO: Save routing table?
}

func (state *Component) PeerDisconnect(client *network.PeerClient) {
	// Delete peer if in routing table.
	if client.ID != nil {
		if state.Routes.PeerExists(*client.ID) {
			state.Routes.RemovePeer(*client.ID)

			glog.Infof("Peer %s has disconnected from %s.", client.ID.Address, client.Network.ID.Address)
		}
	}
}
