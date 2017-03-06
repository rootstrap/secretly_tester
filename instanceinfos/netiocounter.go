package instanceinfos

import (
  "log"
	"github.com/shirou/gopsutil/net"
)

func GetNetIOBytes() (bytesSent, bytesRecv uint64){
  v, err := net.IOCounters(false)
  if err != nil {
    log.Fatal(err)
    return 0, 0
  }
  return v[0].BytesSent, v[0].BytesRecv
}
