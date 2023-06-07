package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/namsral/flag"
)

// time bucket for throttling computation, in miliseconds
const bucket = 100

var (
	version        = "no version set"
	displayVersion = flag.Bool("version", false, "Show version and quit")
	logLevel       = flag.String("logLevel", "warn", "log level from debug, info, warning, error. When debug, genetate 100% Tracing")
	numCPU         = flag.Float64("numCPU", 0, "how many CPU to use (override GOMAXPROCS)")
	testDuration   = flag.Duration("testDuration", time.Duration(100)*time.Second, "how long to run the test, in Golang Duration")
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

	// if not set, use all the CPUs as discovered by the system
	if n == 0 && m == 0 {
		n = runtime.NumCPU()
	} else if m == 0 {
		// if CPU is round, set GOMAXPROCS
		runtime.GOMAXPROCS(n)
	} else {
		runtime.GOMAXPROCS(n + 1)

	}
	fmt.Printf("Starting with GOMAXPROCS=%d and %f cores workload\n", n, m)

	quit := make(chan bool)

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
		fmt.Printf("%d,", i)
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

	time.Sleep(*testDuration)
	for i := 0; i < n+1; i++ {
		quit <- true
	}
}
