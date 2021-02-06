package prometheus

import (
	"github.com/mackerelio/go-osstat/cpu"
)

func getCpuRatioMetrics() ([]metric, error) {
	s, err := cpu.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="user"`,
			metricType: "counter",
			value:      float64(s.User),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="nice"`,
			metricType: "counter",
			value:      float64(s.Nice),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="system"`,
			metricType: "counter",
			value:      float64(s.System),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="idle"`,
			metricType: "counter",
			value:      float64(s.Idle),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="iowait"`,
			metricType: "counter",
			value:      float64(s.Iowait),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="irq"`,
			metricType: "counter",
			value:      float64(s.Irq),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="softirq"`,
			metricType: "counter",
			value:      float64(s.Softirq),
		},
		{
			name:  "node_cpu_count",
			value: float64(s.CPUCount),
		},
	}

	return metrics, nil
}
