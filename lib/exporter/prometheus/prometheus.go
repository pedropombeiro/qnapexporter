// Package prometheus implements an Exporter that collects QNAP NAS metrics and
// writes them in the Prometheus text exposition format.
package prometheus

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pedropombeiro/qnapexporter/lib/exporter"
	"github.com/pedropombeiro/qnapexporter/lib/utils"
)

const (
	devDir                     = "/dev"
	netDir                     = "/sys/class/net"
	flashcacheStatsPath        = "/proc/flashcache/CG0/flashcache_stats"
	dmCacheStatsFilePathFormat = "/sys/block/%s/dm/cache/curr_stats"

	envValidity    = time.Duration(5 * time.Minute)
	volumeValidity = time.Duration(1 * time.Minute)
)

type fetchMetricFn func() ([]metric, error)

type qnapEnclosure struct {
	id        string
	name      string
	diskCount int
	fanCount  int
	tempCount int
}

type promExporter struct {
	ExporterConfig

	status *exporter.Status

	hostname      string
	kernelVersion int

	upsState upsState

	getsysinfo  string
	syshdnum    int
	sysfannum   int
	ifaces      []string
	devices     []string
	nvmePath    string
	nvmeDevices []string
	halApp      string
	enclosures  []qnapEnclosure
	envExpiry   time.Time

	volumes         []volumeInfo
	volumeLastFetch time.Time

	dmCacheClients           []string
	dmCacheDeviceMinorNumber string

	fns     map[string]fetchMetricFn
	fetchMu sync.Mutex
}

// ExporterConfig holds the configuration options for the Prometheus exporter.
type ExporterConfig struct {
	PingTarget string
	Logger     *log.Logger
}

// NewExporter creates a Prometheus exporter using the given configuration and
// optional shared status, registering all supported metric collectors.
func NewExporter(config ExporterConfig, status *exporter.Status) exporter.Exporter {
	now := time.Now()
	e := &promExporter{
		ExporterConfig: config,
		status:         status,
		envExpiry:      now,
	}
	e.fns = map[string]fetchMetricFn{
		"version":         e.getVersionMetrics,
		"uptime":          getUptimeMetrics,
		"loadAvg":         getLoadAvgMetrics,
		"CpuRatio":        getCPURatioMetrics,
		"MemInfo":         getMemInfoMetrics,
		"UpsStats":        e.getUpsStatsMetricsWithRetry,
		"SysInfoTemp":     e.getSysInfoTempMetrics,
		"SysInfoFan":      e.getSysInfoFanMetrics,
		"EnclosureFan":    e.getEnclosureFanMetrics,
		"SysInfoHd":       e.getSysInfoHdMetrics,
		"SysInfoVol":      e.getSysInfoVolMetrics,
		"DiskStats":       e.getDiskStatsMetrics,
		"FlashCacheStats": e.getFlashCacheStatsMetrics,
		"DmCacheStats":    e.getDmCacheStatsMetrics,
		"NetworkStats":    e.getNetworkStatsMetrics,
		"Ping":            e.getPingMetrics,
		"NvmeSmart":       e.getNvmeSmartMetrics,
	}

	if status != nil {
		status.Uptime = now
	}

	return e
}

func (e *promExporter) WriteMetrics(w io.Writer) error {
	e.fetchMu.Lock()
	defer e.fetchMu.Unlock()

	if e.status != nil {
		e.status.MetricCount = 0
		e.status.LastFetch = time.Now()
		defer func() {
			e.status.LastFetchDuration = time.Since(e.status.LastFetch)
		}()
	}

	if time.Now().After(e.envExpiry) {
		e.readEnvironment()
	}

	var wg sync.WaitGroup
	metricsCh := make(chan interface{}, 4)
	for name, fn := range e.fns {
		wg.Add(1)

		go fetchMetricsWorker(&wg, metricsCh, name, fn)
	}

	go func() {
		// Close channel once all workers are done
		wg.Wait()
		close(metricsCh)
	}()

	// Retrieve metrics from channel and write them to the response
	var err error
	for m := range metricsCh {
		switch v := m.(type) {
		case []metric:
			if e.status != nil {
				e.status.MetricCount += len(v)
			}
			for _, m := range v {
				writeMetricMetadata(w, m)

				var timestamp string
				if !m.timestamp.IsZero() {
					timestamp = strconv.Itoa(int(m.timestamp.UnixNano() / 1000000))
				}
				_, _ = fmt.Fprintf(w, "%s %g %s\n", e.getMetricFullName(m), m.value, timestamp)
			}
		case error:
			err = v
			e.Logger.Println(v.Error())

			_, _ = fmt.Fprintf(w, "## %v\n", v)
		}
	}

	return err
}

