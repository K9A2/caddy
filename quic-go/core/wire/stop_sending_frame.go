package wire

import (
	"bytes"

	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	"github.com/caddyserver/caddy/v2/quic-go/core/utils"
)

// A StopSendingFrame is a STOP_SENDING frame
type StopSendingFrame struct {
	StreamID  protocol.StreamID
	ErrorCode protocol.ApplicationErrorCode
}

// parseStopSendingFrame parses a STOP_SENDING frame
func parseStopSendingFrame(r *bytes.Reader, _ protocol.VersionNumber) (*StopSendingFrame, error) {
	if _, err := r.ReadByte(); err != nil {
		return nil, err
	}

	streamID, err := utils.ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	errorCode, err := utils.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	return &StopSendingFrame{
		StreamID:  protocol.StreamID(streamID),
		ErrorCode: protocol.ApplicationErrorCode(errorCode),
	}, nil
}

// Length of a written frame
func (f *StopSendingFrame) Length(_ protocol.VersionNumber) protocol.ByteCount {
	return 1 + utils.VarIntLen(uint64(f.StreamID)) + utils.VarIntLen(uint64(f.ErrorCode))
}

func (f *StopSendingFrame) Write(b *bytes.Buffer, _ protocol.VersionNumber) error {
	b.WriteByte(0x5)
	utils.WriteVarInt(b, uint64(f.StreamID))
	utils.WriteVarInt(b, uint64(f.ErrorCode))
	return nil
}
