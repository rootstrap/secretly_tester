package instanceinfos

import (
  "log"
	"github.com/shirou/gopsutil/cpu"
)

func GetCPUUsage() (float64){
  v, err := cpu.Percent(0, false)
  if err != nil {
    log.Fatal(err)
    return 0.0
  }
  return v[0]
}
