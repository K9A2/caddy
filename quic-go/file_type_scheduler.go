package quic

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"

  "github.com/caddyserver/caddy/v2/quic-go/core/protocol"
)

type sequenceFile struct {
  HighestPriorityQueue    []string `json:"highest"`
  HighPriorityQueue       []string `json:"high"`
  NormalPriorityQueue     []string `json:"normal"`
  LowPriorityQueue        []string `json:"low"`
  LowestPriorityQueue     []string `json:"lowest"`
  BackgroundPriorityQueue []string `json:"background"`
}

// FileTypeScheduler 是基于文件类型和文件大小的调度器
type FileTypeScheduler struct {
  // 对包含在配置文件内的资源提供六种不同的调度优先级
  highestPriorityQueue    []*streamControlBlock // 1
  highPriorityQueue       []*streamControlBlock // 2
  normalPriorityQueue     []*streamControlBlock // 3
  lowPriorityQueue        []*streamControlBlock // 4
  lowestPriorityQueue     []*streamControlBlock // 5
  backgroundPriorityQueue []*streamControlBlock // 6

  // 每条队列中活跃的 stream 数目，unmanaged 队列直接返回长度即可
  highestPriorityQueueActiveCount    int
  highPriorityQueueActiveCount       int
  normalPriorityQueueActiveCount     int
  lowPriorityQueueActiveCount        int
  lowestPriorityQueueActiveCount     int
  backgroundPriorityQueueActiveCount int

  // 未被配置文件包含的资源统一放入此队列，按照 RR 算法进行调度，最终会以最高优先级发送
  unmanagedStreamQueue []protocol.StreamID // -1

  // 以下 map 记录了哪一个 stream 在那一条队列中
  streamQueueMap map[string]int
}

// 读取文件传输顺序，成功读取时会返回解析后的文件顺序数据
func loadConfig(fileName string) *sequenceFile {
  jsonFile, err := os.Open(fileName)
  if err != nil {
    fmt.Printf("error in loading file sequence, err = <%s>\n", err.Error())
    return nil
  }
  defer jsonFile.Close()

  byteValue, _ := ioutil.ReadAll(jsonFile)
  var sequence sequenceFile
  json.Unmarshal(byteValue, &sequence)
  return &sequence
}

// NewFileTypeScheduler 根据加载的配置文件构造一个新的文件类型调度器
func NewFileTypeScheduler() *FileTypeScheduler {
  sequence := loadConfig("static-file-type-sequence.json")
  fts := FileTypeScheduler{
    highestPriorityQueue: make([]*streamControlBlock, 0),
    highPriorityQueue:    make([]*streamControlBlock, 0),
    normalPriorityQueue:  make([]*streamControlBlock, 0),
    lowPriorityQueue:     make([]*streamControlBlock, 0),
    lowestPriorityQueue:  make([]*streamControlBlock, 0),

    unmanagedStreamQueue: make([]protocol.StreamID, 0),

    streamQueueMap: make(map[string]int),
  }

  // 把已经确定传输顺序的 stream 压入队列中
  for _, url := range sequence.HighestPriorityQueue {
    fts.highestPriorityQueue = append(fts.highestPriorityQueue, newStreamControlBlock(-1, url, false))
    fts.streamQueueMap[url] = 1
  }
  for _, url := range sequence.HighPriorityQueue {
    fts.highPriorityQueue = append(fts.highPriorityQueue, newStreamControlBlock(-1, url, false))
    fts.streamQueueMap[url] = 2
  }
  for _, url := range sequence.NormalPriorityQueue {
    fts.normalPriorityQueue = append(fts.normalPriorityQueue, newStreamControlBlock(-1, url, false))
    fts.streamQueueMap[url] = 3
  }
  for _, url := range sequence.LowPriorityQueue {
    fts.lowPriorityQueue = append(fts.lowPriorityQueue, newStreamControlBlock(-1, url, false))
    fts.streamQueueMap[url] = 4
  }
  for _, url := range sequence.LowestPriorityQueue {
    fts.lowestPriorityQueue = append(fts.lowestPriorityQueue, newStreamControlBlock(0, url, false))
    fts.streamQueueMap[url] = 5
  }
  for _, url := range sequence.BackgroundPriorityQueue {
    fts.backgroundPriorityQueue = append(fts.backgroundPriorityQueue, newStreamControlBlock(0, url, false))
    fts.streamQueueMap[url] = 6
  }

  return &fts
}

