package worker

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

type Stats struct {
	CpuStat   *cpu.TimesStat
	MemInfo   *mem.VirtualMemoryStat
	Disk      *disk.UsageStat
	LoadAvg   *load.AvgStat
	TaskCount int
}

func GetStats() *Stats {
	cpuTimeStat, _ := cpu.Times(false)
	memStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")
	loadAvg, _ := load.Avg()

	return &Stats{
		CpuStat: &cpuTimeStat[0],
		MemInfo: memStat,
		Disk:    diskStat,
		LoadAvg: loadAvg,
	}
}
