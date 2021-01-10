package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-ping/ping"
)

const (
	mountsRoot = "/share/"
	metricsDir = "/share/CACHEDEV1_DATA/Container/data/grafana/qnapnodeexporter"
	ifacesPath = "/sys/class/net/"
	pingTarget = "1.1.1.1"
)

var (
	hostname    string
	upsc        string
	upsName     string
	getsysinfo  string
	syshdnum    int
	sysfannum   int
	ifaces      []string
	iostat      string
	devices     []string
	mountpoints []string
)

type metric struct {
	name       string
	attr       string
	value      float64
	help       string
	metricType string
}

var fns = []func() ([]metric, error){
	uptime,
	getLoadAvg,
	getMemInfo,
	getCpuRatio,
	getUpsStats,
	getSysInfo,
	getFlashCacheStats,
	getNetworkStats,
	getDiskStats,
	getVolumeStats,
	getPingStats,
}

func main() {
	// Setup our Ctrl+C handler
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	readEnvironment()

	for {
		writeMetrics()

		select {
		case <-exitCh:
			fmt.Fprintln(os.Stderr, "Program aborted, exiting")
			os.Exit(1)
		case <-time.After(5 * time.Second):
			break
		}
	}
}

func writeMetrics() {
	var err error
	var tmpFile *os.File

	tmpFile, err = ioutil.TempFile(metricsDir, "qnapexporter-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %s\n", err.Error())
		return
	}
	defer os.Remove(tmpFile.Name())

	w := bufio.NewWriter(tmpFile)

	for _, fn := range fns {
		writeNodeMetrics(w, fn)
	}

	// Close the file
	w.Flush()
	if err := tmpFile.Chmod(0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing permissions of temporary file: %s\n", err.Error())
		return
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing temporary file: %s\n", err.Error())
		return
	}

	err = os.Rename(tmpFile.Name(), path.Join(metricsDir, "metrics"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error moving temporary file: %s\n", err.Error())
		return
	}
}

func readEnvironment() {
	hostname = os.Getenv("HOSTNAME")
	upsc, _ = exec.LookPath("upsc")
	if upsc != "" {
		upsName, _ = execCommand(upsc, "-l")
	}
	iostat, _ = exec.LookPath("iostat")
	getsysinfo, _ = exec.LookPath("getsysinfo")
	if getsysinfo != "" {
		hdnumOutput, err := execCommand(getsysinfo, "hdnum")
		if err == nil {
			syshdnum, _ = strconv.Atoi(hdnumOutput)
		} else {
			syshdnum = -1
		}

		sysfannumOutput, err := execCommand(getsysinfo, "sysfannum")
		if err == nil {
			sysfannum, _ = strconv.Atoi(sysfannumOutput)
		} else {
			sysfannum = -1
		}
	}

	info, _ := ioutil.ReadDir(ifacesPath)
	for _, d := range info {
		iface := d.Name()
		if !strings.HasPrefix(iface, "eth") {
			continue
		}

		ifaces = append(ifaces, iface)
	}

	info, _ = ioutil.ReadDir("/dev/")
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

		devices = append(devices, dev)
	}

	info, _ = ioutil.ReadDir(mountsRoot)
	for _, d := range info {
		mount := d.Name()
		if !strings.HasPrefix(mount, "C") || !strings.HasSuffix(mount, "_DATA") {
			continue
		}

		mountpoints = append(mountpoints, mount)
	}
}

func getMetricFullName(m metric) string {
	if m.attr != "" {
		return fmt.Sprintf(`%s{node=%q,%s}`, m.name, hostname, m.attr)
	}

	return fmt.Sprintf(`%s{node=%q}`, m.name, hostname)
}

func writeMetricMetadata(w io.Writer, m metric) {
	if m.help != "" {
		fmt.Fprintln(w, "# HELP "+m.name+" "+m.help)
	}
	if m.metricType != "" {
		fmt.Fprintln(w, "# TYPE "+m.name+" "+m.metricType)
	}
}

