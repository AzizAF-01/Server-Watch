package collector

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
	"serverwatch/internal/storage"
)

const metricsCollectionScript = `
echo "===SECTION:CPU==="
cat /proc/stat 2>/dev/null | grep -E '^cpu'
echo "===SECTION:LOAD==="
cat /proc/loadavg 2>/dev/null
echo "===SECTION:MEM==="
cat /proc/meminfo 2>/dev/null
echo "===SECTION:DISK==="
df -P -k 2>/dev/null
echo "===SECTION:NET==="
cat /proc/net/dev 2>/dev/null
echo "===SECTION:UPTIME==="
cat /proc/uptime 2>/dev/null
echo "===SECTION:OS==="
cat /etc/os-release 2>/dev/null | grep -E '^(PRETTY_NAME)=' | head -1
uname -srm 2>/dev/null
echo "===SECTION:PROC==="
ps -eo pid,user,%cpu,%mem,comm 2>/dev/null | sort -nrk 3 | head -n 6
`

type cpuTickSample struct {
	total uint64
	idle  uint64
}

type netTickSample struct {
	rxBytes uint64
	txBytes uint64
	time    time.Time
}

type prevSample struct {
	cpuTotal     cpuTickSample
	cpuCores     []cpuTickSample
	netDevs      map[string]netTickSample
	sampleTime   time.Time
}

type Collector struct {
	sshPool           *sshpool.SSHPool
	storage           *storage.SQLiteStorage
	mu                sync.Mutex
	prevData          map[string]*prevSample
	inFlight          sync.Map
	onAlertTransition func(serverName, level, message string, isRecovery bool)
}

func NewCollector(pool *sshpool.SSHPool, store *storage.SQLiteStorage) *Collector {
	return &Collector{
		sshPool:  pool,
		storage:  store,
		prevData: make(map[string]*prevSample),
	}
}

func (c *Collector) SetAlertCallback(fn func(serverName, level, message string, isRecovery bool)) {
	c.onAlertTransition = fn
}

func (c *Collector) notifyTransition(serverName, level, message string, isRecovery bool) {
	if c.onAlertTransition != nil {
		c.onAlertTransition(serverName, level, message, isRecovery)
	}
}

