package main

import "time"

// +trace:define gm-interval=30 gm-duration=1800 gm-runtime-metrics=true gm-runtime-metrics-interval=60
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified gm-interval, gm-duration and gm-runtime-metrics-interval. If gm-runtime-metricsis set to true,
// it will also start the runtime metrics collector.

// +trace:func-exec-time name=Test
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	time.Sleep(500 * time.Millisecond)
	return
}

func main() {
	Test()
}
