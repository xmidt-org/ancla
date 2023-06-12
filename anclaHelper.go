package ancla

import (
	"github.com/prometheus/client_golang/prometheus"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func NewHelperMeasures() Measures {
	wlm := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: WebhookListSizeGaugeName,
			Help: WebhookListSizeGaugeHelp,
		},
	)
	cpm := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: ChrysomPollsTotalCounterName,
			Help: ChrysomPollsTotalCounterHelp,
		},
		[]string{OutcomeLabel},
	)

	return Measures{
		WebhookListSizeGaugeName:     wlm,
		ChrysomPollsTotalCounterName: cpm,
	}
}

func AnclaHelperMetrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: WebhookListSizeGaugeName,
			Type: xmetrics.GaugeType,
			Help: "Size of the current list of webhooks.",
		},
		{
			Name:       ChrysomPollsTotalCounterName,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			LabelNames: []string{OutcomeLabel},
		},
	}
}