// CollectServerMetrics fetches metrics over SSH and computes rates
func (c *Collector) CollectServerMetrics(ctx context.Context, srv *model.Server) (*model.ServerMetrics, error) {
	if _, loaded := c.inFlight.LoadOrStore(srv.ID, true); loaded {
		return nil, nil
	}
	defer c.inFlight.Delete(srv.ID)

	ctxTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	output, err := c.sshPool.ExecuteCommand(ctxTimeout, srv, metricsCollectionScript)
	if err != nil {
		_ = c.storage.UpdateServerStatus(srv.ID, model.StatusUnreachable, err.Error())
		msg := "Koneksi SSH gagal: " + err.Error()
		if triggered, _ := c.storage.RecordAlertEventTransition(srv.ID, srv.Name, "rule_ssh", "critical", msg); triggered {
			c.notifyTransition(srv.Name, "critical", msg, false)
		}
		return nil, err
	}
	if resolved, _ := c.storage.ResolveAlertEventTransition(srv.ID, "rule_ssh"); resolved {
		c.notifyTransition(srv.Name, "ok", "Koneksi SSH kembali pulih", true)
	}

	metrics := c.parseOutput(srv.ID, output)
	_ = c.storage.SaveMetrics(metrics)

	// Determine status (Normal / Warning / Critical) and evaluate alert rules
	status := model.StatusNormal
	reason := ""

	maxDisk := 0.0
	criticalMount := "/"
	var freeDiskBytes uint64
	for _, d := range metrics.Disks {
		if d.UsedPercent > maxDisk {
			maxDisk = d.UsedPercent
			criticalMount = d.MountPoint
			freeDiskBytes = d.FreeBytes
		}
	}

	if maxDisk >= 90.0 {
		status = model.StatusCritical
		freeGB := float64(freeDiskBytes) / (1024 * 1024 * 1024)
		reason = fmt.Sprintf("Partisi disk '%s' kritis: %.1f%% terpakai (sisa %.1f GB)", criticalMount, maxDisk, freeGB)
		if triggered, _ := c.storage.RecordAlertEventTransition(srv.ID, srv.Name, "rule_disk", "critical", reason); triggered {
			c.notifyTransition(srv.Name, "critical", reason, false)
		}
	} else {
		if resolved, _ := c.storage.ResolveAlertEventTransition(srv.ID, "rule_disk"); resolved {
			c.notifyTransition(srv.Name, "ok", fmt.Sprintf("Kapasitas disk '%s' kembali normal (%.1f%% terpakai)", criticalMount, maxDisk), true)
		}
	}

	if metrics.CPU.TotalPercent >= 90.0 {
		msg := fmt.Sprintf("Beban CPU tinggi: %.1f%% (ambang batas 90.0%%)", metrics.CPU.TotalPercent)
		if status == model.StatusNormal {
			status = model.StatusWarning
			reason = msg
		}
		if triggered, _ := c.storage.RecordAlertEventTransition(srv.ID, srv.Name, "rule_cpu", "warning", msg); triggered {
			c.notifyTransition(srv.Name, "warning", msg, false)
		}
	} else {
		if resolved, _ := c.storage.ResolveAlertEventTransition(srv.ID, "rule_cpu"); resolved {
			c.notifyTransition(srv.Name, "ok", fmt.Sprintf("Beban CPU kembali normal: %.1f%%", metrics.CPU.TotalPercent), true)
		}
	}

	if metrics.Memory.UsedPercent >= 90.0 {
		freeRAMMB := float64(metrics.Memory.AvailableBytes) / (1024 * 1024)
		if freeRAMMB == 0 && metrics.Memory.FreeBytes > 0 {
			freeRAMMB = float64(metrics.Memory.FreeBytes) / (1024 * 1024)
		}
		msg := fmt.Sprintf("Konsumsi RAM tinggi: %.1f%% (sisa %.0f MB)", metrics.Memory.UsedPercent, freeRAMMB)
		if status == model.StatusNormal {
			status = model.StatusWarning
			reason = msg
		}
		if triggered, _ := c.storage.RecordAlertEventTransition(srv.ID, srv.Name, "rule_mem", "warning", msg); triggered {
			c.notifyTransition(srv.Name, "warning", msg, false)
		}
	} else {
		if resolved, _ := c.storage.ResolveAlertEventTransition(srv.ID, "rule_mem"); resolved {
			c.notifyTransition(srv.Name, "ok", fmt.Sprintf("Penggunaan RAM kembali normal: %.1f%%", metrics.Memory.UsedPercent), true)
		}
	}

	_ = c.storage.UpdateServerStatus(srv.ID, status, reason)
	return metrics, nil
}

