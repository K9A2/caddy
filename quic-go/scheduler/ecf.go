package scheduler

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	"mime"
	"path/filepath"
	"sync"
)

var (
	highestPriorityFileTypes = map[string]int{
		"text/html":        1,
		"application/json": 1,
	}
	highPriorityFileTypes = map[string]int{
		"text/css": 1,
		"font":     1,
	}
	normalPriorityFileTypes = map[string]int{
		"application/javascript": 1,
	}
	lowPriorityFileTypes = map[string]int{
		"image/webp": 1,
		"image/jpeg": 1,
		"image/gif": 1,
		"image/png": 1,
	}
	lowestPriorityFileTypes = map[string]int{
		"text/plain": 1,
		"image/x-icon": 1,
		"unknown": 1,
	}
)

type ecfScheduler struct {

}
type streamRepository struct {
	mutex sync.Mutex

	highestPriorityQueue []protocol.StreamID
	highPriorityQueue    []protocol.StreamID
	normalPriorityQueue  []protocol.StreamID
	lowPriorityQueue     []protocol.StreamID
	lowestPriorityQueue  []protocol.StreamID
}

var repo = streamRepository{}

func (r *streamRepository) popNextActiveStream() protocol.StreamID {
	defer repo.mutex.Unlock()
	repo.mutex.Lock()

	if len(r.highestPriorityQueue) > 0 {
		return r.highestPriorityQueue[0]
	} else if len(r.highPriorityQueue) > 0 {
		return r.highPriorityQueue[0]
	} else if len(r.normalPriorityQueue) > 0 {
		return r.normalPriorityQueue[0]
	} else if len(r.lowPriorityQueue) > 0 {
		return r.lowPriorityQueue[0]
	} else if len(r.lowestPriorityQueue) > 0 {
		return r.lowestPriorityQueue[0]
	}

	return -1
}

func (r *streamRepository) activeStreamsCount() int {
	defer r.mutex.Unlock()
	r.mutex.Lock()
	return len(r.highestPriorityQueue) + len(r.highPriorityQueue) + len(r.normalPriorityQueue) +
		len(r.lowPriorityQueue) + len(r.lowestPriorityQueue)
}

func GetMimeType(fileName string) string {
	fileExt := filepath.Ext(fileName)
	var mtype string
	switch fileExt {
	// deal with some special cases
	case "":
		//fmt.Printf("converted fileName = %v to mime-type = %v\n", fileName, "text/html")
		//return "text/html"
		mtype = "text/html"
	case ".woff2":
		//fmt.Printf("converted fileName = %v to mime-type = %v\n", fileName, "font")
		//return "font"
		mtype = "font"
	}

	if mtype == "" {
		mtype = mime.TypeByExtension(filepath.Ext(fileName))
		if mtype == "" {
			mtype = "unknown"
			fmt.Printf("warning: unknown mimetype for url = %v\n", fileName)
		}
	}

	//fmt.Printf("granted mime-type: %v\n", mtype)
	return mtype
}

func insertBySize(id protocol.StreamID, array []protocol.StreamID, getter streamGetter) []protocol.StreamID {
	if array == nil || len(array) == 0 {
		return []protocol.StreamID{id}
	} else {
		return append(array, id)
	}

	// stream to be inserted
	//str, _ := getter.GetOrOpenSendStream(id)
	//curSize := str.getDataSize()
	//fmt.Printf("target = %v, url = %v\n", float32(curSize) / 1000.0, str.GetUrl())
	//result := make([]protocol.StreamID, 0)
	//// find a proper position to insert it
	//for i := 0; i < len(array); i++ {
	//	next, _ := getter.GetOrOpenSendStream(array[i])
	//	if curSize < next.getDataSize() {
	//		// insert before this position
	//		tempLeft := array[:i]
	//		tempRight := array[i:]
	//		result = append(result, tempLeft...)
	//		result = append(result, id)
	//		result = append(result, tempRight...)
	//	}
	//}
	//
	//// print debug info
	//for i := 0; i < len(result); i++ {
	//	str, _ := getter.GetOrOpenSendStream(result[i])
	//	fmt.Printf("%v, ", float32(str.getDataSize()) / 1000.0)
	//}
	//fmt.Printf("\n")

	//return result
}

func remove(id protocol.StreamID, array []protocol.StreamID) []protocol.StreamID {
	// linear search for targeted stream
	result := make([]protocol.StreamID, 0)
	length := len(array)
	if length == 1 || length == 0 {
		return result
	}
	for i := 0; i < length; i++ {
		if array[i] == id {
			// hit the target at ith position
			result = append(result, array[:i]...)
			result = append(result, array[i+1:]...)
			return result
		}
	}
	return result
}

// removeIdleStream a stream from streamRepository
func removeIdleStream(id protocol.StreamID, getter streamGetter) {
	// add to custom stream repository
	str, _ := getter.GetOrOpenSendStream(id)
	if str == nil {
		return
	}
	mtype := str.GetMtype()
	if _, ok := highestPriorityFileTypes[mtype]; ok {
		repo.highestPriorityQueue = remove(id, repo.highestPriorityQueue)
	} else if _, ok := highPriorityFileTypes[mtype]; ok {
		repo.highPriorityQueue = remove(id, repo.highPriorityQueue)
	} else if _, ok := normalPriorityFileTypes[mtype]; ok {
		repo.normalPriorityQueue = remove(id, repo.normalPriorityQueue)
	} else if _, ok := lowPriorityFileTypes[mtype]; ok {
		repo.lowPriorityQueue = remove(id, repo.lowPriorityQueue)
	} else if _, ok := lowestPriorityFileTypes[mtype]; ok {
		repo.lowestPriorityQueue = remove(id, repo.lowestPriorityQueue)
	}
}
