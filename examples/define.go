package main

import (
	"os"
	"syscall"
	"time"
)

// +trace:define

// start
// +trace:func-exec-time cooldown-time=5ms
func define_func1() {
	// this a comment
	time.Sleep(100 * time.Millisecond)
	return
}

func main() {
	// call definf_func1 in goroutines
	for i := 0; i < 10; i++ {
		go define_func1()
	}

	// +trace:inner-exec-time
	time.Sleep(2 * time.Second)
	// send signal SIGUSR1 to self process trace
	pid := os.Getpid()
	selfProcess, _ := os.FindProcess(pid)
	selfProcess.Signal(syscall.SIGUSR1)
	time.Sleep(2 * time.Second)

	return
}
