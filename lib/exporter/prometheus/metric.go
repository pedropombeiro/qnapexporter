package prometheus

import "time"

type metric struct {
	name       string
	attr       string
	timestamp  time.Time
	value      float64
	help       string
	metricType string
}
