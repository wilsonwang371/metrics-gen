# metrics-gen

`metrics-gen` is a Go code generation tool that can inject code into Go source files based on directive comments specified within the source code. It allows you to automate the generation of metrics-related code, saving you time and ensuring consistency in your codebase.

## Features

- Automatic code generation based on directive comments.
- Support for injecting metrics-related code into Go source files.
- Easy customization and extensibility.

## Installation

```bash
go get github.com/wilsonwang371/metrics-gen/metrics-gen
```

## Compile

```bash
make build
```

## Usage

### Metrics Types

1. `define`: Define the metrics provider and its parameters.
2. `set`: Use specified metrics provider to initialize metrics.
3. `func-exec-time`: Measure the execution time of a function.
4. `inner-exec-time`: Measure the execution time of a code block.
5. `inner-counter`: Count the number of times a line of code is executed.


### 1. Add directive comments to your source code

`metrics-gen` uses directive comments to determine where to inject code into your source files. These comments are of the form `//+trace:...`. For example:

```go

package main

import "time"

// +trace:define gm-interval=30 gm-duration=1800 gm-runtime-metrics=true gm-runtime-metrics-interval=60
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified gm-interval, gm-duration and gm-runtime-metrics-interval. If gm-runtime-metricsis set to true,
// it will also start the runtime metrics collector.

// +trace:func-exec-time
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	time.Sleep(500 * time.Millisecond)
	return
}


```

Meaning of the `//+trace:define` parameters:

- `gm-interval`: The interval at which metrics are collected by go-metrics.
- `gm-duration`: The duration for which metrics are stored by go-metrics.
- `runtime-metrics`: Whether to collect runtime metrics, such as memory usage and goroutine count.
- `runtime-metrics-interval`: The interval at which runtime metrics are collected.

Meaning of the `//+trace:func-exec-time` parameters:

- `gm-cooldown-time`: The cooldown time. If the function is called again within the cooldown time, the execution time will not be measured.
- `prom-port`: The port on which the Prometheus server is listening. If this parameter is specified, the generated code will expose the metrics to the Prometheus server.
### 2. Run `metrics-gen`

```bash
# Run metrics-gen on a go project
# -i: in-place (overwrite the original source files)
# -r: recursive (process all files in the specified directory and its subdirectories)
metrics-gen generate -i -r <path/to/your/project>

```

### 3. Check the generated code

By default, `metrics-gen` will generate code that uses the `prometheus` provider. If you want to use the `go-metrics` provider, you can specify the `-p` option when running `metrics-gen`.

The final generated code will look like this if you use `prometheus` provider

```go
package main

import prometheus "github.com/prometheus/client_golang/prometheus"
import promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
import http "net/http"

import "time"

// +trace:define gm-interval=30 gm-duration=1800 gm-runtime-metrics=true gm-runtime-metrics-interval=60
// +trace:begin-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
func init() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9123", nil)
	}()
}

// +trace:end-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified gm-interval, gm-duration and gm-runtime-metrics-interval. If gm-runtime-metricsis set to true,
// it will also start the runtime metrics collector.

// +trace:func-exec-time
// +trace:begin-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
var test_Test_duration prometheus.Histogram = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "test_Test_duration", Help: "test_Test_duration"})

func init() { prometheus.MustRegister(test_Test_duration) }

// +trace:end-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
	// +trace:begin-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
	defer func() {
		d := time.Since(time.Now())
		test_Test_duration.Observe(d.Seconds())
	}()
	// +trace:end-generated uuid=05eff6f7-15ad-4e2d-b144-2fdec692f051
	time.Sleep(500 * time.Millisecond)
	return
}

```

### 4. Dump metrics

For go-metrics provider, you can dump metrics by sending a USR1 signal to the process.

```bash
# Send a USR1 signal to the process to dump metrics
# In ArgoCD, the output can be found in the pod logs
kill -USR1 <pid>

```

## Limitations

- `metrics-gen` only supports Go source files.
- `metrics-gen` only supports `//+trace:...` comments. Other comment formats are not supported.
- `metrics-gen` only supports top-level `//+trace:...` comments. It does not support comments defined within other functions.
