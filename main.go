package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	stdlog "log"

	"github.com/armadillica/flamenco-manager/flamenco"
	"github.com/armadillica/flamenco-sync-server/httphandler"
	"github.com/armadillica/flamenco-sync-server/rsync"
	log "github.com/sirupsen/logrus"
)

const applicationVersion = "0.1-dev"
const applicationName = "Flamenco Sync Server"

// Components that make up the application
var rsyncServer *rsync.Server
var httpServer *http.Server

// Signalling channels
var shutdownComplete chan struct{}
var httpShutdownComplete chan struct{}

var cliArgs struct {
	version bool
	verbose bool
	debug   bool
	listen  string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.verbose, "verbose", false, "Enable info-level logging.")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.StringVar(&cliArgs.listen, "listen", "[::]:8084", "Address to listen on.")
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

func shutdown(signum os.Signal) {
	// Force shutdown after a bit longer than the HTTP server timeout.
	timeout := flamenco.TimeoutAfter(17 * time.Second)

	go func() {
		log.WithField("signal", signum).Info("Signal received, shutting down.")

		if httpServer != nil {
			log.Info("Shutting down HTTP server")
			// the Shutdown() function seems to hang sometime, even though the
			// main goroutine continues execution after ListenAndServe().
			go httpServer.Shutdown(context.Background())
			<-httpShutdownComplete
		} else {
			log.Warning("HTTP server was not even started yet")
		}

		rsyncServer.Shutdown()

		timeout <- false
	}()

	if <-timeout {
		log.Error("Shutdown forced, stopping process.")
		os.Exit(-2)
	}

	log.Warning("Shutdown complete, stopping process.")
	close(shutdownComplete)
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
	rsyncServer = rsync.CreateServer()
	httpHandler := httphandler.CreateHTTPHandler(rsyncServer)

	// Create the HTTP server before allowing the shutdown signal Handler
	// to exist. This prevents a race condition when Ctrl+C is pressed after
	// the http.Server is created, but before it is assigned to httpServer.
	httpServer = &http.Server{
		Addr:        cliArgs.listen,
		Handler:     httpHandler,
		ReadTimeout: 15 * time.Second,
	}

	shutdownComplete = make(chan struct{})
	httpShutdownComplete = make(chan struct{})

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for signum := range c {
			// Run the shutdown sequence in a goroutine, so that multiple Ctrl+C presses can be handled in parallel.
			go shutdown(signum)
		}
	}()

	log.WithFields(logFields).Info("Starting HTTP server")
	httpError := httpServer.ListenAndServe()
	if httpError != nil && httpError != http.ErrServerClosed {
		log.WithError(httpError).Error("HTTP server stopped")
	}
	close(httpShutdownComplete)

	log.Info("Waiting for shutdown to complete.")

	<-shutdownComplete
}
