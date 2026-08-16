package system

import (
	"math"

	"fmt"

	"github.com/shirou/gopsutil/host"
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

// Calculate the system's uptime in days, hours, minutes, and seconds
func CalculateUptime(bootTime uint64) string {
	uptime, err := host.Uptime()
	if err != nil {
		return "0 days, 0 hours, 0 minutes, 0 seconds"
	}
	totalSeconds := uptime - bootTime

	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	// Convert to a string representation
	return fmt.Sprintf("%dD, %dH, %dM, %dS", days, hours, minutes, seconds)
}
