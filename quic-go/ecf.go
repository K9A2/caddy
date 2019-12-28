package quic

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	"mime"
	"path/filepath"
	"strings"
)

var (
	highestPriorityFileTypes = map[string]uint8{
		"text/html":        1,
		"application/json": 1,
	}
	highPriorityFileTypes = map[string]uint8{
		"text/css": 1,
		"font":     1,
	}
	normalPriorityFileTypes = map[string]uint8{
		"application/javascript": 1,
	}
	lowPriorityFileTypes = map[string]uint8{
		"image/webp": 1,
		"image/jpeg": 1,
		"image/gif":  1,
		"image/png":  1,
	}
	lowestPriorityFileTypes = map[string]uint8{
		"text/plain":   1,
		"image/x-icon": 1,
		"unknown":      1,
	}
)

type ecfScheduler struct {
	highestPriorityQueue []protocol.StreamID
	highPriorityQueue    []protocol.StreamID
	normalPriorityQueue  []protocol.StreamID
	lowPriorityQueue     []protocol.StreamID
	lowestPriorityQueue  []protocol.StreamID
}

func NewEcfScheduler() *ecfScheduler {
	return &ecfScheduler{
		highestPriorityQueue: make([]protocol.StreamID, 0),
		highPriorityQueue:    make([]protocol.StreamID, 0),
		normalPriorityQueue:  make([]protocol.StreamID, 0),
		lowPriorityQueue:     make([]protocol.StreamID, 0),
		lowestPriorityQueue:  make([]protocol.StreamID, 0),
	}
}

// pop the first stream in the highest available queue
func (s *ecfScheduler) PopNextActiveStream() protocol.StreamID {
	var result protocol.StreamID
	if len(s.highestPriorityQueue) > 0 {
		result = s.highestPriorityQueue[0]
		//fmt.Printf("pop stream %v from highest priority queue\n", result)
		return result
	} else if len(s.highPriorityQueue) > 0 {
		result = s.highPriorityQueue[0]
		//fmt.Printf("pop stream %v from high priority queue\n", result)
		return result
	} else if len(s.normalPriorityQueue) > 0 {
		result = s.normalPriorityQueue[0]
		//fmt.Printf("pop stream %v from normal priority queue\n", result)
		return result
	} else if len(s.lowPriorityQueue) > 0 {
		result = s.lowPriorityQueue[0]
		//fmt.Printf("pop stream %v from low priority queue\n", result)
		return result
	} else if len(s.lowestPriorityQueue) > 0 {
		result = s.lowestPriorityQueue[0]
		//fmt.Printf("pop stream %v from lowest priority queue\n", result)
		return result
	} else {
		//fmt.Println("  warning: not enough stream to return")
		return -1
	}
}

func (s *ecfScheduler) ActiveStreamsCount() int {
	return len(s.highestPriorityQueue) + len(s.highPriorityQueue) + len(s.normalPriorityQueue) +
		len(s.lowPriorityQueue) + len(s.lowestPriorityQueue)
}

func GetMimeType(fileName string) string {
	fileExt := filepath.Ext(fileName)
	var mtype string
	switch fileExt {
	// deal with some special cases
	case "":
		mtype = "text/html"
	case ".woff2":
		mtype = "font"
	}

	if mtype == "" {
		mtype = mime.TypeByExtension(filepath.Ext(fileName))
		if mtype == "" {
			mtype = "unknown"
			fmt.Printf("warning: unknown mimetype for url = %v\n", fileName)
		}
		semicolonIndex := strings.Index(mtype, ";")
		if semicolonIndex > 0 {
			// the mtype contains a substring likes "text/css; charset=utf-8",
			// the substring after semicolon has to be removed
			mtype = mtype[:semicolonIndex]
		}
	}
	//fmt.Printf("grant [%v] with mime type [%v]\n", fileName, mtype)
	return mtype
}

// insert an active stream by its mime type
func (s *ecfScheduler) InsertByType(id protocol.StreamID, mtype string, url string) {
	if _, ok := highestPriorityFileTypes[mtype]; ok {
		s.highestPriorityQueue = insert(id, s.highestPriorityQueue)
		//fmt.Printf("insert stream %v, type [%v], url [%v] to highest priority queue\n", id, mtype, url)
		return
	} else if _, ok := highPriorityFileTypes[mtype]; ok {
		s.highPriorityQueue = insert(id, s.highPriorityQueue)
		//fmt.Printf("insert stream %v, type [%v], url [%v] to high priority queue\n", id, mtype, url)
		return
	} else if _, ok := normalPriorityFileTypes[mtype]; ok {
		s.normalPriorityQueue = insert(id, s.normalPriorityQueue)
		//fmt.Printf("insert stream %v, type [%v], url [%v] to normal priority queue\n", id, mtype, url)
		return
	} else if _, ok := lowPriorityFileTypes[mtype]; ok {
		s.lowPriorityQueue = insert(id, s.lowPriorityQueue)
		//fmt.Printf("insert stream %v, type [%v], url [%v] to low priority queue\n", id, mtype, url)
		return
	} else if _, ok := lowestPriorityFileTypes[mtype]; ok {
		s.lowestPriorityQueue = insert(id, s.lowestPriorityQueue)
		//fmt.Printf("insert stream %v, type [%v], url [%v] to lowest priority queue\n", id, mtype, url)
		return
	}
}

