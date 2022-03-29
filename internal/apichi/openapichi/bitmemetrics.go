package openapichi

import "github.com/prometheus/client_golang/prometheus"

const (
	Namespace   = "bitmemetrics"
	LabelMethod = "method"
	LabelStatus = "status"
)

type BitmeMetrics struct {
	fullUrlLengthHistogram prometheus.Gauge
}

func (m *BitmeMetrics) Init() error {
	// prometheus type: histogram
	m.fullUrlLengthHistogram = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "last_line_length",
		Help:      "The length of the full url recieved",
	})

	err := prometheus.Register(m.fullUrlLengthHistogram)

	if err != nil {
		return err
	}

	return nil
}
