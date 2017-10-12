package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	stdlog "log"

	"github.com/armadillica/flamenco-sync-server/httphandler"
	"github.com/armadillica/flamenco-sync-server/rsync"
	log "github.com/sirupsen/logrus"
)

const applicationVersion = "0.1-dev"
const applicationName = "Flamenco Sync Server"

var cliArgs struct {
	version bool
	verbose bool
	debug   bool
	listen  string
	tlsCert string
	tlsKey  string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.verbose, "verbose", false, "Enable info-level logging.")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.StringVar(&cliArgs.listen, "listen", "[::]:8084", "Address to listen on.")
	flag.StringVar(&cliArgs.tlsCert, "cert", "", "TLS certificate file.")
	flag.StringVar(&cliArgs.tlsKey, "key", "", "TLS key file.")
	flag.Parse()
}

func configLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := log.WarnLevel
	if cliArgs.debug {
		level = log.DebugLevel
	} else if cliArgs.verbose {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	stdlog.SetOutput(log.StandardLogger().Writer())
}

func logStartup() {
	level := log.GetLevel()
	defer log.SetLevel(level)

	log.SetLevel(log.InfoLevel)
	log.WithFields(log.Fields{
		"version": applicationVersion,
	}).Infof("Starting %s", applicationName)
}

func main() {
	parseCliArgs()
	if cliArgs.version {
		fmt.Println(applicationVersion)
		return
	}

	configLogging()
	logStartup()

	// Set some more or less sensible limits & timeouts.
	http.DefaultTransport = &http.Transport{
		MaxIdleConns:          100,
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       15 * time.Minute,
		ResponseHeaderTimeout: 15 * time.Second,
	}

	logFields := log.Fields{"listen": cliArgs.listen}
	rsyncServer := rsync.CreateServer()
	httpHandler := httphandler.CreateHTTPHandler(rsyncServer)

	var httpError error
	if cliArgs.tlsCert != "" {
		log.WithFields(logFields).Info("Starting HTTPS server")
		httpError = http.ListenAndServeTLS(cliArgs.listen, cliArgs.tlsCert, cliArgs.tlsKey, httpHandler)
	} else {
		log.WithFields(logFields).Info("Starting HTTP server")
		httpError = http.ListenAndServe(cliArgs.listen, httpHandler)
	}
	log.WithFields(logFields).WithError(httpError).Fatal("HTTP server failed")
}
