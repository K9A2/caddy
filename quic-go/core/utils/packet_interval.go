package utils

import "github.com/caddyserver/caddy/v2/quic-go/core/protocol"

// PacketInterval is an interval from one PacketNumber to the other
type PacketInterval struct {
	Start protocol.PacketNumber
	End   protocol.PacketNumber
}
