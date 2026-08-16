package system

import (
	"math"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// Get the system's CPU, memory usage, and storage usage percentages
func GetSystemUsage() (float64, float64, float64, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return 0, 0, 0, err
	}
	memoryPercent, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0, err
	}
	storagePercent, err := disk.Usage("/")
	if err != nil {
		return 0, 0, 0, err
	}
	return math.Round(cpuPercent[0]*100) / 100, math.Round(memoryPercent.UsedPercent*100) / 100, math.Round(storagePercent.UsedPercent*100) / 100, nil
}
