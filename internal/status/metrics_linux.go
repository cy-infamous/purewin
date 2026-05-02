//go:build linux

package status

import (
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ─── Metric structs ──────────────────────────────────────────────────────────

// CPUMetrics holds processor utilization data.
type CPUMetrics struct {
	TotalPercent float64
	PerCore      []float64
	CoreCount    int
	ModelName    string
}

// MemoryMetrics holds RAM and swap utilization.
type MemoryMetrics struct {
	Total       uint64
	Used        uint64
	Available   uint64
	Free        uint64
	UsedPercent float64
	SwapTotal   uint64
	SwapUsed    uint64
	SwapPercent float64
}

// DiskMetrics holds partition usage and I/O counters.
type DiskMetrics struct {
	Partitions []DiskPartition
	ReadBytes  uint64
	WriteBytes uint64
}

// DiskPartition is a single mount point.
type DiskPartition struct {
	Path        string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// NetworkMetrics holds aggregate network I/O.
type NetworkMetrics struct {
	BytesSent uint64
	BytesRecv uint64
	SendSpeed uint64 // bytes/sec
	RecvSpeed uint64 // bytes/sec
}

// ProcessInfo describes a single process for the top-N list.
type ProcessInfo struct {
	PID    int32
	Name   string
	CPUPct float64
	MemPct float32
}

// GPUInfo holds basic GPU information.
// On Linux there is no cross-platform WMI equivalent; zero values are returned.
type GPUInfo struct {
	Name       string
	AdapterRAM uint32
}

// BatteryInfo holds battery status (laptops only).
// On Linux this is left empty; desktop machines rarely expose battery via gopsutil.
type BatteryInfo struct {
	HasBattery bool
	Charge     uint16
	IsCharging bool
}

// HardwareInfo holds static machine identification.
type HardwareInfo struct {
	Hostname     string
	OS           string
	OSVersion    string
	CPUModel     string
	CPUCores     int
	RAMTotal     uint64
	Architecture string
}

// SystemMetrics is the aggregate result of a single collection cycle.
type SystemMetrics struct {
	CPU         CPUMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Disk        DiskMetrics    `json:"disk"`
	Network     NetworkMetrics `json:"network"`
	TopProcs    []ProcessInfo  `json:"top_processes"`
	GPU         GPUInfo        `json:"gpu"`
	Battery     BatteryInfo    `json:"battery"`
	Hardware    HardwareInfo   `json:"hardware"`
	CollectedAt time.Time      `json:"collected_at"`
}

// ─── Collection ──────────────────────────────────────────────────────────────

// CollectMetrics gathers all system metrics in parallel.
// prevNet provides the previous network counters for speed calculation;
// interval is the time elapsed since prevNet was recorded.
func CollectMetrics(prevNet *NetworkMetrics, interval time.Duration) (*SystemMetrics, error) {
	m := &SystemMetrics{
		CollectedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// ── CPU ──────────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		total, _ := cpu.Percent(200*time.Millisecond, false)
		perCore, _ := cpu.Percent(200*time.Millisecond, true)
		infos, _ := cpu.Info()

		mu.Lock()
		if len(total) > 0 {
			m.CPU.TotalPercent = total[0]
		}
		m.CPU.PerCore = perCore
		m.CPU.CoreCount = runtime.NumCPU()
		if len(infos) > 0 {
			m.CPU.ModelName = infos[0].ModelName
		}
		mu.Unlock()
	}()

	// ── Memory ───────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		vm, err := mem.VirtualMemory()
		if err != nil {
			return
		}
		swap, _ := mem.SwapMemory()

		mu.Lock()
		m.Memory = MemoryMetrics{
			Total:       vm.Total,
			Used:        vm.Used,
			Available:   vm.Available,
			Free:        vm.Free,
			UsedPercent: vm.UsedPercent,
		}
		if swap != nil {
			m.Memory.SwapTotal = swap.Total
			m.Memory.SwapUsed = swap.Used
			m.Memory.SwapPercent = swap.UsedPercent
		}
		mu.Unlock()
	}()

	// ── Disk ─────────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		parts, err := disk.Partitions(false)
		if err != nil {
			return
		}
		var partitions []DiskPartition
		for _, p := range parts {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			partitions = append(partitions, DiskPartition{
				Path:        p.Mountpoint,
				Total:       usage.Total,
				Used:        usage.Used,
				Free:        usage.Free,
				UsedPercent: usage.UsedPercent,
			})
		}
		ioCounters, _ := disk.IOCounters()
		var readB, writeB uint64
		for _, io := range ioCounters {
			readB += io.ReadBytes
			writeB += io.WriteBytes
		}

		mu.Lock()
		m.Disk = DiskMetrics{
			Partitions: partitions,
			ReadBytes:  readB,
			WriteBytes: writeB,
		}
		mu.Unlock()
	}()

	// ── Network ──────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		counters, err := net.IOCounters(false)
		if err != nil || len(counters) == 0 {
			return
		}

		nm := NetworkMetrics{
			BytesSent: counters[0].BytesSent,
			BytesRecv: counters[0].BytesRecv,
		}
		if prevNet != nil && interval > 0 {
			secs := interval.Seconds()
			if secs > 0 {
				// Calculate speed only if counters didn't wrap/reset.
				// Cap at 10 Gbps (1.25 GB/s) to filter counter resets.
				const maxBytesPerSec uint64 = 10 * 1024 * 1024 * 1024 / 8 // ~1.25 GB/s
				if nm.BytesSent >= prevNet.BytesSent {
					speed := uint64(float64(nm.BytesSent-prevNet.BytesSent) / secs)
					if speed <= maxBytesPerSec {
						nm.SendSpeed = speed
					}
				}
				if nm.BytesRecv >= prevNet.BytesRecv {
					speed := uint64(float64(nm.BytesRecv-prevNet.BytesRecv) / secs)
					if speed <= maxBytesPerSec {
						nm.RecvSpeed = speed
					}
				}
			}
		}

		mu.Lock()
		m.Network = nm
		mu.Unlock()
	}()

	// ── Top processes ────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		procs, err := process.Processes()
		if err != nil {
			return
		}
		var infos []ProcessInfo
		for _, p := range procs {
			name, err := p.Name()
			if err != nil {
				continue
			}
			cpuPct, _ := p.CPUPercent()
			memPct, _ := p.MemoryPercent()
			infos = append(infos, ProcessInfo{
				PID:    p.Pid,
				Name:   name,
				CPUPct: cpuPct,
				MemPct: memPct,
			})
		}
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].CPUPct > infos[j].CPUPct
		})
		if len(infos) > 5 {
			infos = infos[:5]
		}

		mu.Lock()
		m.TopProcs = infos
		mu.Unlock()
	}()

	// ── GPU ──────────────────────────────────────────────────
	// No cross-platform GPU detection on Linux; leave zero-valued.
	// GPU info stays at default (empty) values.

	// ── Battery ──────────────────────────────────────────────
	// gopsutil does not provide a reliable cross-platform battery API.
	// Leave Battery at zero values.

	// ── Hardware info ────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		hw := GetHardwareInfo()
		mu.Lock()
		m.Hardware = hw
		mu.Unlock()
	}()

	wg.Wait()

	return m, nil
}