func insert(id protocol.StreamID, array []protocol.StreamID) []protocol.StreamID {
	if array == nil || len(array) == 0 {
		return []protocol.StreamID{id}
	} else {
		return append(array, id)
	}
}

func removeFrom(id protocol.StreamID, array []protocol.StreamID) []protocol.StreamID {
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

func contains(id protocol.StreamID, array []protocol.StreamID) bool {
	for _, val := range array {
		if val == id {
			return true
		}
	}
	return false
}

func (s *ecfScheduler) RemoveNilStream(id protocol.StreamID) {
	// as we do not know the mime type of give stream, we need to search it everywhere
	var message string
	if len(s.highestPriorityQueue) > 0 && contains(id, s.highestPriorityQueue){
		lenBefore := len(s.highestPriorityQueue)
		s.highestPriorityQueue = removeFrom(id, s.highestPriorityQueue)
		lenAfter := len(s.highestPriorityQueue)
		message = fmt.Sprintf("nil string %v removed from highest priority queue, len before %v, len after %v\n",
			id, lenBefore, lenAfter)
	} else if len(s.highPriorityQueue) > 0 && contains(id, s.highPriorityQueue) {
		lenBefore := len(s.highPriorityQueue)
		s.highPriorityQueue = removeFrom(id, s.highPriorityQueue)
		lenAfter := len(s.highPriorityQueue)
		message = fmt.Sprintf("nil string %v removed from high priority queue, len before %v, len after %v\n",
			id, lenBefore, lenAfter)
	} else if len(s.normalPriorityQueue) > 0 && contains(id, s.normalPriorityQueue) {
		lenBefore := len(s.normalPriorityQueue)
		s.normalPriorityQueue = removeFrom(id, s.normalPriorityQueue)
		lenAfter := len(s.normalPriorityQueue)
		message = fmt.Sprintf("nil string %v removed from normal priority queue, len before %v, len after %v\n",
			id, lenBefore, lenAfter)
	} else if len(s.lowPriorityQueue) > 0 && contains(id, s.lowPriorityQueue) {
		lenBefore := len(s.lowPriorityQueue)
		s.lowPriorityQueue = removeFrom(id, s.lowPriorityQueue)
		lenAfter := len(s.lowPriorityQueue)
		message = fmt.Sprintf("nil string %v removed from low priority queue, len before %v, len after %v\n",
			id, lenBefore, lenAfter)
	} else if len(s.lowestPriorityQueue) > 0 && contains(id, s.lowestPriorityQueue) {
		lenBefore := len(s.lowestPriorityQueue)
		s.lowestPriorityQueue = removeFrom(id, s.lowestPriorityQueue)
		lenAfter := len(s.lowestPriorityQueue)
		message = fmt.Sprintf("nil string %v removed from lowest priority queue, len before %v, len after %v\n",
			id, lenBefore, lenAfter)
	} else {
		message = "error: removing an non-existent stream"
	}
	fmt.Println(message)
}

// removeIdleStream a stream from streamRepository
func (s *ecfScheduler) RemoveStream(id protocol.StreamID, mtype string, url string) {
	// add to custom stream repository
	if _, ok := highestPriorityFileTypes[mtype]; ok {
		s.highestPriorityQueue = removeFrom(id, s.highestPriorityQueue)
		//fmt.Printf("remove stream %v, type [%v], url [%v] from highest priority queue\n", id, mtype, url)
	} else if _, ok := highPriorityFileTypes[mtype]; ok {
		s.highPriorityQueue = removeFrom(id, s.highPriorityQueue)
		//fmt.Printf("remove stream %v, type [%v], url [%v] from high priority queue\n", id, mtype, url)
	} else if _, ok := normalPriorityFileTypes[mtype]; ok {
		s.normalPriorityQueue = removeFrom(id, s.normalPriorityQueue)
		//fmt.Printf("remove stream %v, type [%v], url [%v] from normal priority queue\n", id, mtype, url)
	} else if _, ok := lowPriorityFileTypes[mtype]; ok {
		s.lowPriorityQueue = removeFrom(id, s.lowPriorityQueue)
		//fmt.Printf("remove stream %v, type [%v], url [%v] from low priority queue\n", id, mtype, url)
	} else if _, ok := lowestPriorityFileTypes[mtype]; ok {
		s.lowestPriorityQueue = removeFrom(id, s.lowestPriorityQueue)
		//fmt.Printf("remove stream %v, type [%v], url [%v] from lowest priority queue\n", id, mtype, url)
	}
}
