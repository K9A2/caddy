package quic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/logger"
	"github.com/caddyserver/caddy/v2/quic-go/core/ackhandler"
	"github.com/caddyserver/caddy/v2/quic-go/core/flowcontrol"
	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	"github.com/caddyserver/caddy/v2/quic-go/core/utils"
	"github.com/caddyserver/caddy/v2/quic-go/core/wire"
)

type sendStreamI interface {
	SendStream
	handleStopSendingFrame(*wire.StopSendingFrame)
	hasData() bool
	popStreamFrame(maxBytes protocol.ByteCount) (*ackhandler.Frame, bool)
	closeForShutdown(error)
	handleMaxStreamDataFrame(*wire.MaxStreamDataFrame)

	getDataSize() int
	setDataToWrite([]byte)
	SetUrl(url string)
	GetUrl() string
	GetMtype() string
	SetMtype(string)
}

type sendStream struct {
	mutex sync.Mutex

	numOutstandingFrames int64
	retransmissionQueue  []*wire.StreamFrame

	ctx       context.Context
	ctxCancel context.CancelFunc

	streamID protocol.StreamID
	sender   streamSender

	writeOffset protocol.ByteCount

	cancelWriteErr      error
	closeForShutdownErr error

	closedForShutdown bool // set when CloseForShutdown() is called
	finishedWriting   bool // set once Close() is called
	canceledWrite     bool // set when CancelWrite() is called, or a STOP_SENDING frame is received
	finSent           bool // set when a STREAM_FRAME with FIN bit has been sent
	completed         bool // set when this stream has been reported to the streamSender as completed

	DataForWriting []byte

	writeChan chan struct{}
	deadline  time.Time

	flowController flowcontrol.StreamFlowController

	version protocol.VersionNumber

	url string
	mtype string
}

var _ SendStream = &sendStream{}
var _ sendStreamI = &sendStream{}

func newSendStream(
	streamID protocol.StreamID,
	sender streamSender,
	flowController flowcontrol.StreamFlowController,
	version protocol.VersionNumber,
) *sendStream {
	s := &sendStream{
		streamID:       streamID,
		sender:         sender,
		flowController: flowController,
		writeChan:      make(chan struct{}, 1),
		version:        version,
	}
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	return s
}

func (s *sendStream) getDataSize() int {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	return len(s.DataForWriting)
}

// 通过这个函数把 stream 要写入的数据替换为内存中的数据
func (s *sendStream) setDataToWrite(content []byte) {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	s.DataForWriting = content
	logger.Info("data replaced")
}

func (s *sendStream) SetUrl(url string) {
	s.url = url
}

func (s *sendStream) GetUrl() string {
	return s.url
}

func (s *sendStream) GetMtype() string {
	return s.mtype
}

func (s *sendStream) SetMtype(mtype string) {
	s.mtype = mtype
}

func (s *sendStream) StreamID() protocol.StreamID {
	return s.streamID // same for receiveStream and sendStream
}

func (s *sendStream) Write(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.finishedWriting {
		return 0, fmt.Errorf("write on closed stream %d", s.streamID)
	}
	if s.canceledWrite {
		return 0, s.cancelWriteErr
	}
	if s.closeForShutdownErr != nil {
		return 0, s.closeForShutdownErr
	}
	if !s.deadline.IsZero() && !time.Now().Before(s.deadline) {
		return 0, errDeadline
	}
	if len(p) == 0 {
		return 0, nil
	}

	s.DataForWriting = p

	//fmt.Printf("stream %v write with data size = %v\n", s.StreamID(), s.getDataSize())

	var (
		deadlineTimer  *utils.Timer
		bytesWritten   int
		notifiedSender bool
	)
	for {
		bytesWritten = len(p) - len(s.DataForWriting)
		deadline := s.deadline
		if !deadline.IsZero() {
			if !time.Now().Before(deadline) {
				s.DataForWriting = nil
				return bytesWritten, errDeadline
			}
			if deadlineTimer == nil {
				deadlineTimer = utils.NewTimer()
			}
			deadlineTimer.Reset(deadline)
		}
		if s.DataForWriting == nil || s.canceledWrite || s.closedForShutdown {
			break
		}

		s.mutex.Unlock()
		if !notifiedSender {
			s.sender.onHasStreamData(s.streamID) // must be called without holding the mutex
			notifiedSender = true
		}
		if deadline.IsZero() {
			<-s.writeChan
		} else {
			select {
			case <-s.writeChan:
			case <-deadlineTimer.Chan():
				deadlineTimer.SetRead()
			}
		}
		s.mutex.Lock()
	}

	if s.closeForShutdownErr != nil {
		return bytesWritten, s.closeForShutdownErr
	} else if s.cancelWriteErr != nil {
		return bytesWritten, s.cancelWriteErr
	}
	return bytesWritten, nil
}

