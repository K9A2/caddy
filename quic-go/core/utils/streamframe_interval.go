package utils

import "github.com/caddyserver/caddy/v2/quic-go/core/protocol"

// ByteInterval is an interval from one ByteCount to the other
type ByteInterval struct {
	Start protocol.ByteCount
	End   protocol.ByteCount
}