func fetchMetricsWorker(wg *sync.WaitGroup, metricsCh chan<- interface{}, name string, fetchMetricsFn fetchMetricFn) {
	defer wg.Done()

	metrics, err := fetchMetricsFn()
	if err != nil {
		metricsCh <- fmt.Errorf("retrieve '#%s' metric: %w", name, err)
		return
	}

	metricsCh <- metrics
}

func (e *promExporter) Close() {
	if e.upsState.upsClient.ProtocolVersion != "" {
		e.upsState.upsLock.Lock()
		_, _ = e.upsState.upsClient.Disconnect()
		e.upsState.upsLock.Unlock()
	}
}

func (e *promExporter) readEnvironment() {
	e.Logger.Println("Reading environment...")

	e.readHostInfo()
	e.readSysInfo()
	e.readEnclosures()
	e.readNetworkInterfaces()
	e.readDevices()
	e.readNvmePath()
	e.readDmCacheDevices()

	e.envExpiry = e.envExpiry.Add(envValidity)

	e.updateStatusEnvironment()
}

func (e *promExporter) readHostInfo() {
	var err error
	e.hostname = os.Getenv("HOSTNAME")
	if e.hostname == "" {
		e.hostname, err = utils.ExecCommand("hostname")
	}
	e.Logger.Printf("Hostname: %s, err=%v", e.hostname, err)

	e.Logger.Println("Retrieving QTS version")
	kernelVersionStr, err := utils.ExecCommand("uname", "-r")
	if err == nil {
		e.kernelVersion, err = strconv.Atoi(strings.SplitN(kernelVersionStr, ".", 2)[0])
	}
	if err != nil {
		e.kernelVersion = 4
	}
}

func (e *promExporter) readSysInfo() {
	if e.getsysinfo == "" {
		var err error
		e.getsysinfo, err = exec.LookPath("getsysinfo")
		if err == nil {
			e.Logger.Printf("Retrieved getsysinfo path: %q", e.getsysinfo)
		} else {
			e.Logger.Printf("Failed to find getsysinfo: %v", err)
		}
	}
	if e.getsysinfo == "" {
		return
	}

	hdnumOutput, err := utils.ExecCommand(e.getsysinfo, "hdnum")
	if err == nil {
		e.syshdnum, _ = strconv.Atoi(hdnumOutput)
	} else {
		e.syshdnum = -1
	}
	e.Logger.Printf("Retrieved sysdhnum: %d", e.syshdnum)

	sysfannumOutput, err := utils.ExecCommand(e.getsysinfo, "sysfannum")
	if err == nil {
		e.sysfannum, _ = strconv.Atoi(sysfannumOutput)
	} else {
		e.sysfannum = -1
	}
	e.Logger.Printf("Retrieved sysfannum: %d", e.sysfannum)

	e.readSysVolInfo()
	e.Logger.Printf("Retrieved sysvolinfo")
}

func (e *promExporter) readEnclosures() {
	if e.halApp == "" {
		var err error
		e.halApp, err = exec.LookPath("hal_app")
		if err != nil {
			e.Logger.Printf("Failed to find hal_app: %v", err)
		}
		e.Logger.Printf("Retrieved hal_app path: %q", e.halApp)
	}
	e.enclosures = nil
	e.status.Enclosures = nil
	if e.halApp == "" {
		return
	}

	e.Logger.Println("Retrieving QM2 enclosures")
	seEnumOutput, err := utils.ExecCommand(e.halApp, "--se_enum")
	if err != nil {
		return
	}

	for _, line := range utils.FindMatchingLines("qm2_", seEnumOutput) {
		fields := strings.Fields(line)
		enc := qnapEnclosure{
			id:   fields[2],
			name: fields[4],
		}
		enc.diskCount, _ = strconv.Atoi(fields[7])
		enc.fanCount, _ = strconv.Atoi(fields[8])
		enc.tempCount, _ = strconv.Atoi(fields[10])
		if enc.fanCount != 0 {
			e.enclosures = append(e.enclosures, enc)
			e.status.Enclosures = append(e.status.Enclosures, enc.name)
		}
	}
}

