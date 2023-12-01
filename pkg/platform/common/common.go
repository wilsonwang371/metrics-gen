package common

import (
	"code.byted.org/bge-infra/metrics-gen/pkg/platform"
	"code.byted.org/bge-infra/metrics-gen/pkg/platform/gometrics"
	"code.byted.org/bge-infra/metrics-gen/pkg/platform/prometheus"
)

func MetricsProviderFactory(
	config platform.MetricsProviderConfig,
) platform.MetricsProvider {
	switch config.Provider {
	case "prometheus":
		return prometheus.NewPrometheusProvider(
			config.Inplace,
			config.Suffix,
			config.DryRun,
			config.MetricsPrefix,
		)
	case "gometrics":
		return gometrics.NewGoMetricsProvider(
			config.Inplace,
			config.Suffix,
			config.DryRun,
		)
	default:
		return nil
	}
}
