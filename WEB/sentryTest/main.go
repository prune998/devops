package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/namsral/flag"
)

var (
	version        = "no version set"
	displayVersion = flag.Bool("version", false, "Show version and quit")
	DSN            = flag.String("DSN", "http://localhost:3000", "the Sentry DSN (full URL) like https://<token>@<URL>.ingest.sentry.io/<project>")
)

func printVersion() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("App Version: %v\n", version)
}

type devNullTransport struct{}

func (t *devNullTransport) Configure(options sentry.ClientOptions) {
	dsn, _ := sentry.NewDsn(options.Dsn)
	fmt.Println()
	fmt.Println("Store Endpoint:", dsn.GetAPIURL())
	fmt.Println("Headers:", dsn.RequestHeaders())
	fmt.Println()
}

func main() {

	flag.Parse()
	if *displayVersion {
		printVersion()
		os.Exit(0)
	}

	// Init Sentry connector
	err := sentry.Init(sentry.ClientOptions{
		Dsn: *DSN,
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
		SampleRate:       1,
		Debug:            true,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if ex, ok := hint.OriginalException.(CustomComplexError); ok {
				event.Message = event.Message + " - " + ex.GimmeMoreData()
			}

			fmt.Printf("%s\n\n", prettyPrint(event))

			return event
		},
		BeforeBreadcrumb: func(breadcrumb *sentry.Breadcrumb, _ *sentry.BreadcrumbHint) *sentry.Breadcrumb {
			if breadcrumb.Message == "Random breadcrumb 3" {
				breadcrumb.Message = "Not so random breadcrumb 3"
			}

			fmt.Printf("%s\n\n", prettyPrint(breadcrumb))

			return breadcrumb
		},
		Environment: "dev",
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	dsn, err := sentry.NewDsn(*DSN)
	fmt.Printf("project %s, %v\n", dsn.GetProjectID(), err)

	beforeSend()
	configureScope()

	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	// this file actually does not exist and sentry will capture the error
	f, err := os.Open("filename.ext")
	if err != nil {
		sentry.CaptureException(err)
	}
	f.Close()
}

type CustomComplexError struct {
	Message      string
	AnswerToLife int
}

func (e CustomComplexError) Error() string {
	return "CustomComplexError: " + e.Message
}

func (e CustomComplexError) GimmeMoreData() string {
	return strconv.Itoa(e.AnswerToLife)
}

func prettyPrint(v interface{}) string {
	pp, _ := json.MarshalIndent(v, "", "  ")
	return string(pp)
}

func beforeSend() {
	sentry.CaptureMessage("Drop me!")
}

func configureScope() {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetExtra("oristhis", "justfantasy")
		scope.SetTag("isthis", "reallife")
		scope.SetLevel(sentry.LevelFatal)
		scope.SetUser(sentry.User{
			ID: "1337",
		})
	})
}
