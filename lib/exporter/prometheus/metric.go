package prometheus

type metric struct {
	name       string
	attr       string
	value      float64
	help       string
	metricType string
}