func (c *Collector) parseOutput(serverID string, raw string) *model.ServerMetrics {
	now := time.Now()
	metrics := &model.ServerMetrics{
		ServerID:  serverID,
		Timestamp: now,
		CPU:       model.CPUStats{},
		Memory:    model.MemStats{},
	}

	sections := strings.Split(raw, "===SECTION:")
	var rawCPU, rawLoad, rawMem, rawDisk, rawNet, rawUptime, rawOS, rawProc string

	for _, sec := range sections {
		if strings.HasPrefix(sec, "CPU===") {
			rawCPU = strings.TrimPrefix(sec, "CPU===\n")
		} else if strings.HasPrefix(sec, "LOAD===") {
			rawLoad = strings.TrimPrefix(sec, "LOAD===\n")
		} else if strings.HasPrefix(sec, "MEM===") {
			rawMem = strings.TrimPrefix(sec, "MEM===\n")
		} else if strings.HasPrefix(sec, "DISK===") {
			rawDisk = strings.TrimPrefix(sec, "DISK===\n")
		} else if strings.HasPrefix(sec, "NET===") {
			rawNet = strings.TrimPrefix(sec, "NET===\n")
		} else if strings.HasPrefix(sec, "UPTIME===") {
			rawUptime = strings.TrimPrefix(sec, "UPTIME===\n")
		} else if strings.HasPrefix(sec, "OS===") {
			rawOS = strings.TrimPrefix(sec, "OS===\n")
		} else if strings.HasPrefix(sec, "PROC===") {
			rawProc = strings.TrimPrefix(sec, "PROC===\n")
		}
	}

	uptimeFields := strings.Fields(rawUptime)
	if len(uptimeFields) > 0 {
		if upSec, err := strconv.ParseFloat(uptimeFields[0], 64); err == nil {
			metrics.UptimeSeconds = uint64(upSec)
		}
	}

	loadFields := strings.Fields(rawLoad)
	if len(loadFields) >= 3 {
		metrics.CPU.Load1, _ = strconv.ParseFloat(loadFields[0], 64)
		metrics.CPU.Load5, _ = strconv.ParseFloat(loadFields[1], 64)
		metrics.CPU.Load15, _ = strconv.ParseFloat(loadFields[2], 64)
	}

	currCPUTotal, currCPUCores := parseCPULines(rawCPU)
	c.mu.Lock()
	prev, hasPrev := c.prevData[serverID]
	if hasPrev && prev != nil {
		metrics.CPU.TotalPercent = calcCPUUsage(prev.cpuTotal, currCPUTotal)
		for i := range currCPUCores {
			if i < len(prev.cpuCores) {
				corePct := calcCPUUsage(prev.cpuCores[i], currCPUCores[i])
				metrics.CPU.Cores = append(metrics.CPU.Cores, corePct)
			}
		}
	} else {
		// First poll: provide an estimated load from loadavg or core count
		if metrics.CPU.Load1 > 0 {
			metrics.CPU.TotalPercent = math.Min(100.0, metrics.CPU.Load1*15.0)
		}
	}
	c.mu.Unlock()

	metrics.Memory = parseMemInfo(rawMem)
	metrics.Disks = parseDiskUsage(rawDisk)

	currNetDevs, netInterfaces := parseNetDev(rawNet)
	if hasPrev && prev != nil && prev.netDevs != nil {
		elapsedSec := now.Sub(prev.sampleTime).Seconds()
		if elapsedSec > 0 {
			for i := range netInterfaces {
				name := netInterfaces[i].Name
				if pNet, exists := prev.netDevs[name]; exists {
					if netInterfaces[i].TotalRxBytes >= pNet.rxBytes {
						netInterfaces[i].RxBytesSec = float64(netInterfaces[i].TotalRxBytes-pNet.rxBytes) / elapsedSec
					}
					if netInterfaces[i].TotalTxBytes >= pNet.txBytes {
						netInterfaces[i].TxBytesSec = float64(netInterfaces[i].TotalTxBytes-pNet.txBytes) / elapsedSec
					}
				}
			}
		}
	}
	metrics.Networks = netInterfaces

	osLines := strings.Split(strings.TrimSpace(rawOS), "\n")
	for _, l := range osLines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			metrics.OSInfo = strings.Trim(strings.TrimPrefix(l, "PRETTY_NAME="), `"`)
		} else if strings.Contains(l, "Linux") {
			metrics.KernelInfo = l
		}
	}

	metrics.Processes = parseProcesses(rawProc)

	// Update previous sample
	c.mu.Lock()
	c.prevData[serverID] = &prevSample{
		cpuTotal:   currCPUTotal,
		cpuCores:   currCPUCores,
		netDevs:    currNetDevs,
		sampleTime: now,
	}
	c.mu.Unlock()

	return metrics
}

func parseCPULines(raw string) (cpuTickSample, []cpuTickSample) {
	var total cpuTickSample
	var cores []cpuTickSample

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var sum, idle uint64
		for i := 1; i < len(fields); i++ {
			val, _ := strconv.ParseUint(fields[i], 10, 64)
			sum += val
			if i == 4 || i == 5 { // idle + iowait
				idle += val
			}
		}
		sample := cpuTickSample{total: sum, idle: idle}
		if fields[0] == "cpu" {
			total = sample
		} else if strings.HasPrefix(fields[0], "cpu") {
			cores = append(cores, sample)
		}
	}
	return total, cores
}

