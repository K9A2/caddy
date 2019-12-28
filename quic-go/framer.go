package quic

import (
	"sync"

	"github.com/caddyserver/caddy/v2/quic-go/core/ackhandler"
	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	"github.com/caddyserver/caddy/v2/quic-go/core/utils"
	"github.com/caddyserver/caddy/v2/quic-go/core/wire"
)

type framer interface {
	QueueControlFrame(wire.Frame)
	AppendControlFrames([]ackhandler.Frame, protocol.ByteCount) ([]ackhandler.Frame, protocol.ByteCount)

	AddActiveStream(protocol.StreamID)
	AppendStreamFrames([]ackhandler.Frame, protocol.ByteCount) ([]ackhandler.Frame, protocol.ByteCount)
}

type framerI struct {
	mutex sync.Mutex

	streamGetter streamGetter
	version      protocol.VersionNumber

	activeStreams map[protocol.StreamID]struct{}
	streamQueue   []protocol.StreamID

	controlFrameMutex sync.Mutex
	controlFrames     []wire.Frame

	scheduler *ecfScheduler
}

var _ framer = &framerI{}

func newFramer(
	streamGetter streamGetter,
	v protocol.VersionNumber,
) framer {
	return &framerI{
		streamGetter:  streamGetter,
		activeStreams: make(map[protocol.StreamID]struct{}),
		version:       v,
		scheduler:     NewEcfScheduler(),
	}
}

func (f *framerI) QueueControlFrame(frame wire.Frame) {
	f.controlFrameMutex.Lock()
	f.controlFrames = append(f.controlFrames, frame)
	f.controlFrameMutex.Unlock()
}

func (f *framerI) AppendControlFrames(frames []ackhandler.Frame, maxLen protocol.ByteCount) ([]ackhandler.Frame, protocol.ByteCount) {
	var length protocol.ByteCount
	f.controlFrameMutex.Lock()
	for len(f.controlFrames) > 0 {
		frame := f.controlFrames[len(f.controlFrames)-1]
		frameLen := frame.Length(f.version)
		if length+frameLen > maxLen {
			break
		}
		frames = append(frames, ackhandler.Frame{Frame: frame})
		length += frameLen
		f.controlFrames = f.controlFrames[:len(f.controlFrames)-1]
	}
	f.controlFrameMutex.Unlock()
	return frames, length
}

func (f *framerI) AddActiveStream(id protocol.StreamID) {
	defer f.mutex.Unlock()
	f.mutex.Lock()
	if _, ok := f.activeStreams[id]; !ok {
		// add only when it is absent
		//f.streamQueue = append(f.streamQueue, id)
		f.activeStreams[id] = struct{}{}

		// add to custom stream repository
		str, _ := f.streamGetter.GetOrOpenSendStream(id)
		//fmt.Printf("adding stream %v to transmission queue, mime-type = %v\n", str.StreamID(), str.GetMtype())
		//if str == nil || err != nil {
		//	delete(f.activeStreams, id)
		//}
		if str.GetMtype() == "" {
			// fallback method for control streams
			str.SetMtype("text/html")
			//fmt.Printf("warning: grant mimt-type unknown to stream %v\n", str.StreamID())
		}
		f.scheduler.InsertByType(id, str.GetMtype(), str.GetUrl())
	}
}

func (f *framerI) AppendStreamFrames(frames []ackhandler.Frame, maxLen protocol.ByteCount) ([]ackhandler.Frame, protocol.ByteCount) {
	var length protocol.ByteCount
	var lastFrame *ackhandler.Frame

	f.mutex.Lock()
	// pop STREAM frames, until less than MinStreamFrameSize bytes are left in the packet
	//numActiveStreams := len(f.streamQueue)
	numActiveStreams := f.scheduler.ActiveStreamsCount()
	//fmt.Printf("numActiveStreams = %v\n", numActiveStreams)
	for i := 0; i < numActiveStreams; i++ {
		if protocol.MinStreamFrameSize+length > maxLen {
			break
		}

		// removeIdleStream the first one from stream queue
		//id := f.streamQueue[0]
		//f.streamQueue = f.streamQueue[1:]
		id := f.scheduler.PopNextActiveStream()

		// This should never return an error. Better check it anyway.
		// The stream will only be in the streamQueue, if it enqueued itself there.
		str, err := f.streamGetter.GetOrOpenSendStream(id)
		// The stream can be nil if it completed after it said it had data.
		if str == nil || err != nil {
			//message := ""
			//if str == nil {
			//	message += fmt.Sprintf("str = %v is nil", id)
			//}
			//if err != nil {
			//	message += fmt.Sprintf("err = %v", err.Error())
			//}
			//message += "\n"
			//fmt.Print(message)

			delete(f.activeStreams, id)
			f.scheduler.RemoveNilStream(id)
			continue
		}

		remainingLen := maxLen - length
		// For the last STREAM frame, we'll removeIdleStream the DataLen field later.
		// Therefore, we can pretend to have more bytes available when popping
		// the STREAM frame (which will always have the DataLen set).
		remainingLen += utils.VarIntLen(uint64(remainingLen))
		frame, hasMoreData := str.popStreamFrame(remainingLen)

		// if the first stream still has more data to sent, it will still be placed
		// at the first place. if the first stream has no more data to sent, it will
		// be removeIdleStream from activeStreamQueue and activeStreams (a map).
		//if hasMoreData { // put the stream back in the queue (at the end)
		//	f.streamQueue = append(f.streamQueue, id)
		//} else { // no more data to send. Stream is not active any more
		//	delete(f.activeStreams, id)
		//}

		//fmt.Println("before removeFrom")
		if !hasMoreData {
			f.scheduler.RemoveStream(id, str.GetMtype(), str.GetUrl())
			delete(f.activeStreams, id)
		}
		//fmt.Println("after removeFrom")

		// The frame can be nil
		// * if the receiveStream was canceled after it said it had data
		// * the remaining size doesn't allow us to add another STREAM frame
		if frame == nil {
			continue
		}
		frames = append(frames, *frame)

		l := frame.Length(f.version)
		//fmt.Printf("  str = %v, mime-type = %v, url = %v, size = %v, available streams = %v\n",
		//	str.StreamID(), GetMimeType(str.GetUrl()), str.GetUrl(), l, f.scheduler.ActiveStreamsCount())

		length += l
		lastFrame = frame
	}

	f.mutex.Unlock()

	if lastFrame != nil {
		lastFrameLen := lastFrame.Length(f.version)
		// account for the smaller size of the last STREAM frame
		lastFrame.Frame.(*wire.StreamFrame).DataLenPresent = false
		length += lastFrame.Length(f.version) - lastFrameLen
	}
	return frames, length
}
