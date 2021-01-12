package prometheus

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-ping/ping"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/loadavg"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	nut "github.com/robbiet480/go.nut"

	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
	"gitlab.com/pedropombeiro/qnapexporter/lib/utils"
)

const (
	mountsRoot          = "/share"
	devDir              = "/dev"
	netDir              = "/sys/class/net"
	flashcacheStatsPath = "/proc/flashcache/CG0/flashcache_stats"
)

type promExporter struct {
	hostname        string
	pingTarget      string
	upsClient       nut.Client
	upsConnErr      error
	upsConnAttempts int
	upsLock         sync.Mutex
	getsysinfo      string
	syshdnum        int
	sysfannum       int
	ifaces          []string
	iostat          string
	devices         []string
	mountpoints     []string

	fns []func() ([]metric, error)
}

func NewExporter(pingTarget string) exporter.Exporter {
	e := &promExporter{
		pingTarget: pingTarget,
	}
	e.fns = []func() ([]metric, error){
		getUptimeMetrics,
		getLoadAvgMetrics,
		getMemInfoMetrics,
		getCpuRatioMetrics,
		e.getUpsStatsMetrics,
		e.getSysInfoMetrics,
		getFlashCacheStatsMetrics,
		e.getNetworkStatsMetrics,
		e.getDiskStatsMetrics,
		e.getVolumeStatsMetrics,
		e.getPingMetrics,
	}

	e.readEnvironment()

	return e
}

func (e *promExporter) WriteMetrics(w io.Writer) error {
	for idx, fn := range e.fns {
		err := e.writeNodeMetrics(w, fn, idx)
		if err != nil {
			e.logger.Println(err.Error())

			_, _ = fmt.Fprintf(w, "## %v\n", err)
		}
	}

	return nil
}

func (e *promExporter) Close() {
	if e.upsClient.ProtocolVersion != "" {
		e.upsLock.Lock()
		e.upsClient.Disconnect()
		e.upsLock.Unlock()
	}
}

func (e *promExporter) readEnvironment() {
	log.Println("Reading environment values")

	e.hostname = os.Getenv("HOSTNAME")
	e.iostat, _ = exec.LookPath("iostat")
	e.getsysinfo, _ = exec.LookPath("getsysinfo")
	if e.getsysinfo != "" {
		hdnumOutput, err := utils.ExecCommand(e.getsysinfo, "hdnum")
		if err == nil {
			e.syshdnum, _ = strconv.Atoi(hdnumOutput)
		} else {
			e.syshdnum = -1
		}

		sysfannumOutput, err := utils.ExecCommand(e.getsysinfo, "sysfannum")
		if err == nil {
			e.sysfannum, _ = strconv.Atoi(sysfannumOutput)
		} else {
			e.sysfannum = -1
		}
	}

	info, _ := ioutil.ReadDir(netDir)
	for _, d := range info {
		iface := d.Name()
		if !strings.HasPrefix(iface, "eth") {
			continue
		}

		e.ifaces = append(e.ifaces, iface)
	}

	info, _ = ioutil.ReadDir(devDir)
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
	}

	info, _ = ioutil.ReadDir(mountsRoot)
	for _, d := range info {
		mount := d.Name()
		if !strings.HasPrefix(mount, "C") || !strings.HasSuffix(mount, "_DATA") {
			continue
		}

		e.mountpoints = append(e.mountpoints, mount)
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
		fmt.Fprintln(w, "# HELP "+m.name+" "+m.help)
	}
	if m.metricType != "" {
		fmt.Fprintln(w, "# TYPE "+m.name+" "+m.metricType)
	}
}

func (e *promExporter) writeNodeMetrics(w io.Writer, getMetricFn func() ([]metric, error), index int) error {
	metrics, err := getMetricFn()
	if err != nil {
		return fmt.Errorf("retrieve metric #%d: %w", 1+index, err)
	}

	for _, metric := range metrics {
		writeMetricMetadata(w, metric)
		_, _ = fmt.Fprintf(w, "%s %f\n", e.getMetricFullName(metric), metric.value)
	}

	return nil
}

func getUptimeMetrics() ([]metric, error) {
	u, err := uptime.Get()
	if err != nil {
		return nil, err
	}

	return []metric{
		{
			name:       "node_time_seconds",
			value:      u.Seconds(),
			help:       "System uptime measured in seconds",
			metricType: "counter",
		},
	}, err
}

func getLoadAvgMetrics() ([]metric, error) {
	s, err := loadavg.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{name: "node_load1", value: s.Loadavg1},
		{name: "node_load5", value: s.Loadavg5},
		{name: "node_load15", value: s.Loadavg15},
	}
	return metrics, nil
}

