package ackhandler

import "github.com/caddyserver/caddy/v2/quic-go/core/wire"

type Frame struct {
	wire.Frame // nil if the frame has already been acknowledged in another packet
	OnLost     func(wire.Frame)
	OnAcked    func(wire.Frame)
}
