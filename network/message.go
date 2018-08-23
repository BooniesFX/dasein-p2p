package network

import (
	"github.com/daseinio/dasein-p2p/peer"
	"github.com/gogo/protobuf/proto"
)

// ComponentContext provides parameters and helper functions to a Component
// for interacting with/analyzing incoming messages from a select peer.
type ComponentContext struct {
	client  *PeerClient
	message proto.Message
	nonce   uint64
}

// Reply sends back a message to an incoming message's incoming stream.
func (ctx *ComponentContext) Reply(message proto.Message) error {
	return ctx.client.Reply(ctx.nonce, message)
}

// Message returns the decoded protobuf message.
func (ctx *ComponentContext) Message() proto.Message {
	return ctx.message
}

// Client returns the peer client.
func (ctx *ComponentContext) Client() *PeerClient {
	return ctx.client
}

// Network returns the entire node's network.
func (ctx *ComponentContext) Network() *Network {
	return ctx.client.Network
}

// Self returns the node's ID.
func (ctx *ComponentContext) Self() peer.ID {
	return ctx.Network().ID
}

// Sender returns the peer's ID.
func (ctx *ComponentContext) Sender() peer.ID {
	return *ctx.client.ID
}
