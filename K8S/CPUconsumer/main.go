package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	version        = "no version set"
	debug          = flag.Bool("debug", false, "display debugs")
	displayVersion = flag.Bool("version", false, "Show version and quit")
	logLevel       = flag.String("logLevel", "warn", "log level from debug, info, warning, error. When debug, genetate 100% Tracing")
	waitCPU        = flag.Float64("waitCPU", 0, "how many CPU to use in the wait phase(override GOMAXPROCS)")
	workCPU        = flag.Float64("workCPU", 0, "how many CPU to use in the work phase(override GOMAXPROCS)")
	waitDuration   = flag.Duration("waitDuration", time.Duration(30)*time.Second, "how long to wait before work, in Golang Duration")
	workDuration   = flag.Duration("workDuration", time.Duration(10)*time.Second, "how long to work, then wait, in Golang Duration")
	httpPort       = flag.String("httpport", "7789", "port to bind for HTTP server used for Health and Metrics")
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

	if *workCPU < *waitCPU {
		fmt.Println("workCPU must be higher than waitCPU")
		os.Exit(256)
	}

	// Logrus
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
	logger := logrus.New()
	log := logger.WithFields(logrus.Fields{
		"application": "cpuconsumer",
	})
	// healthz basic
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		m := map[string]interface{}{"version": version, "status": "OK"}

		b, err := json.Marshal(m)
		if err != nil {
			http.Error(w, "no valid point for this device_id", 500)
			return
		}

		w.Write(b)
	})

	// prometheus metrics
	http.Handle("/metrics", promhttp.Handler())

	// listen on the HTTP port for metrics
	go func() {
		log.Warn(fmt.Sprintf("listening HTTP (metrics & health) on %v", *httpPort))
		log.Warn(http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil))
	}()

	// values for the working period
	n, m, t := setCPU(*workCPU)
	runtime.GOMAXPROCS(t)

	// values for the waiting period
	a, s, v := setCPU(*waitCPU)

	// quit channel is used to force exit of all threads
	quit := make(chan bool, t)

	// Grab SIGINT and exit all threads
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(t int) {
		<-c
		log.Info("got SIGINT, terminating")
		cleanup(t, quit)
		os.Exit(1)
	}(t)

	for {
		// start all consumer threads
		log.Info(fmt.Sprintf("starting %d+%.3f (%d total) working threads\n", n, m, t))
		ctx, cancel := context.WithCancel(context.Background())
		work(ctx, n, m)

		log.Info(fmt.Sprintf("working for %s\n", *workDuration))
		time.Sleep(*workDuration)

		// for i := 0; i < t; i++ {
		// 	fmt.Println("sending quit signal for full")
		// 	quit <- true
		// 	time.Sleep(time.Duration(10) * time.Millisecond)
		// }
		cancel()
		log.Info(fmt.Sprintf("Job done, starting %d+%.3f (%d total) waiting threads for %s\n", a, s, v, *waitDuration))

		if *waitCPU > 0 {
			ctx, cancel := context.WithCancel(context.Background())
			work(ctx, a, s)

			time.Sleep(*waitDuration)

			// for i := 0; i < v; i++ {
			// 	fmt.Println("sending quit signal for fraction")
			// 	quit <- true
			// 	time.Sleep(time.Duration(10) * time.Millisecond)
			// }
			cancel()
		} else {
			time.Sleep(*waitDuration)
		}
	}
}

// cleanup terminate the program clean
func cleanup(t int, quit chan<- bool) {
	fmt.Printf("cleaning up %d threads\n", t)
	for i := 0; i < t; i++ {
		quit <- true
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

// work will start some routines to consume CPU
func work(ctx context.Context, n int, m float64) {
	// start a goroutine that will not consume 100% of a CPU
	if m > 0 {
		PromThreadGauge.Inc()
		go func() {
			fmt.Println("fraction thread started")
			dummy := 0
			for {
				select {
				case <-ctx.Done():
					PromThreadGauge.Dec()
					fmt.Println("fraction thread stopped")
					return
				default:
					start := time.Now().UnixNano() / int64(time.Millisecond)
					for {
						end := time.Now().UnixNano() / int64(time.Millisecond)
						if end-start > int64(m*100) {
							// sleep for the remaining percent of a CPU times 100 milliseconds
							b := 100 - m*100

							time.Sleep(time.Duration(b) * time.Millisecond)
							break
						}
					}
					dummy++

					// end := time.Now().UnixNano() / int64(time.Millisecond)
					// diff := end - start
					// log.Printf("Duration(ms): %d", diff)
				}
			}
		}()
	}
	// start routines that will consume all the full CPUs requested
	for i := 0; i < n; i++ {
		PromThreadGauge.Inc()
		go func() {
			fmt.Println("full thread started")
			dummy := 0
			for {
				select {
				case <-ctx.Done():
					PromThreadGauge.Dec()
					fmt.Println("full thread stopped")
					return
				default:
					dummy++
				}
			}
		}()
	}
}

// setCPU split the number of milicores into full cores + subsides
func setCPU(c float64) (n int, m float64, t int) {
	// Set the number of full CPU
	n = int(c)

	// CPU mili cores
	m = math.Mod(c, 1)

	// t is the total number of full routines to start

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
	return n, m, t
}