func calcCPUUsage(prev, curr cpuTickSample) float64 {
	if curr.total <= prev.total {
		return 0.0
	}
	totalDiff := float64(curr.total - prev.total)
	idleDiff := float64(curr.idle - prev.idle)
	usage := 100.0 * (totalDiff - idleDiff) / totalDiff
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return math.Round(usage*10) / 10
}

func parseMemInfo(raw string) model.MemStats {
	var mem model.MemStats
	lines := strings.Split(raw, "\n")
	var memTotal, memFree, memAvail, buffers, cached, swapTotal, swapFree uint64

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valFields := strings.Fields(parts[1])
		if len(valFields) == 0 {
			continue
		}
		kb, _ := strconv.ParseUint(valFields[0], 10, 64)
		bytes := kb * 1024

		switch key {
		case "MemTotal":
			memTotal = bytes
		case "MemFree":
			memFree = bytes
		case "MemAvailable":
			memAvail = bytes
		case "Buffers":
			buffers = bytes
		case "Cached":
			cached = bytes
		case "SwapTotal":
			swapTotal = bytes
		case "SwapFree":
			swapFree = bytes
		}
	}

	mem.TotalBytes = memTotal
	mem.FreeBytes = memFree
	mem.CachedBytes = cached + buffers
	mem.AvailableBytes = memAvail
	if memAvail > 0 && memTotal >= memAvail {
		mem.UsedBytes = memTotal - memAvail
	} else if memTotal >= (memFree + buffers + cached) {
		mem.UsedBytes = memTotal - (memFree + buffers + cached)
	}

	if memTotal > 0 {
		mem.UsedPercent = math.Round((float64(mem.UsedBytes)/float64(memTotal))*1000) / 10
	}

	mem.SwapTotalBytes = swapTotal
	if swapTotal >= swapFree {
		mem.SwapUsedBytes = swapTotal - swapFree
		if swapTotal > 0 {
			mem.SwapUsedPercent = math.Round((float64(mem.SwapUsedBytes)/float64(swapTotal))*1000) / 10
		}
	}

	return mem
}

func parseDiskUsage(raw string) []model.DiskPartition {
	var disks []model.DiskPartition
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" { // skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		device := fields[0]
		// Skip tmpfs, udev, loop devices for cleanliness unless it's root
		if strings.HasPrefix(device, "tmpfs") || strings.HasPrefix(device, "udev") || strings.HasPrefix(device, "/dev/loop") {
			continue
		}

		totalKB, _ := strconv.ParseUint(fields[1], 10, 64)
		usedKB, _ := strconv.ParseUint(fields[2], 10, 64)
		freeKB, _ := strconv.ParseUint(fields[3], 10, 64)
		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.ParseFloat(pctStr, 64)
		mount := fields[5]

		disks = append(disks, model.DiskPartition{
			MountPoint:  mount,
			Device:      device,
			TotalBytes:  totalKB * 1024,
			UsedBytes:   usedKB * 1024,
			FreeBytes:   freeKB * 1024,
			UsedPercent: pct,
		})
	}
	return disks
}

func parseNetDev(raw string) (map[string]netTickSample, []model.NetInterface) {
	currMap := make(map[string]netTickSample)
	var ifaces []model.NetInterface

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "lo" { // ignore loopback
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)

		currMap[name] = netTickSample{rxBytes: rx, txBytes: tx, time: time.Now()}
		ifaces = append(ifaces, model.NetInterface{
			Name:         name,
			TotalRxBytes: rx,
			TotalTxBytes: tx,
		})
	}
	return currMap, ifaces
}

func parseProcesses(raw string) []model.ProcessItem {
	var items []model.ProcessItem
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		user := fields[1]
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		comm := strings.Join(fields[4:], " ")

		items = append(items, model.ProcessItem{
			PID:        pid,
			User:       user,
			CPUPercent: cpu,
			MemPercent: mem,
			Command:    comm,
		})
	}
	return items
}
