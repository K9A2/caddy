package quic

import (
  "fmt"
  "mime"
  "path/filepath"
	"strings"

  "github.com/caddyserver/caddy/v2/quic-go/core/protocol"
)

type scheduler interface {
	// 调度器所管理的活跃 stream 数目
	ActiveStreamCount() int
	// 调度器中是否有活跃 stream
	Empty() bool

  // 把 id 对应的 stream 设为活跃状态，调度器可以让 framer 从此 stream 中提取数据
	SetActiveStream(id protocol.StreamID, url string)
	// 把 id 对应的 stream 设为空闲装填，调度器不可以让 framer 从此 stream 中提取数据
	SetIdleStream(id protocol.StreamID, url string)
	// 从调度器中移除空 stream
	RemoveNilStream(id protocol.StreamID)

	// 从调度器中获取一条活跃 stream
	PopNextActiveStream() protocol.StreamID
}

type streamControlBlock struct {
  StreamID protocol.StreamID
  URL      string
  Active   bool
}

func newStreamControlBlock(id protocol.StreamID, url string, active bool) *streamControlBlock {
  return &streamControlBlock{
    StreamID: id,
    URL:      url,
    Active:   active,
  }
}

// GetMimeType 根据资源路径返回对应的 MIME 类型
func GetMimeType(fileName string) string {
  // 获取资源的扩展名
  fileExt := filepath.Ext(fileName)
  var mtype string

  // 以下扩展名的资源无法从系统库中获取 MIME 类型，需要在本函数中指定 MIME 类型
  switch fileExt {
  case "":
    mtype = "text/html"
  case ".woff2":
    mtype = "font"
  }
  if mtype != "" {
    return mtype
  }

  // 不是已知的无法从系统库中获取到 MIME 类型的资源，则需要尝试从系统库中获取资源
  mtype = mime.TypeByExtension(filepath.Ext(fileName))
  if mtype == "" {
    // 无法从系统库中查找到该资源对应的 MIME 类型
    mtype = "unknown"
    fmt.Printf("warning: unknown mimetype for url = %v\n", fileName)
  } else {
    // 针对可以从系统库中获取到 MIME 类型的资源，检查系统库给的 MIME 类型中是否包含
    // 类似于 “text/css; charset=utf-8” 的字符集信息
    semicolonIndex := strings.Index(mtype, ";")
    if semicolonIndex > 0 {
      // 移除字符集信息
      mtype = mtype[:semicolonIndex]
    }
  }
  return mtype
}