// popStreamFrame returns the next STREAM frame that is supposed to be sent on this stream
// maxBytes is the maximum length this frame (including frame header) will have.
func (s *sendStream) popStreamFrame(maxBytes protocol.ByteCount) (*ackhandler.Frame, bool /* has more data to send */) {
	s.mutex.Lock()
	logger.Infof("popStreamFrame: hit with maxBytes <%v>", maxBytes)
	f, hasMoreData := s.popNewOrRetransmittedStreamFrame(maxBytes)
	if f != nil {
		s.numOutstandingFrames++
	}
	s.mutex.Unlock()

	if f == nil {
		return nil, hasMoreData
	}
	return &ackhandler.Frame{Frame: f, OnLost: s.queueRetransmission, OnAcked: s.frameAcked}, hasMoreData
}

func (s *sendStream) popNewOrRetransmittedStreamFrame(maxBytes protocol.ByteCount) (*wire.StreamFrame, bool /* has more data to send */) {
	if len(s.retransmissionQueue) > 0 {
		f, hasMoreRetransmissions := s.maybeGetRetransmission(maxBytes)
		if f != nil || hasMoreRetransmissions {
			if f == nil {
				return nil, true
			}
			// We always claim that we have more data to send.
			// This might be incorrect, in which case there'll be a spurious call to popStreamFrame in the future.
			return f, true
		}
	}

	f := wire.GetStreamFrame()
	f.FinBit = false
	f.StreamID = s.streamID
	f.Offset = s.writeOffset
	f.DataLenPresent = true
	f.Data = f.Data[:0]

	logger.Infof("popNewOrRetransmittedStreamFrame: hit with maxBytes <%v>")

	hasMoreData := s.popNewStreamFrame(f, maxBytes)

	if len(f.Data) == 0 && !f.FinBit {
		f.PutBack()
		return nil, hasMoreData
	}
	return f, hasMoreData
}

func (s *sendStream) popNewStreamFrame(f *wire.StreamFrame, maxBytes protocol.ByteCount) bool {
	if s.canceledWrite || s.closeForShutdownErr != nil {
		return false
	}

	logger.Infof("popNewStreamFrame: hit with maxBytes <%v>", maxBytes)
	maxDataLen := f.MaxDataLen(maxBytes, s.version)
	if maxDataLen == 0 { // a STREAM frame must have at least one byte of data
		return s.DataForWriting != nil
	}
	s.getDataForWriting(f, maxDataLen)
	if len(f.Data) == 0 && !f.FinBit {
		// this can happen if:
		// - popStreamFrame is called but there's no data for writing
		// - there's data for writing, but the stream is stream-level flow control blocked
		// - there's data for writing, but the stream is connection-level flow control blocked
		if s.DataForWriting == nil {
			return false
		}
		if isBlocked, offset := s.flowController.IsNewlyBlocked(); isBlocked {
			s.sender.queueControlFrame(&wire.StreamDataBlockedFrame{
				StreamID:  s.streamID,
				DataLimit: offset,
			})
			return false
		}
		return true
	}
	if f.FinBit {
		s.finSent = true
	}
	return s.DataForWriting != nil
}

func (s *sendStream) maybeGetRetransmission(maxBytes protocol.ByteCount) (*wire.StreamFrame, bool /* has more retransmissions */) {
	f := s.retransmissionQueue[0]
	newFrame, needsSplit := f.MaybeSplitOffFrame(maxBytes, s.version)
	if needsSplit {
		return newFrame, true
	}
	s.retransmissionQueue = s.retransmissionQueue[1:]
	return f, len(s.retransmissionQueue) > 0
}

func (s *sendStream) hasData() bool {
	s.mutex.Lock()
	hasData := len(s.DataForWriting) > 0
	s.mutex.Unlock()
	return hasData
}

func (s *sendStream) getDataForWriting(f *wire.StreamFrame, maxBytes protocol.ByteCount) {
	if s.DataForWriting == nil {
		f.FinBit = s.finishedWriting && !s.finSent
		return
	}

	maxBytes = utils.MinByteCount(maxBytes, s.flowController.SendWindowSize())
	if maxBytes == 0 {
		logger.Infof("getDataForWriting: not enough send window")
		return
	}

	logger.Infof("getDataForWriting: maxBytes <%v>", maxBytes)

	if protocol.ByteCount(len(s.DataForWriting)) > maxBytes {
		f.Data = f.Data[:maxBytes]
		copy(f.Data, s.DataForWriting)
		s.DataForWriting = s.DataForWriting[maxBytes:]
	} else {
		f.Data = f.Data[:len(s.DataForWriting)]
		copy(f.Data, s.DataForWriting)
		s.DataForWriting = nil
		s.signalWrite()
	}
	s.writeOffset += f.DataLen()
	s.flowController.AddBytesSent(f.DataLen())
	f.FinBit = s.finishedWriting && s.DataForWriting == nil && !s.finSent
}

