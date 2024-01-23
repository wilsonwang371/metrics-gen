package main

import "sync"
import globalvar "github.com/wilsonwang371/globalvar/pkg"

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// +trace:define empty=true

// +trace:func-exec-time name=Test
// +trace:begin-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
var fn_Test_initialized bool = false
var fn_Test_mutex sync.Mutex
var fn_Test prometheus.Summary = prometheus.NewSummary(prometheus.SummaryOpts{Name: "metrics_gen_fn_Test", Help: "metrics_gen_fn_Test", Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}})

// +trace:end-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	// +trace:begin-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
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
	// +trace:end-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
	time.Sleep(500 * time.Millisecond)
	return
}

// +trace:begin-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
var empty_main_16287105_initialized bool = false
var empty_main_16287105_mutex sync.Mutex

// +trace:end-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f

func main() {
	reg := prometheus.NewRegistry()

	// +trace:set prom-registry=reg
	// +trace:begin-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
	if !empty_main_16287105_initialized {
		empty_main_16287105_mutex.Lock()
		if !empty_main_16287105_initialized {
			globalvar.Set("metrics_gen", reg)
			empty_main_16287105_initialized = true
		}
		empty_main_16287105_mutex.Unlock()
	}
	// +trace:end-generated uuid=50138ffc-a0ce-4207-bb79-ddcd7891c33f
	fmt.Printf("reg: %v\n", reg)
	Test()
}