func getMemInfoMetrics() ([]metric, error) {
	s, err := memory.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{name: "node_memory_MemTotal_bytes", value: float64(s.Total)},
		{name: "node_memory_MemFree_bytes", value: float64(s.Free)},
		{name: "node_memory_Cached_bytes", value: float64(s.Cached)},
		{name: "node_memory_Active_bytes", value: float64(s.Active)},
		{name: "node_memory_Inactive_bytes", value: float64(s.Inactive)},
		{name: "node_memory_SwapTotal_bytes", value: float64(s.SwapTotal)},
		{name: "node_memory_SwapFree_bytes", value: float64(s.SwapFree)},
	}
	if s.MemAvailableEnabled {
		metrics = append(metrics, metric{name: "node_memory_MemAvailable_bytes", value: float64(s.Available)})
	}

	return metrics, nil
}

func getCpuRatioMetrics() ([]metric, error) {
	s, err := cpu.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{
			name:  "node_cpu_ratio",
			attr:  `mode="user"`,
			value: float64(s.User) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="nice"`,
			value: float64(s.Nice) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="system"`,
			value: float64(s.System) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="idle"`,
			value: float64(s.Idle) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="iowait"`,
			value: float64(s.Iowait) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="irq"`,
			value: float64(s.Irq) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="softirq"`,
			value: float64(s.Softirq) / float64(s.Total),
		},
	}

	return metrics, nil
}

func (e *promExporter) getUpsStatsMetrics() ([]metric, error) {
	e.upsLock.Lock()
	defer e.upsLock.Unlock()

	if e.upsClient.ProtocolVersion == "" {
		if e.upsConnAttempts < 10 {
			e.upsConnAttempts++
			e.upsClient, e.upsConnErr = nut.Connect("127.0.0.1")
		}
		if e.upsConnErr != nil {
			return nil, e.upsConnErr
		}
	}

	upsList, err := e.upsClient.GetUPSList()
	if err != nil {
		return nil, err
	}

	if len(upsList) == 0 {
		return nil, nil
	}

	var metrics []metric
	for _, ups := range upsList {
		vars, err := ups.GetVariables()
		if err != nil {
			return nil, err
		}

		if metrics == nil {
			metrics = make([]metric, 0, len(vars)*len(upsList))
		}

		attr := fmt.Sprintf("ups=%q", ups.Name)

		var status, statusHelp, firmware string
		for _, v := range vars {
			switch v.Name {
			case "ups.status":
				status = v.Value.(string)
				statusHelp = v.Description
				continue
			case "ups.firmware":
				firmware = v.Value.(string)
				continue
			}

			var value float64
			switch v.Type {
			case "INTEGER":
				value = float64(v.Value.(int64))
				break
			case "FLOAT_64":
				value = v.Value.(float64)
				break
			default:
				continue
			}

			metrics = append(metrics, metric{
				name:  "ups_" + strings.ReplaceAll(v.Name, ".", "_"),
				attr:  attr,
				value: value,
				help:  v.Description,
			})
		}
		metrics = append(metrics, metric{
			name:  "ups_ups_status",
			attr:  fmt.Sprintf(`status=%q,firmware=%q,%s`, status, firmware, attr),
			value: getUpsStatus(status),
			help:  statusHelp,
		})
	}

	return metrics, nil
}

func getUpsStatus(status string) float64 {
	switch status {
	case "OL":
		return 0
	case "OL CHRG":
		return 1
	case "OB", "LB", "HB", "DISCHRG":
		return 2
	case "OFF":
		return 3
	case "RB":
		return 999
	default:
		return 99
	}
}

func (e *promExporter) getSysInfoMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, 8)

	for _, dev := range []string{"cputmp", "systmp"} {
		output, err := utils.ExecCommand(e.getsysinfo, dev)
		if err != nil {
			return nil, err
		}
		tokens := strings.SplitN(output, " ", 2)
		value, err := strconv.ParseFloat(tokens[0], 64)
		if err == nil {
			metrics = append(metrics, metric{
				name:  fmt.Sprintf("node_%s_C", dev),
				value: value,
			})
		}
	}

	for hdnum := 1; hdnum <= e.syshdnum; hdnum++ {
		hdnumStr := strconv.Itoa(hdnum)
		smart, err := utils.ExecCommand(e.getsysinfo, "hdsmart", hdnumStr)
		if err != nil {
			return nil, err
		}
		if smart == "--" {
			continue
		}

		tempStr, err := utils.ExecCommand(e.getsysinfo, "hdtmp", hdnumStr)
		if err != nil {
			return nil, err
		}

		temp, err := strconv.ParseFloat(strings.SplitN(tempStr, " ", 2)[0], 64)
		if err != nil {
			return metrics, err
		}
		metrics = append(metrics, metric{
			name:  "node_hdtmp_C",
			attr:  fmt.Sprintf(`hd=%q,smart=%q`, hdnumStr, smart),
			value: temp,
		})
	}

	for fannum := 1; fannum <= e.sysfannum; fannum++ {
		fannumStr := strconv.Itoa(fannum)

		fanStr, err := utils.ExecCommand(e.getsysinfo, "sysfan", fannumStr)
		if err != nil {
			return nil, err
		}

		fan, err := strconv.ParseFloat(strings.SplitN(fanStr, " ", 2)[0], 64)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric{
			name:  "node_sysfan_RPM",
			attr:  fmt.Sprintf(`fan=%q`, fannumStr),
			value: fan,
		})
	}

	return metrics, nil
}