func (e *promExporter) readNetworkInterfaces() {
	e.Logger.Printf("Retrieving network interfaces in %q...", netDir)
	info, _ := os.ReadDir(netDir)
	e.ifaces = make([]string, 0, len(info))
	for _, d := range info {
		iface := d.Name()
		if !strings.HasPrefix(iface, "eth") {
			continue
		}

		e.ifaces = append(e.ifaces, iface)
	}
}

func (e *promExporter) readDevices() {
	e.Logger.Printf("Retrieving devices in %q...", devDir)
	info, _ := os.ReadDir(devDir)
	e.devices = make([]string, 0, len(info))
	e.nvmeDevices = make([]string, 0)
	for _, d := range info {
		dev := d.Name()
		if d.IsDir() || !strings.HasPrefix(dev, "nvme") && !strings.HasPrefix(dev, "sd") {
			continue
		}
		switch {
		case strings.HasPrefix(dev, "nvme") && len(dev) != 7:
			continue
		case strings.HasPrefix(dev, "sd") && len(dev) != 3:
			continue
		}

		e.devices = append(e.devices, dev)
		if strings.HasPrefix(dev, "nvme") {
			e.nvmeDevices = append(e.nvmeDevices, dev)
		}
	}
	e.Logger.Printf("Found devices: %v", e.devices)
	e.Logger.Printf("Found NVMe devices: %v", e.nvmeDevices)
}

func (e *promExporter) readNvmePath() {
	// Look up nvme command path for NVMe SMART metrics
	if e.nvmePath == "" && len(e.nvmeDevices) > 0 {
		e.nvmePath, _ = exec.LookPath("nvme")
		if e.nvmePath != "" {
			e.Logger.Printf("Retrieved nvme path: %q", e.nvmePath)
		} else {
			e.Logger.Println("nvme command not found, NVMe SMART metrics will not be available")
		}
	}
}

func (e *promExporter) readDmCacheDevices() {
	e.dmCacheClients = []string{}
	if e.kernelVersion < 5 {
		return
	}

	e.Logger.Print("Retrieving dm-cache devices...")

	table, err := utils.ExecCommand("dmsetup", "table")
	if err == nil {
		cacheClients := utils.FindMatchingLines("cache_client", table)
		for _, cacheClient := range cacheClients {
			e.dmCacheClients = append(e.dmCacheClients, strings.SplitN(cacheClient, ":", 2)[0])
		}
	}
	e.Logger.Printf("Found cache clients: %v", e.dmCacheClients)

	table, err = utils.ExecCommand("dmsetup", "ls")
	if err == nil {
		cacheDevices := utils.FindMatchingLines("vg256-lv256\t", table)
		e.Logger.Printf("Found cache volumes: %v", cacheDevices)
		if len(cacheDevices) == 1 {
			e.dmCacheDeviceMinorNumber = strings.Split(cacheDevices[0], ":")[1]
			e.dmCacheDeviceMinorNumber = strings.TrimRight(e.dmCacheDeviceMinorNumber, ")")
		}
	}
}

func (e *promExporter) updateStatusEnvironment() {
	if e.status == nil {
		return
	}

	e.status.Devices = e.devices
	e.status.NvmeDevices = e.nvmeDevices
	e.status.Interfaces = e.ifaces
	e.status.DmCaches = e.dmCacheClients
	if e.dmCacheDeviceMinorNumber != "" {
		e.status.DmCacheDevice = fmt.Sprintf("dm-%s", e.dmCacheDeviceMinorNumber)
	} else {
		e.status.DmCacheDevice = ""
	}
}

func (e *promExporter) getMetricFullName(m metric) string {
	if m.attr != "" {
		return fmt.Sprintf(`%s{node=%q,%s}`, m.name, e.hostname, m.attr)
	}

	return fmt.Sprintf(`%s{node=%q}`, m.name, e.hostname)
}

func writeMetricMetadata(w io.Writer, m metric) {
	if m.help != "" {
		_, _ = fmt.Fprintln(w, "# HELP "+m.name+" "+m.help)
	}
	if m.metricType != "" {
		_, _ = fmt.Fprintln(w, "# TYPE "+m.name+" "+m.metricType)
	}
}