// ActiveStreamCount 返回调度器所管理的活跃 stream 数目
func (fts *FileTypeScheduler) ActiveStreamCount() int {
  return fts.highestPriorityQueueActiveCount + fts.highPriorityQueueActiveCount +
    fts.normalPriorityQueueActiveCount + fts.lowPriorityQueueActiveCount +
    fts.lowestPriorityQueueActiveCount + fts.backgroundPriorityQueueActiveCount +
    len(fts.unmanagedStreamQueue)
}

// Empty 返回该调度器中是否有活跃 stream
func (fts *FileTypeScheduler) Empty() bool {
  return fts.ActiveStreamCount() == 0
}

// findAndSetActive 把该队列中具有指定 id 和 url 的 stream 设为指定状态，即活跃和非活跃
func findStreamAndSetStatus(id protocol.StreamID, url string, status bool, queue *[]*streamControlBlock) {
  for _, val := range *queue {
    if val.URL == url {
      val.StreamID = id
      val.Active = status
      return
    }
  }
}

// SetActiveStream 把具有指定 id 和 url 的 stream 设为活跃状态。此 stream 被设为活跃
// 状态之后可被 framer 封装数据
func (fts *FileTypeScheduler) SetActiveStream(id protocol.StreamID, url string) {
  // 针对处于 unmanaged 状态的控制 stream 的处理
  if url == "" {
    // 没有 url 的 stream 为控制 stream，需要以最高优先级传输
    newQueue := make([]*streamControlBlock, 0)
    newQueue = append(newQueue, newStreamControlBlock(id, url, true))
    // 把这些没有 url 的控制 stream 插到最高优先级队列的首位
    fts.highestPriorityQueue = append(newQueue, fts.highestPriorityQueue...)
    fts.highestPriorityQueueActiveCount++
    _, ok := fts.streamQueueMap[url]
    if ok {
      // url 为 "" 的 控制 stream 已存在
      fmt.Printf("error in inserting control stream <%v> into highest queue, error: url exists", id)
      return
    }
    fts.streamQueueMap[url] = 1
    return
  }

  // 针对数据 stream 的处理策略
  queueIndex, ok := fts.streamQueueMap[url]
  if !ok {
    // 此 stream 是一条 unmanaged 的 stream，以最低优先级传输
    fts.streamQueueMap[url] = -1
    fts.unmanagedStreamQueue = append(fts.unmanagedStreamQueue, id)
    return
  }

  // 根据队列标记来在对应的 managed stream 队列中激活此 stream
  switch queueIndex {
  case 1:
    findStreamAndSetStatus(id, url, true, &fts.highestPriorityQueue)
    fts.highestPriorityQueueActiveCount++
    return
  case 2:
    findStreamAndSetStatus(id, url, true, &fts.highPriorityQueue)
    fts.highPriorityQueueActiveCount++
    return
  case 3:
    findStreamAndSetStatus(id, url, true, &fts.normalPriorityQueue)
    fts.normalPriorityQueueActiveCount++
    return
  case 4:
    findStreamAndSetStatus(id, url, true, &fts.lowPriorityQueue)
    fts.lowPriorityQueueActiveCount++
    return
  case 5:
    findStreamAndSetStatus(id, url, true, &fts.lowestPriorityQueue)
    fts.lowestPriorityQueueActiveCount++
    return
  case 6:
    findStreamAndSetStatus(id, url, true, &fts.backgroundPriorityQueue)
    fts.backgroundPriorityQueueActiveCount++
    return
  default:
    fmt.Printf("error in set active stream: stream <%v>, url <%v>: not found " +
      "in any available queue\n", id, url)
    return
  }
}

// SetIdleStream 把具有指定 id 和 url 的 stream 设为空闲状态，调度器不会把此类 stream
// 向 framer 提供
func (fts *FileTypeScheduler) SetIdleStream(id protocol.StreamID, url string) {
  queueIndex, ok := fts.streamQueueMap[url]
  if !ok {
    // 该 stream 并没有在调度器中
    fmt.Printf("warning: disabling non-existent stream, id <%v>, url <%v>", id, url)
    return
  }

  switch queueIndex {
  case 1:
    findStreamAndSetStatus(id, url, false, &fts.highestPriorityQueue)
    fts.highestPriorityQueueActiveCount--
    return
  case 2:
    findStreamAndSetStatus(id, url, false, &fts.highPriorityQueue)
    fts.highPriorityQueueActiveCount--
    return
  case 3:
    findStreamAndSetStatus(id, url, false, &fts.normalPriorityQueue)
    fts.normalPriorityQueueActiveCount--
    return
  case 4:
    findStreamAndSetStatus(id, url, false, &fts.lowPriorityQueue)
    fts.lowPriorityQueueActiveCount--
    return
  case 5:
    findStreamAndSetStatus(id, url, false, &fts.lowestPriorityQueue)
    fts.lowestPriorityQueueActiveCount--
    return
  case 6:
    findStreamAndSetStatus(id, url, false, &fts.backgroundPriorityQueue)
    fts.lowestPriorityQueueActiveCount--
		return
  default:
		// 从 unmanaged stream 队列中移除第一条 stream
		delete(fts.streamQueueMap, url)
    fts.unmanagedStreamQueue = fts.unmanagedStreamQueue[1:]
  }
}

