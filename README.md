# metrics-gen

`metrics-gen` is a Go code generation tool that can inject code into Go source files based on directive comments specified within the source code. It allows you to automate the generation of metrics-related code, saving you time and ensuring consistency in your codebase.

## Features

- Automatic code generation based on directive comments.
- Support for injecting metrics-related code into Go source files.
- Easy customization and extensibility.

## Installation

```bash
go get code.byted.org/bge-infra/metrics-gen
```

## Compile

```bash
make build
```

## Usage

### 1. Add directive comments to your source code

`metrics-gen` uses directive comments to determine where to inject code into your source files. These comments are of the form `//+trace:...`. For example:

```go

// +trace:define interval=30 duration=1800 runtime-metrics=true runtime-metrics-interval=60
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified interval, duration and runtime-metrics-interval. If runtime-metrics is set to true,
// it will also start the runtime metrics collector.


// +trace:execution-time
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
  // ...
}

```

Meaning of the `//+trace:define` parameters:

- `interval`: The interval at which metrics are collected.
- `duration`: The duration for which metrics are stored.
- `runtime-metrics`: Whether to collect runtime metrics, such as memory usage and goroutine count.
- `runtime-metrics-interval`: The interval at which runtime metrics are collected.

Meaning of the `//+trace:execution-time` parameters:

- `cooldown-time-ms`: The cooldown time in milliseconds. If the function is called again within the cooldown time, the execution time will not be measured.

### 2. Run `metrics-gen`

```bash
# Run metrics-gen on a go project
# -i: in-place (overwrite the original source files)
# -r: recursive (process all files in the specified directory and its subdirectories)
metrics-gen generate -i -r <path/to/your/project>

```

### 3. Check the generated code

The final generated code will look like this:

```go

// +trace:define interval=30 duration=1800 runtime-metrics=true runtime-metrics-interval=60
// Above comment will generate a function named "init" in the same package. It will initialize the metrics
// with the specified interval, duration and runtime-metrics-interval. If runtime-metrics is set to true,
// it will also start the runtime metrics collector.
// +trace:begin-generated
func init() {
  // Setup the inmem sink and signal handler
  inm := metrics.NewInmemSink(30*time.Second, 1800*time.Second)
  metrics.DefaultInmemSignal(inm)
  cfg := metrics.DefaultConfig("application")
  cfg.EnableRuntimeMetrics = true
  cfg.ProfileInterval = 60 * time.Second
  metrics.NewGlobal(cfg, inm)
}

// +trace:end-generated

// +trace:execution-time
// Above comment will generate code to measure a function execution time. It will measure the time from
// the beginning of the function to the end of the function.
func Test() {
  // +trace:begin-generated
  defer metrics.MeasureSince([]string{"application#Create"}, time.Now())
  // +trace:end-generated
  // ...
}

```

### 4. Dump metrics

```bash
# Send a USR1 signal to the process to dump metrics
# In ArgoCD, the output can be found in the pod logs
kill -USR1 <pid>

```

## Limitations

- `metrics-gen` only supports Go source files.
- `metrics-gen` only supports `//+trace:...` comments. Other comment formats are not supported.
- `metrics-gen` only supports top-level `//+trace:...` comments. It does not support comments defined within other functions.
- `//+trace:define` cannot be used in a file that already contains a function named `init`.