func getFlashCacheStatsMetrics() ([]metric, error) {
	lines, err := utils.ReadFileLines(flashcacheStatsPath)
	if err != nil {
		return nil, err
	}

	metrics := make([]metric, 0, len(lines))
	for _, line := range lines {
		tokens := strings.SplitN(line, ":", 2)
		valueStr := strings.TrimSpace(tokens[1])
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, metric{
			name:  "node_flashcache_" + tokens[0],
			value: value,
		})
	}

	return metrics, nil
}

func (e *promExporter) getNetworkStatsMetrics() ([]metric, error) {
	metrics := make([]metric, 0, len(e.ifaces)*2)
	for _, iface := range e.ifaces {
		rxMetric, err := getNetworkStatMetric("node_network_receive_bytes_total", "Total number of bytes received", iface, "rx")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, rxMetric)

		txMetric, err := getNetworkStatMetric("node_network_transmit_bytes_total", "Total number of bytes transmitted", iface, "tx")
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, txMetric)
	}

	return metrics, nil
}

func getNetworkStatMetric(name string, help string, iface string, direction string) (metric, error) {
	str, err := utils.ReadFile(path.Join(netDir, iface, "statistics", direction+"_bytes"))
	if err != nil {
		return metric{}, err
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return metric{}, err
	}

	return metric{
		name:       name,
		attr:       fmt.Sprintf(`device=%q`, iface),
		value:      value,
		help:       help,
		metricType: "counter",
	}, nil
}

func (e *promExporter) getDiskStatsMetrics() ([]metric, error) {
	if e.iostat == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, len(e.devices)*2)
	for _, dev := range e.devices {
		readMetric, err := e.getDiskStatMetric("node_disk_read_kbytes_total", "Total number of kilobytes read", dev, 5)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, readMetric)

		writeMetric, err := e.getDiskStatMetric("node_disk_written_kbytes_total", "Total number of kilobytes written", dev, 6)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, writeMetric)
	}

	return metrics, nil
}

func (e *promExporter) getDiskStatMetric(name string, help string, dev string, field int) (metric, error) {
	if e.iostat == "" {
		return metric{}, nil
	}

	lines, err := utils.ExecCommandGetLines(e.iostat, "-d", dev, "-k")
	if err != nil {
		return metric{}, err
	}
	line := lines[len(lines)-1]
	fields := strings.Fields(line)

	value, err := strconv.ParseFloat(fields[field], 64)
	if err != nil {
		return metric{}, err
	}

	return metric{
		name:       name,
		attr:       fmt.Sprintf(`device=%q`, dev),
		value:      value,
		help:       help,
		metricType: "counter",
	}, nil
}

func (e *promExporter) getVolumeStatsMetrics() ([]metric, error) {
	metrics := make([]metric, 0, len(e.mountpoints)*2)

	for _, mountpoint := range e.mountpoints {
		var stat syscall.Statfs_t

		dir := path.Join(mountsRoot, mountpoint)
		attr := fmt.Sprintf(`filesystem=%q`, dir)
		err := syscall.Statfs(dir, &stat)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, metric{
			name:  "node_filesystem_avail_kbytes",
			attr:  attr,
			value: float64(stat.Bavail * uint64(stat.Bsize) / 1024),
		})

		metrics = append(metrics, metric{
			name:  "node_filesystem_size_kbytes",
			attr:  attr,
			value: float64(stat.Blocks * uint64(stat.Bsize) / 1024),
		})
	}

	return metrics, nil
}

func (e *promExporter) getPingMetrics() ([]metric, error) {
	pinger, err := ping.NewPinger(e.pingTarget)
	if err != nil {
		return nil, err
	}

	pinger.SetPrivileged(true)
	pinger.Timeout = 2 * time.Second
	pinger.Count = 1
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return nil, err
	}

	stats := pinger.Statistics() // get send/receive/rtt stats
	value := float64(stats.AvgRtt.Seconds()) * 1000.0
	if stats.PacketLoss > 0 {
		value = math.NaN()
	}
	m := metric{
		name:  "node_network_external_roundtrip_time_ms",
		attr:  fmt.Sprintf("target=%q", pinger.IPAddr().String()),
		value: value,
	}

	return []metric{m}, nil
}