// ─── Hardware ────────────────────────────────────────────────────────────────

// GetHardwareInfo collects static machine identification data.
func GetHardwareInfo() HardwareInfo {
	info := HardwareInfo{
		Architecture: runtime.GOARCH,
		CPUCores:     runtime.NumCPU(),
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	if hi, err := host.Info(); err == nil {
		info.OS = hi.Platform
		info.OSVersion = hi.PlatformVersion
	}
	if cpus, err := cpu.Info(); err == nil && len(cpus) > 0 {
		info.CPUModel = cpus[0].ModelName
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		info.RAMTotal = vm.Total
	}

	return info
}

// ─── Health score ────────────────────────────────────────────────────────────

// HealthScore computes a 0–100 composite health score.
//
// Deductions:
//
//	CPU  >80 → -30, >60 → -20, >40 → -10
//	Mem  >90 → -25, >75 → -15, >60 → -10
//	Disk >95 → -20, >85 → -15, >75 → -10  (worst partition)
func HealthScore(m *SystemMetrics) int {
	score := 100

	switch {
	case m.CPU.TotalPercent > 80:
		score -= 30
	case m.CPU.TotalPercent > 60:
		score -= 20
	case m.CPU.TotalPercent > 40:
		score -= 10
	}

	switch {
	case m.Memory.UsedPercent > 90:
		score -= 25
	case m.Memory.UsedPercent > 75:
		score -= 15
	case m.Memory.UsedPercent > 60:
		score -= 10
	}

	// Use the worst (highest usage) partition.
	var worstDisk float64
	for _, p := range m.Disk.Partitions {
		if p.UsedPercent > worstDisk {
			worstDisk = p.UsedPercent
		}
	}
	switch {
	case worstDisk > 95:
		score -= 20
	case worstDisk > 85:
		score -= 15
	case worstDisk > 75:
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	return score
}