// findFirstActiveStream 查找队列中第一条为 active 状态的 stream
func findFirstActiveStream(queue *[]*streamControlBlock) (protocol.StreamID, bool) {
  for _, val := range *queue {
    if val.Active {
      return val.StreamID, true
    }
  }
  return -1, false
}

// PopNextActiveStream 给出下一条活跃 stream 的 stream ID，由 framer 从此 stream 中
// 封装数据。没有 unmanaged 的 stream 的时候才轮到 managed 的 stream 传输
func (fts *FileTypeScheduler) PopNextActiveStream() protocol.StreamID {
  var id protocol.StreamID
  var ok bool
  if fts.highestPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.highestPriorityQueue)
    if ok {
      return id
    }
    return -1
  }
  if fts.highPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.highPriorityQueue)
    if ok {
      return id
    }
    return -1
  }
  if fts.normalPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.normalPriorityQueue)
    if ok {
      return id
    }
    return -1
  }
  if fts.lowPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.lowPriorityQueue)
    if ok {
      return id
    }
    return -1
  }
  if fts.lowestPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.lowestPriorityQueue)
    if ok {
      return id
    }
    return -1
  }
  if fts.backgroundPriorityQueueActiveCount > 0 {
    id, ok = findFirstActiveStream(&fts.backgroundPriorityQueue)
    if ok {
      return id
    }
    return -1
  }

  // managed stream 队列中没有活跃的 stream，需要在 unmanaged stream 队列中找
  // fmt.Println("no active stream in managed stream queue")

  // unmanaged 的 stream 具有最低优先级
  if len(fts.unmanagedStreamQueue) > 0 {
		result := fts.unmanagedStreamQueue[0]
    return result
  }
  // fmt.Println("PopNextActiveStream: return -1")
  // 当前没有任何活跃 stream
  return -1
}

// findNilStream 从给定的队列中查找具有指定 StreamID 的 stream 的数组下标
func findNilStream(id protocol.StreamID, queue *[]*streamControlBlock) (int, bool) {
  for index, val := range *queue {
    if val.StreamID == id {
      return index, true
    }
  }
  return -1, false
}

// RemoveNilStream 从调度器中移除现值已为 nil 的 stream
func (fts *FileTypeScheduler) RemoveNilStream(id protocol.StreamID) {
  // 由于目前并没有从 id 到 url 的反查 map，使得在只通过 id 移除 stream 时
  // 需要遍历整个调度器
  index, ok := findNilStream(id, &fts.highestPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.highestPriorityQueue[index].URL)
    return
  }
  index, ok = findNilStream(id, &fts.highPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.highPriorityQueue[index].URL)
    return
  }
  index, ok = findNilStream(id, &fts.normalPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.normalPriorityQueue[index].URL)
    return
  }
  index, ok = findNilStream(id, &fts.lowPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.lowPriorityQueue[index].URL)
    return
  }
  index, ok = findNilStream(id, &fts.lowestPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.lowestPriorityQueue[index].URL)
    return
  }
  index, ok = findNilStream(id, &fts.backgroundPriorityQueue)
  if ok {
    fts.SetIdleStream(id, fts.backgroundPriorityQueue[index].URL)
    return
  }

  // 要找的 stream 可能是 unmanaged 的 stream
  if len(fts.unmanagedStreamQueue) <= 0 {
    // unmanaged stream 队列中没有任何数据
    // fmt.Println("empty unmanaged stream queue")
    return
  }
  if id != fts.unmanagedStreamQueue[0] {
    fmt.Printf("error in removing nil stream <%d>, error: stream not found\n", id)
    return
  }
  fts.unmanagedStreamQueue = fts.unmanagedStreamQueue[1:]
}
