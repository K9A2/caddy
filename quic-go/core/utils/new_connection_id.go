package utils

import (
	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
)

// NewConnectionID is a new connection ID
type NewConnectionID struct {
	SequenceNumber      uint64
	ConnectionID        protocol.ConnectionID
	StatelessResetToken *[16]byte
}
