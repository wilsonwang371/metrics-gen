package main

import globalvar "github.com/wilsonwang371/globalvar/pkg"
import promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
import prometheus "github.com/prometheus/client_golang/prometheus"
import http "net/http"

import (
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// +trace:define prom-port=9123
// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
func init() {
	reg := prometheus.NewRegistry()
	globalvar.Set("metrics_gen", reg)
	go func() {
		http.Handle("/metrics-gen", promhttp.HandlerFor(prometheus.Gatherers{reg, prometheus.DefaultGatherer}, promhttp.HandlerOpts{}))
		http.ListenAndServe(":9123", nil)
	}()
}

// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7

// start
// +trace:func-exec-time name=define_func1 gm-cooldown-time=5ms
// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
var fn_define_func1_initialized bool = false
var fn_define_func1_mutex sync.Mutex
var fn_define_func1 prometheus.Summary = prometheus.NewSummary(prometheus.SummaryOpts{Name: "metrics_gen_fn_define_func1", Help: "metrics_gen_fn_define_func1", Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}})

// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
func define_func1() {
	// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	defer func(t time.Time) {
		if !fn_define_func1_initialized {
			fn_define_func1_mutex.Lock()
			if !fn_define_func1_initialized {
				reg, err := globalvar.Get("metrics_gen")
				if err == nil {
					fn_define_func1_initialized = true
					reg.(*prometheus.Registry).MustRegister(fn_define_func1)
				}
			}
			fn_define_func1_mutex.Unlock()
		}
		d := time.Since(t)
		fn_define_func1.Observe(d.Seconds())
	}(time.Now())
	// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	// this a comment
	time.Sleep(500 * time.Millisecond)

	// +trace:inner-counter name=main_func_counter2
	// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	if !main_define_func1_main_func_counter2_17956061_initialized {
		main_define_func1_main_func_counter2_17956061_mutex.Lock()
		if !main_define_func1_main_func_counter2_17956061_initialized {
			reg, err := globalvar.Get("metrics_gen")
			if err == nil {
				main_define_func1_main_func_counter2_17956061_initialized = true
				reg.(*prometheus.Registry).MustRegister(main_define_func1_main_func_counter2_17956061)
			}
		}
		main_define_func1_main_func_counter2_17956061_mutex.Unlock()
	}
	main_define_func1_main_func_counter2_17956061.Inc()
	// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	return
}

// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
var main_define_func1_main_func_counter2_17956061_initialized bool = false
var main_define_func1_main_func_counter2_17956061_mutex sync.Mutex
var main_define_func1_main_func_counter2_17956061 prometheus.Counter = prometheus.NewCounter(prometheus.CounterOpts{Name: "metrics_gen_main_define_func1_main_func_counter2", Help: "metrics_gen_main_define_func1_main_func_counter2"})

// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7

// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
var main_func_exec_time_initialized bool = false
var main_func_exec_time_mutex sync.Mutex
var main_func_exec_time prometheus.Summary = prometheus.NewSummary(prometheus.SummaryOpts{Name: "metrics_gen_main_func_exec_time", Help: "metrics_gen_main_func_exec_time", Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}})

// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7

// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
var main_main_main_func_counter_96182894_initialized bool = false
var main_main_main_func_counter_96182894_mutex sync.Mutex
var main_main_main_func_counter_96182894 prometheus.Counter = prometheus.NewCounter(prometheus.CounterOpts{Name: "metrics_gen_main_main_main_func_counter", Help: "metrics_gen_main_main_main_func_counter"})

// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7

func main() {
	wg := sync.WaitGroup{}
	// call definf_func1 in goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			define_func1()
			wg.Done()
		}()
	}

	wg.Wait()
	log.Infof("main func done")

	func() {
		// +trace:inner-exec-time name=inner_func_exec_time
		time.Sleep(1 * time.Second)
	}()

	// +trace:inner-exec-time name=main_func_exec_time
	// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	defer func(t time.Time) {
		if !main_func_exec_time_initialized {
			main_func_exec_time_mutex.Lock()
			if !main_func_exec_time_initialized {
				reg, err := globalvar.Get("metrics_gen")
				if err == nil {
					main_func_exec_time_initialized = true
					reg.(*prometheus.Registry).MustRegister(main_func_exec_time)
				}
			}
			main_func_exec_time_mutex.Unlock()
		}
		d := time.Since(t)
		main_func_exec_time.Observe(d.Seconds())
	}(time.Now())

	// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	time.Sleep(10 * time.Second)
	// send signal SIGUSR1 to self process trace
	pid := os.Getpid()

	// +trace:inner-counter name=main_func_counter
	// +trace:begin-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	if !main_main_main_func_counter_96182894_initialized {
		main_main_main_func_counter_96182894_mutex.Lock()
		if !main_main_main_func_counter_96182894_initialized {
			reg, err := globalvar.Get("metrics_gen")
			if err == nil {
				main_main_main_func_counter_96182894_initialized = true
				reg.(*prometheus.Registry).MustRegister(main_main_main_func_counter_96182894)
			}
		}
		main_main_main_func_counter_96182894_mutex.Unlock()
	}
	main_main_main_func_counter_96182894.Inc()

	// +trace:end-generated uuid=f5ae69ce-4efe-482d-922c-61542d0bd8b7
	selfProcess, _ := os.FindProcess(pid)
	selfProcess.Signal(syscall.SIGUSR1)
	time.Sleep(5 * time.Second)

	return
}
