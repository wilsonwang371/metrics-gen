package main

import (
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// +trace:define prom-port=9123

// start
// +trace:func-exec-time name=define_func1 gm-cooldown-time=5ms
func define_func1() {
	// this a comment
	time.Sleep(500 * time.Millisecond)
	// +trace:inner-counter name=main_func_counter2
	return
}

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

	// +trace:inner-exec-time
	time.Sleep(10 * time.Second)
	// send signal SIGUSR1 to self process trace
	pid := os.Getpid()

	// +trace:inner-counter name=main_func_counter
	selfProcess, _ := os.FindProcess(pid)
	selfProcess.Signal(syscall.SIGUSR1)
	time.Sleep(5 * time.Second)

	return
}