func writeNodeMetrics(w io.Writer, getMetricFn func() ([]metric, error)) {
	metrics, err := getMetricFn()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to retrieve metric: %s", err.Error())
		return
	}

	for _, metric := range metrics {
		writeMetricMetadata(w, metric)
		_, _ = fmt.Fprintf(w, "%s %f\n", getMetricFullName(metric), metric.value)
	}
}

func readFile(f string) (string, error) {
	contents, err := ioutil.ReadFile(f)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}

func readFileLines(f string) ([]string, error) {
	contents, err := readFile(f)
	if err != nil {
		return nil, err
	}

	return strings.Split(contents, "\n"), nil
}

func uptime() ([]metric, error) {
	data, err := readFile("/proc/uptime")
	if err != nil {
		return nil, err
	}

	value, err := strconv.ParseFloat(strings.SplitN(data, " ", 2)[0], 64)

	return []metric{
		{
			name:       "node_time_seconds",
			value:      value,
			help:       "System uptime measured in seconds",
			metricType: "counter",
		},
	}, err
}

func getLoadAvg() ([]metric, error) {
	data, err := readFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{name: "node_load1"},
		{name: "node_load5"},
		{name: "node_load15"},
	}
	values := strings.SplitN(data, " ", len(metrics)+1)[0:len(metrics)]
	for idx, value := range values {
		metrics[idx].value, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

func getMemInfo() ([]metric, error) {
	lines, err := readFileLines("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	namesMap := map[string]string{
		"MemTotal":     "",
		"MemFree":      "",
		"MemAvailable": "",
		"Buffers":      "",
		"Cached":       "",
		"SwapCached":   "",
		"Active":       "",
		"Inactive":     "",
		"Unevictable":  "",
		"Mlocked":      "",
		"SwapTotal":    "",
		"SwapFree":     "",
		"Dirty":        "",
		"Writeback":    "",
	}
	for _, line := range lines {
		tokens := strings.SplitN(line, ":", 2)
		name := tokens[0]
		if _, ok := namesMap[name]; ok {
			namesMap[name] = strings.TrimSpace(strings.TrimSuffix(tokens[1], " kB"))
		}
	}

	idx := 0
	metrics := make([]metric, len(namesMap))
	for name, value := range namesMap {
		metrics[idx].name = fmt.Sprintf("node_memory_%s_kbytes", name)
		metrics[idx].value, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		idx = idx + 1
	}

	return metrics, nil
}

func getCpuRatio() ([]metric, error) {
	data, err := readFile("/proc/stat")
	if err != nil {
		return nil, err
	}

	lines := strings.SplitN(data, "\n", 2)
	fields := strings.Fields(lines[0])
	user, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, err
	}
	nice, err := strconv.Atoi(fields[2])
	if err != nil {
		return nil, err
	}
	system, err := strconv.Atoi(fields[3])
	if err != nil {
		return nil, err
	}
	idle, err := strconv.Atoi(fields[4])
	if err != nil {
		return nil, err
	}
	iowait, err := strconv.Atoi(fields[5])
	if err != nil {
		return nil, err
	}
	irq, err := strconv.Atoi(fields[6])
	if err != nil {
		return nil, err
	}
	softirq, err := strconv.Atoi(fields[7])
	if err != nil {
		return nil, err
	}

	total := user + nice + system + idle + iowait + irq + softirq
	metrics := []metric{
		{
			name:  "node_cpu_ratio",
			attr:  `mode="user"`,
			value: float64(user) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="nice"`,
			value: float64(nice) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="system"`,
			value: float64(system) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="idle"`,
			value: float64(idle) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="iowait"`,
			value: float64(iowait) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="irq"`,
			value: float64(irq) / float64(total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="softirq"`,
			value: float64(softirq) / float64(total),
		},
	}

	return metrics, nil
}

