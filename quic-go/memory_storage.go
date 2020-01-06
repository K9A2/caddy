package quic

import (
  "io/ioutil"
  "os"

  "github.com/google/logger"
)

// MemoryStorage 在内存中存放所有静态文件的控制字段和数据字段
type MemoryStorage struct {
  resourceMap map[string]*StreamControlBlock
}
// 全局唯一的实例
var storage MemoryStorage

// InitMemoryStorage 负责在内存中加载指定的资源文件
func InitMemoryStorage() {
  // 加载配置文件
  sequenceFile := LoadConfig()
  if sequenceFile == nil {
    logger.Error("error in loading sequence file")
    return
  }
  // 把所有 managed 的资源 url 全部读出，加载时要去除路径首位的 "/"
  filesToLoad := make([]string, 0)
  var url string
  for _, val := range sequenceFile.HighestPriorityQueue {
    if val == "/" {
      url = "index.html"
    } else {
      url = val[1:]
    }
    filesToLoad = append(filesToLoad, url)
  }
  for _, val := range sequenceFile.HighPriorityQueue {
    filesToLoad = append(filesToLoad, val[1:])
  }
  for _, val := range sequenceFile.NormalPriorityQueue {
    filesToLoad = append(filesToLoad, val[1:])
  }
  for _, val := range sequenceFile.LowPriorityQueue {
    filesToLoad = append(filesToLoad, val[1:])
  }
  for _, val := range sequenceFile.LowestPriorityQueue {
    filesToLoad = append(filesToLoad, val[1:])
  }
  for _, val := range sequenceFile.BackgroundPriorityQueue {
    filesToLoad = append(filesToLoad, val[1:])
  }

  // 初始化全局变量
  storage = MemoryStorage{
    resourceMap: make(map[string]*StreamControlBlock),
  }

  // 读取所有文件的内容
  for _, url := range filesToLoad {
    target, err := os.Open(url)
    if err != nil {
      logger.Errorf("error in opening file, file <%v>, err <%v>\n", url, err.Error())
      return
    }
    info, err := target.Stat()
    if err != nil {
      logger.Errorf("error in accessing file info, file <%v>, err <%v>\n", url, err.Error())
      return
    }
    expectedLen := info.Size()

    content, err := ioutil.ReadFile(url)
    if err != nil {
      logger.Errorf("error in loading static file <%s> into memory storage, err: %s\n", url, err.Error())
      return
    }
    actualLen := len(content)

    if int(expectedLen) != actualLen {
      logger.Errorf("file <%v> len error, expected len <%v>, actual len = <%v>\n", url, expectedLen, actualLen)
      return
    }
    
    storage.Set(url, NewStreamControlBlock(-1, url, false, &content, false))
    logger.Infof("key = <%v>, len = <%v>", url, len(content))
  }
  logger.Infof("memory storage inited")
}

// GetMemoryStorage 返回指向 memory storage 全局变量的指针
func GetMemoryStorage() *MemoryStorage {
  return &storage
}

// Contains 检查给出的 url 是否有收录在 MemoryStorage 中
func (ms *MemoryStorage) Contains(url string) bool {
  _, ok := ms.resourceMap[url]
  return ok
}

// GetByURL 以 url 为 key 查找资源对应的 stream control block
func (ms *MemoryStorage) GetByURL(url string) *StreamControlBlock {
  val, ok := ms.resourceMap[url]
  if !ok {
    logger.Errorf("error: loading non-existent value by key <%v>\n", url)
    return nil
  }
  logger.Infof("cache hit, key = <%v>, len = <%v>", url, len(*val.Data))
  return val
}

// Set 向 MemoryBuffer 中添加具有指定 key 和 content 的资源
func (ms *MemoryStorage) Set(url string, block *StreamControlBlock) {
  val, ok := ms.resourceMap[url]
  if ok {
    logger.Warningf("warning: overriding existing resource, new val len <%v>, "+
      "original val len <%v>\n", len(*val.Data), len(*val.Data))
  }
  ms.resourceMap[url] = block
}

// MarkAsReplaced 把 url 指定的资源的替换标志位设为 true
func (ms *MemoryStorage) MarkAsReplaced(url string) bool {
	val, ok := ms.resourceMap[url]
	if ok {
		val.Replaced = true
		logger.Infof("replace bit for url <%v> set to true", url)
		return ok
	}
	return ok
}

