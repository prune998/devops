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
	"strconv"
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
	numCPU         = strconv.Itoa(runtime.NumCPU()) // Keep track on the number of CPU as set by GOMAXPROCS
	numMaxProcs    = strconv.Itoa(runtime.GOMAXPROCS(-1))
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
		log.Info(fmt.Sprintf("listening HTTP (metrics & health) on %v", *httpPort))
		log.Warn(http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil))
	}()

	log.Info(fmt.Sprintf("app started on a %s CPU server with MAXPROCS %s", numCPU, numMaxProcs))

	// values for the working period
	n, m, t := setCPU(*workCPU)
	// runtime.GOMAXPROCS(t) // we don't override the GOMAXPROCS as we set it in the env

	// values for the waiting period
	a, s, v := setCPU(*waitCPU)

	// main context to cancel when we get a stop signal
	ctx, cancel := context.WithCancel(context.Background())

	// Grab SIGINT and exit all threads
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("got SIGINT, terminating")
		cancel()
		time.Sleep(time.Duration(10) * time.Millisecond)
		os.Exit(1)
	}()

	for {
		// start all consumer threads
		log.Info(fmt.Sprintf("starting %d+%.3f (%d total) working threads", n, m, t))
		ctx1, cancel1 := context.WithCancel(ctx)
		work(ctx1, n, m, log)

		log.Info(fmt.Sprintf("working for %s", *workDuration))
		time.Sleep(*workDuration)

		cancel1()
		log.Info(fmt.Sprintf("Job done, starting %d+%.3f (%d total) waiting threads for %s", a, s, v, *waitDuration))

		if v > 0 {
			ctx2, cancel2 := context.WithCancel(ctx)
			work(ctx2, a, s, log)

			log.Info(fmt.Sprintf("working for %s", *waitDuration))
			time.Sleep(*waitDuration)
			cancel2()

		} else {
			time.Sleep(*waitDuration)
		}
	}
}

// work will start some routines to consume CPU
func work(ctx context.Context, n int, m float64, log *logrus.Entry) {
	callingTime := time.Now().UnixNano()

	// start a goroutine that will not consume 100% of a CPU
	if m > 0 {
		PromThreadGauge.Inc()
		go func(jobID int64) {
			log.Info(fmt.Sprintf("fraction thread started (%d)", jobID))
			dummy := 0
			for {
				select {
				case <-ctx.Done():
					PromThreadGauge.Dec()
					log.Info(fmt.Sprintf("fraction thread stopped (%d)", jobID))
					return
				default:
					start := time.Now().UnixNano() / int64(time.Millisecond)
					// This for loop represent the work to be done in the % of the CPU allocated
					for {
						end := time.Now().UnixNano() / int64(time.Millisecond)
						if end-start > int64(m*100) {
							// sleep for the remaining percent of a CPU times 100 milliseconds
							b := 100 - m*100

							time.Sleep(time.Duration(b) * time.Millisecond)
							break
						}
						PromOpsCounter.WithLabelValues(numMaxProcs).Inc()
						dummy++
						if dummy%1000000000 == 0 {
							log.Info(fmt.Sprintf("fraction thread still working (%d)", jobID))
							dummy = 1
						}
					}
				}
			}
		}(callingTime + int64(n))
	}
	// start routines that will consume all the full CPUs requested
	// for i := 0; i < n; i++ {
	// 	PromThreadGauge.Inc()
	// 	go func(jobID int64) {
	// 		log.Info(fmt.Sprintf("full thread started (%d)", jobID))
	// 		dummy := 0
	// 		for {
	// 			select {
	// 			case <-ctx.Done():
	// 				PromThreadGauge.Dec()
	// 				log.Info(fmt.Sprintf("full thread stopped (%d)", jobID))
	// 				return
	// 			default:
	// 				PromOpsCounter.WithLabelValues(numMaxProcs).Inc()
	// 				dummy++
	// 				if dummy%10000000000 == 0 {
	// 					log.Info(fmt.Sprintf("full thread still working (%d)", jobID))
	// 					dummy = 1
	// 				}
	// 			}
	// 		}
	// 	}(callingTime + int64(i))
	// }

	// start a full thread but still cap the usage to 100ms
	for i := 0; i < n; i++ {
		PromThreadGauge.Inc()
		go func(jobID int64) {
			log.Info(fmt.Sprintf("full thread started (%d)", jobID))
			dummy := 0
			for {
				select {
				case <-ctx.Done():
					PromThreadGauge.Dec()
					log.Info(fmt.Sprintf("full thread stopped (%d)", jobID))
					return
				default:
					start := time.Now().UnixNano() / int64(time.Millisecond)
					// This for loop represent the work to be done in the % of the CPU allocated
					for {
						end := time.Now().UnixNano() / int64(time.Millisecond)
						if end-start > int64(100) {
							// sleep few ms to compensate for over-usage
							b := 100 - end - start

							time.Sleep(time.Duration(b) * time.Millisecond)
							break
						}
						PromOpsCounter.WithLabelValues(numMaxProcs).Inc()
						dummy++
						if dummy%1000000000 == 0 {
							log.Info(fmt.Sprintf("full thread still working (%d)", jobID))
							dummy = 1
						}
					}
				}
			}
		}(callingTime + int64(i))
	}
}

// setCPU split the number of milicores into full cores + fraction
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
		// if CPU is round, set total number of routines to the number of full CPU
		t = n
	} else {
		// if CPU has a fraction, set total number of routines to the number of full CPU + 1
		t = n + 1
	}
	return n, m, t
}