func execCommand(cmd string, args ...string) (string, error) {
	var (
		err    error
		output []byte
	)

	c := exec.Command(cmd, args...)
	if output, err = c.Output(); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func execCommandGetLines(cmd string, args ...string) ([]string, error) {
	output, err := execCommand(cmd, args...)
	if err != nil {
		return nil, err
	}

	return strings.Split(output, "\n"), nil
}

func getUpsStats() ([]metric, error) {
	if upsName == "" {
		return nil, nil
	}

	lines, err := execCommandGetLines(upsc, upsName)
	if err != nil {
		return nil, err
	}

	metrics := make([]metric, 0, len(lines))
	var status, firmware string
	for _, line := range lines {
		tokens := strings.SplitN(line, ":", 2)
		valueStr := strings.TrimSpace(tokens[1])
		switch tokens[0] {
		case "ups.status":
			status = valueStr
			continue
		case "ups.firmware":
			firmware = valueStr
			continue
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err == nil {
			metrics = append(metrics, metric{
				name:  "ups_" + strings.ReplaceAll(tokens[0], ".", "_"),
				value: value,
			})
		}
	}
	metrics = append(metrics, metric{
		name:  "ups_ups_status",
		attr:  fmt.Sprintf(`status=%q,firmware=%q`, status, firmware),
		value: getUpsStatus(status),
	})

	return metrics, nil
}

func getUpsStatus(status string) float64 {
	switch status {
	case "OL":
		return 0
	case "CHRG":
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

func getSysInfo() ([]metric, error) {
	if getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, 8)

	for _, dev := range []string{"cputmp", "systmp"} {
		output, err := execCommand(getsysinfo, dev)
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

	for hdnum := 1; hdnum <= syshdnum; hdnum++ {
		hdnumStr := strconv.Itoa(hdnum)
		smart, err := execCommand(getsysinfo, "hdsmart", hdnumStr)
		if err != nil {
			return nil, err
		}
		if smart == "--" {
			continue
		}

		tempStr, err := execCommand(getsysinfo, "hdtmp", hdnumStr)
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

	for fannum := 1; fannum <= sysfannum; fannum++ {
		fannumStr := strconv.Itoa(fannum)

		fanStr, err := execCommand(getsysinfo, "sysfan", fannumStr)
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

func getFlashCacheStats() ([]metric, error) {
	lines, err := readFileLines("/proc/flashcache/CG0/flashcache_stats")
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

func getNetworkStats() ([]metric, error) {
	metrics := make([]metric, 0, len(ifaces)*2)
	for _, iface := range ifaces {
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
	str, err := readFile("/sys/class/net/" + iface + "/statistics/" + direction + "_bytes")
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

func getDiskStats() ([]metric, error) {
	if iostat == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, len(devices)*2)
	for _, dev := range devices {
		readMetric, err := getDiskStatMetric("node_disk_read_kbytes_total", "Total number of kilobytes read", dev, 5)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, readMetric)

		writeMetric, err := getDiskStatMetric("node_disk_written_kbytes_total", "Total number of kilobytes written", dev, 6)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, writeMetric)
	}

	return metrics, nil
}

func getDiskStatMetric(name string, help string, dev string, field int) (metric, error) {
	if iostat == "" {
		return metric{}, nil
	}

	lines, err := execCommandGetLines(iostat, "-d", dev, "-k")
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

func getVolumeStats() ([]metric, error) {
	metrics := make([]metric, 0, len(mountpoints)*2)

	for _, mountpoint := range mountpoints {
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

func getPingStats() ([]metric, error) {
	pinger, err := ping.NewPinger(pingTarget)
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
	m := metric{
		name:  "node_network_external_roundtrip_time_ms",
		value: float64(stats.AvgRtt.Seconds()) * 1000.0,
	}

	return []metric{m}, nil
}
