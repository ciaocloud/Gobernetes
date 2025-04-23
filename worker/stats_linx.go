package worker

import (
	"github.com/c9s/goprocinfo/linux"
	"log"
)

type LinuxStats struct {
	CpuStat   *linux.CPUStat
	MemInfo   *linux.MemInfo
	Disk      *linux.Disk
	LoadAvg   *linux.LoadAvg
	TaskCount int
}

func (s *LinuxStats) MemUsedKb() uint64 {
	return s.MemInfo.MemTotal - s.MemInfo.MemAvailable
}

func (s *LinuxStats) MemAvailableKb() uint64 {
	return s.MemInfo.MemAvailable
}

func (s *LinuxStats) MemTotalKb() uint64 {
	return s.MemInfo.MemTotal
}

func (s *LinuxStats) MemUsedPercent() float64 {
	return float64(s.MemUsedKb()) / float64(s.MemTotalKb())
}

func (s *LinuxStats) DiskTotal() uint64 {
	return s.Disk.All
}

func (s *LinuxStats) DiskUsed() uint64 {
	return s.Disk.Used
}

func (s *LinuxStats) DiskFree() uint64 {
	return s.Disk.Free
}

func (s *LinuxStats) CpuUsage() float64 {
	idle := s.CpuStat.Idle + s.CpuStat.IOWait
	nonIdle := s.CpuStat.User + s.CpuStat.Nice + s.CpuStat.System + s.CpuStat.IRQ + s.CpuStat.SoftIRQ + s.CpuStat.Steal
	tot := idle + nonIdle
	if tot == 0 {
		return float64(0)
	}
	return float64(nonIdle) / float64(tot)
}

func GetLinuxStats() *LinuxStats {
	return &LinuxStats{
		CpuStat: GetCpuStats(),
		MemInfo: GetMemInfo(),
		Disk:    GetDiskInfo(),
		LoadAvg: GetLoadAvg(),
	}
}

func GetCpuStats() *linux.CPUStat {
	stat, err := linux.ReadStat("/proc/stat")
	//cpuStat, err := linux.ReadCPUStat("/proc/stat")
	if err != nil {
		log.Printf("Error reading from /proc/stat: %v\n", err)
		return &linux.CPUStat{}
	}
	return &stat.CPUStatAll
}

func GetMemInfo() *linux.MemInfo {
	memInfo, err := linux.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Printf("Error reading from /proc/meminfo: %v\n", err)
		return &linux.MemInfo{}
	}
	return memInfo
}

func GetDiskInfo() *linux.Disk {
	diskInfo, err := linux.ReadDisk("/")
	if err != nil {
		log.Printf("Error reading from /: %v\n", err)
		return &linux.Disk{}
	}
	return diskInfo
}

func GetLoadAvg() *linux.LoadAvg {
	loadAvg, err := linux.ReadLoadAvg("/proc/loadavg")
	if err != nil {
		log.Printf("Error reading from /proc/loadavg: %v\n", err)
		return &linux.LoadAvg{}
	}
	return loadAvg
}
