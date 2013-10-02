package services

import (
	"net/rpc"
)

func RegisterRpcs() {
	rpc.Register(new(Counter))
	// rpc.Register(new(Draft))
	rpc.Register(new(Pulse))
}