func (s *sendStream) frameAcked(f wire.Frame) {
	f.(*wire.StreamFrame).PutBack()

	s.mutex.Lock()
	s.numOutstandingFrames--
	if s.numOutstandingFrames < 0 {
		panic("numOutStandingFrames negative")
	}
	newlyCompleted := s.isNewlyCompleted()
	s.mutex.Unlock()

	if newlyCompleted {
		s.sender.onStreamCompleted(s.streamID)
	}
}

func (s *sendStream) isNewlyCompleted() bool {
	completed := (s.finSent || s.canceledWrite) && s.numOutstandingFrames == 0 && len(s.retransmissionQueue) == 0
	if completed && !s.completed {
		s.completed = true
		return true
	}
	return false
}

func (s *sendStream) queueRetransmission(f wire.Frame) {
	//fmt.Printf("calling in send_stream.go - queueRetransmission(), size = %v\n", s.getDataSize())
	sf := f.(*wire.StreamFrame)
	sf.DataLenPresent = true
	s.mutex.Lock()
	s.retransmissionQueue = append(s.retransmissionQueue, sf)
	s.numOutstandingFrames--
	if s.numOutstandingFrames < 0 {
		panic("numOutStandingFrames negative")
	}
	s.mutex.Unlock()

	s.sender.onHasStreamData(s.streamID)
}

func (s *sendStream) Close() error {
	s.mutex.Lock()
	if s.canceledWrite {
		s.mutex.Unlock()
		return fmt.Errorf("Close called for canceled stream %d", s.streamID)
	}
	s.ctxCancel()
	s.finishedWriting = true
	s.mutex.Unlock()

	s.sender.onHasStreamData(s.streamID) // need to send the FIN, must be called without holding the mutex
	return nil
}

func (s *sendStream) CancelWrite(errorCode protocol.ApplicationErrorCode) {
	s.cancelWriteImpl(errorCode, fmt.Errorf("Write on stream %d canceled with error code %d", s.streamID, errorCode))

}

// must be called after locking the mutex
func (s *sendStream) cancelWriteImpl(errorCode protocol.ApplicationErrorCode, writeErr error) {
	s.mutex.Lock()
	if s.canceledWrite {
		s.mutex.Unlock()
		return
	}
	s.ctxCancel()
	s.canceledWrite = true
	s.cancelWriteErr = writeErr
	newlyCompleted := s.isNewlyCompleted()
	s.mutex.Unlock()

	s.signalWrite()
	s.sender.queueControlFrame(&wire.ResetStreamFrame{
		StreamID:   s.streamID,
		ByteOffset: s.writeOffset,
		ErrorCode:  errorCode,
	})
	if newlyCompleted {
		s.sender.onStreamCompleted(s.streamID)
	}
}

func (s *sendStream) handleMaxStreamDataFrame(frame *wire.MaxStreamDataFrame) {
	//fmt.Printf("calling in send_stream.go - handleMaxStreamDataFrame(), size = %v\n", s.getDataSize())
	s.mutex.Lock()
	hasStreamData := s.DataForWriting != nil
	s.mutex.Unlock()

	s.flowController.UpdateSendWindow(frame.ByteOffset)
	if hasStreamData {
		s.sender.onHasStreamData(s.streamID)
	}
}

func (s *sendStream) handleStopSendingFrame(frame *wire.StopSendingFrame) {
	writeErr := streamCanceledError{
		errorCode: frame.ErrorCode,
		error:     fmt.Errorf("stream %d was reset with error code %d", s.streamID, frame.ErrorCode),
	}
	s.cancelWriteImpl(frame.ErrorCode, writeErr)
}

func (s *sendStream) Context() context.Context {
	return s.ctx
}

func (s *sendStream) SetWriteDeadline(t time.Time) error {
	s.mutex.Lock()
	s.deadline = t
	s.mutex.Unlock()
	s.signalWrite()
	return nil
}

// CloseForShutdown closes a stream abruptly.
// It makes Write unblock (and return the error) immediately.
// The peer will NOT be informed about this: the stream is closed without sending a FIN or RST.
func (s *sendStream) closeForShutdown(err error) {
	s.mutex.Lock()
	s.ctxCancel()
	s.closedForShutdown = true
	s.closeForShutdownErr = err
	s.mutex.Unlock()
	s.signalWrite()
}

// signalWrite performs a non-blocking send on the writeChan
func (s *sendStream) signalWrite() {
	select {
	case s.writeChan <- struct{}{}:
	default:
	}
}
