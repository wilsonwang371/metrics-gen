package main

import "sync"
import promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
import prometheus "github.com/prometheus/client_golang/prometheus"
import http "net/http"
import globalvar "github.com/wilsonwang371/globalvar/pkg"

import "time"

// +trace:define gm-interval=30 gm-duration=1800 gm-runtime-metrics=true gm-runtime-metrics-interval=60
// +trace:begin-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
func init() {
	reg := prometheus.NewRegistry()
	globalvar.Set("metrics_gen", reg)
	go func() {
		http.Handle("/metrics-gen", promhttp.HandlerFor(prometheus.Gatherers{reg, prometheus.DefaultGatherer}, promhttp.HandlerOpts{}))
		http.ListenAndServe(":9123", nil)
	}()
}

// +trace:end-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified gm-interval, gm-duration and gm-runtime-metrics-interval. If gm-runtime-metricsis set to true,
// it will also start the runtime metrics collector.

// +trace:func-exec-time name=Test
// +trace:begin-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
var fn_Test_initialized bool = false
var fn_Test_mutex sync.Mutex
var fn_Test prometheus.Summary = prometheus.NewSummary(prometheus.SummaryOpts{Name: "metrics_gen_fn_Test", Help: "metrics_gen_fn_Test", Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}})

// +trace:end-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	// +trace:begin-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
	defer func(t time.Time) {
		if !fn_Test_initialized {
			fn_Test_mutex.Lock()
			if !fn_Test_initialized {
				reg, err := globalvar.Get("metrics_gen")
				if err == nil {
					fn_Test_initialized = true
					reg.(*prometheus.Registry).MustRegister(fn_Test)
				}
			}
			fn_Test_mutex.Unlock()
		}
		d := time.Since(t)
		fn_Test.Observe(d.Seconds())
	}(time.Now())
	// +trace:end-generated uuid=fe9d9b50-0fee-47a4-a407-6e3a5fb48b9b
	time.Sleep(500 * time.Millisecond)
	return
}

func main() {
	Test()
}
