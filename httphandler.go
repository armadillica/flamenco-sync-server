package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type syncServer struct{}

func createSyncServer() *syncServer {
	return &syncServer{}
}

// ServeHTTP performs auth and then starts and defers to an rsync daemon.
func (ss *syncServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := 0
	fields := log.Fields{
		"remote_addr": r.RemoteAddr,
		"method":      r.Method,
		"url":         r.URL.String(),
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		fields["x_forwarded_for"] = xff
	}

	startTime := time.Now().UTC()
	defer func() {
		endTime := time.Now().UTC()
		fields["duration"] = endTime.Sub(startTime)
		if status != 0 {
			fields["status"] = status
		}
		log.WithFields(fields).Info("Request handled")
	}()

	if r.Method != "RSYNC" {
		status = http.StatusMethodNotAllowed
		w.WriteHeader(status)
		return
	}

	if r.Header.Get("Upgrade") != "websocket" {
		log.WithFields(fields).Warning("No websocket, no request")
		status = http.StatusNotImplemented
		w.WriteHeader(status)
		return
	}

	// All our checks were fine, so now we can defer RSync to do the actual work.
	status = http.StatusOK
	w.WriteHeader(status)
	fmt.Fprint(w, "Upgrading to RSYNC protocol")
}
