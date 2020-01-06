package quic

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"
)

const configFileName = "static-file-type-sequence.json"

// SequenceFile 是 JSON 格式的传输顺序文件
type SequenceFile struct {
  HighestPriorityQueue    []string `json:"highest"`
  HighPriorityQueue       []string `json:"high"`
  NormalPriorityQueue     []string `json:"normal"`
  LowPriorityQueue        []string `json:"low"`
  LowestPriorityQueue     []string `json:"lowest"`
  BackgroundPriorityQueue []string `json:"background"`
}

// LoadConfig 读取文件传输顺序，成功读取时会返回解析后的文件顺序数据
func LoadConfig() *SequenceFile {
  jsonFile, err := os.Open(configFileName)
  if err != nil {
    fmt.Printf("error in loading file sequence, err = <%s>\n", err.Error())
    return nil
  }
  defer jsonFile.Close()

  byteValue, _ := ioutil.ReadAll(jsonFile)
  var sequence SequenceFile
  json.Unmarshal(byteValue, &sequence)
  return &sequence
}
