package main

import (
	"flag"
	"log/slog"

	"github.com/prune998/devops/GOLANG/boilerplate/logging"
)

var (
	myURL    = flag.String("url", "https://www.google.com", "URL to the service")
	extended = flag.Bool("extended", false, "if true, do extended queries")

	logLevel  = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none")
	logFormat = flag.String("logFormat", "json", "one of JSON, PLAIN, NONE")

	appNAme = "boilerplate"
)

func main() {
	flag.Parse()
	logging.InitDefaultLogger()
	logging.SetGlobalLogLevel(*logLevel)
	logging.SetGlobalFormat(*logFormat)
	slog.SetDefault(slog.With("app", appNAme))

	slog.Info("log level set", "level", *logLevel, "format", *logFormat)
}
