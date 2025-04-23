package stats

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"time"
)

type Stats struct {
	CpuPercent float64 //`json:"cpuPercent"`
	CpuStat    *cpu.TimesStat
	MemStat    *mem.VirtualMemoryStat
	DiskStat   *disk.UsageStat
	LoadAvg    *load.AvgStat
	TaskCount  int
}

func GetStats() *Stats {
	CpuPercent, _ := cpu.Percent(time.Second, false)
	cpuTimeStat, _ := cpu.Times(false)
	memStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")
	loadAvg, _ := load.Avg()

	return &Stats{
		CpuPercent: CpuPercent[0],
		CpuStat:    &cpuTimeStat[0],
		MemStat:    memStat,
		DiskStat:   diskStat,
		LoadAvg:    loadAvg,
	}
}
