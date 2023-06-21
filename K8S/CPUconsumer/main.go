package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/namsral/flag"
)

var (
	version        = "no version set"
	displayVersion = flag.Bool("version", false, "Show version and quit")
	logLevel       = flag.String("logLevel", "warn", "log level from debug, info, warning, error. When debug, genetate 100% Tracing")
	numCPU         = flag.Float64("numCPU", 0, "how many CPU to use (override GOMAXPROCS)")
	waitDuration   = flag.Duration("waitDuration", time.Duration(30)*time.Second, "how long to wait between work, in Golang Duration")
	workDuration   = flag.Duration("workDuration", time.Duration(10)*time.Second, "how long to work, then wait, in Golang Duration")
)

func printVersion() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("App Version: %v\n", version)
}

func main() {

	flag.Parse()
	if *displayVersion {
		printVersion()
		os.Exit(0)
	}

	// Set the number of full CPU
	n := int(*numCPU)

	// CPU mili cores
	m := math.Mod(*numCPU, 1)

	// total number of routines to start
	t := int(*numCPU)

	// if not set, use all the CPUs as discovered by the system
	if n == 0 && m == 0 {
		n = runtime.NumCPU()
		t = n
	} else if m == 0 {
		// if CPU is round, set GOMAXPROCS to the number of full CPU
		t = n
	} else {
		t = n + 1
	}

	runtime.GOMAXPROCS(t)

	// quit channel is used to force exit of all threads
	quit := make(chan bool, t)

	// Grab SIGINT and exit all threads
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(t int) {
		<-c
		fmt.Println("got SIGINT, terminating")
		cleanup(t, quit)
		os.Exit(1)
	}(t)

	for {
		// start all consumer threads
		fmt.Printf("starting %d+%.3f threads\n", n, m)
		work(n, m, quit)

		fmt.Printf("waiting for %s\n", *workDuration)
		time.Sleep(*workDuration)

		for i := 0; i < t; i++ {
			quit <- true
		}

		fmt.Printf("Job done, sleeping for %s\n", *waitDuration)
		time.Sleep(*waitDuration)
	}
}

// cleanup terminate the program clean
func cleanup(t int, quit chan<- bool) {
	fmt.Printf("cleaning up %d threads\n", t)
	for i := 0; i < t; i++ {
		quit <- true
	}
}

// work will start some routines to consume CPU
func work(n int, m float64, quit <-chan bool) {
	// start a goroutine that will not consume 100% of a CPU
	if m > 0 {
		go func() {
			for {
				select {
				case <-quit:
					return
				default:
					start := time.Now().UnixNano() / int64(time.Millisecond)
					for {
						end := time.Now().UnixNano() / int64(time.Millisecond)
						if end-start > int64(m*100) {
							// sleep for the remaining percent of a CPU times 100 milliseconds
							b := (1 - m) * 100

							time.Sleep(time.Duration(b) * time.Millisecond)
							break
						}
					}

					// end := time.Now().UnixNano() / int64(time.Millisecond)
					// diff := end - start
					// log.Printf("Duration(ms): %d", diff)
				}
			}
		}()
	}
	// start routines that will consume all the full CPUs requested
	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
				case <-quit:
					return
				default:
				}
			}
		}()
	}
}
