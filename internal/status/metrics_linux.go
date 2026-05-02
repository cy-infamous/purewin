//go:build linux

package status

import (
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

// GPUInfo holds GPU information detected via nvidia-smi or lspci.
type GPUInfo struct {
	Name        string
	MemoryTotal uint64
	MemoryUsed  uint64
	Utilization float64
	Temperature float64
	PowerDraw   float64
	PowerLimit  float64
	AdapterRAM  uint32
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		gpu := detectGPU()
		mu.Lock()
		m.GPU = gpu
		mu.Unlock()
	}()

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

// ─── GPU Detection ──────────────────────────────────────────────────────────

func detectGPU() GPUInfo {
	if gpu := detectNvidiaGPU(); gpu.Name != "" {
		return gpu
	}
	if gpu := detectLspciGPU(); gpu.Name != "" {
		return gpu
	}
	return detectSysDRMGPU()
}

func detectNvidiaGPU() GPUInfo {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return GPUInfo{}
	}

	out, err := exec.Command(path,
		"--query-gpu=name,memory.total,memory.used,utilization.gpu,temperature.gpu,power.draw,power.limit",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return GPUInfo{}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return GPUInfo{}
	}

	fields := strings.Split(lines[0], ",")
	if len(fields) < 4 {
		return GPUInfo{}
	}

	gpu := GPUInfo{
		Name: strings.TrimSpace(fields[0]),
	}

	if v, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64); err == nil {
		gpu.MemoryTotal = v * 1024 * 1024
	}
	if v, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64); err == nil {
		gpu.MemoryUsed = v * 1024 * 1024
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		gpu.Utilization = v
	}
	if len(fields) > 4 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
			gpu.Temperature = v
		}
	}
	if len(fields) > 5 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64); err == nil {
			gpu.PowerDraw = v
		}
	}
	if len(fields) > 6 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64); err == nil {
			gpu.PowerLimit = v
		}
	}

	return gpu
}

func detectLspciGPU() GPUInfo {
	path, err := exec.LookPath("lspci")
	if err != nil {
		return GPUInfo{}
	}

	out, err := exec.Command(path).Output()
	if err != nil {
		return GPUInfo{}
	}

	var best string
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "vga") && !strings.Contains(lower, "3d") && !strings.Contains(lower, "display") {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[idx+2:])

		if strings.Contains(strings.ToLower(name), "nvidia") {
			best = name
			break
		}
		if strings.Contains(strings.ToLower(name), "amd") || strings.Contains(strings.ToLower(name), "radeon") {
			if best == "" {
				best = name
			}
			continue
		}
		if best == "" {
			best = name
		}
	}

	if best != "" {
		return GPUInfo{Name: best}
	}
	return GPUInfo{}
}

func detectSysDRMGPU() GPUInfo {
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return GPUInfo{}
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
			continue
		}

		cardPath := "/sys/class/drm/" + name

		vendorPath := cardPath + "/device/vendor"
		data, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}
		vendor := strings.TrimSpace(string(data))

		var driverName string
		if link, err := os.Readlink(cardPath + "/device/driver"); err == nil {
			if idx := strings.LastIndex(link, "/"); idx >= 0 {
				driverName = link[idx+1:]
			}
		}

		gpuName := ""
		switch {
		case strings.Contains(vendor, "10de"):
			gpuName = "NVIDIA GPU"
		case strings.Contains(vendor, "1002"):
			gpuName = "AMD GPU"
		case strings.Contains(vendor, "8086"):
			ueventData, _ := os.ReadFile(cardPath + "/device/uevent")
			for _, line := range strings.Split(string(ueventData), "\n") {
				if strings.HasPrefix(line, "PCI_ID=") {
					parts := strings.Split(strings.TrimPrefix(line, "PCI_ID="), ":")
					if len(parts) == 2 {
						gpuName = "Intel Graphics"
						_ = parts[1]
					}
					break
				}
			}
			if gpuName == "" {
				gpuName = "Intel GPU"
			}
		default:
			if driverName != "" {
				gpuName = driverName + " GPU"
			} else {
				continue
			}
		}

		return GPUInfo{Name: gpuName}
	}

	return GPUInfo{}
}
