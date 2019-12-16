package congestion

import "github.com/caddyserver/caddy/v2/quic-go/core/protocol"

type connectionStats struct {
	slowstartPacketsLost protocol.PacketNumber
	slowstartBytesLost   protocol.ByteCount
}
