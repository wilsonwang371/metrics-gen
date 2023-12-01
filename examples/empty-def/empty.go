package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// +trace:define empty=true

// +trace:func-exec-time name=Test
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	time.Sleep(500 * time.Millisecond)
	return
}

func main() {
	reg := prometheus.NewRegistry()
	// +trace:set prom-registry=reg
	fmt.Printf("reg: %v\n", reg)
	Test()
}
