package main

import (
	"net/http"
	"time"

	"github.com/armadillica/flamenco-sync-server/rsync"
	log "github.com/sirupsen/logrus"
)

type httpHandler struct {
	rsyncServer *rsync.Server
}

func createHTTPHandler(rsyncServer *rsync.Server) *httpHandler {
	if rsyncServer == nil {
		panic("rsyncServer == nil")
	}
	return &httpHandler{rsyncServer}
}

// ServeHTTP performs auth and then starts and defers to an rsync daemon.
func (ss *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := 0
	logger := log.WithFields(log.Fields{
		"remote_addr": r.RemoteAddr,
		"method":      r.Method,
		"url":         r.URL.String(),
	})
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		logger = logger.WithField("x_forwarded_for", xff)
	}

	startTime := time.Now().UTC()
	defer func() {
		endTime := time.Now().UTC()
		fields := log.Fields{"duration": endTime.Sub(startTime)}
		if status != 0 {
			fields["status"] = status
		}
		logger.WithFields(fields).Info("Request handled")
	}()

	if r.Method != "RSYNC" {
		status = http.StatusMethodNotAllowed
		w.WriteHeader(status)
		return
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		logger.Error("httphandler: response does not implement http.Hijacker")
		status = http.StatusInternalServerError
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Transfer-Encoding", "identity")
	w.Header().Set("Upgrade", "rsync")
	w.Header().Set("Connection", "upgrade")
	w.WriteHeader(http.StatusSwitchingProtocols)

	netConn, brw, err := h.Hijack()
	if err != nil {
		logger.WithError(err).Error("httphandler: unable to hijack HTTP connection")
		status = http.StatusInternalServerError
		w.WriteHeader(status)
		return
	}
	if brw.Reader.Buffered() > 0 {
		netConn.Close()
		logger.Error("httphandler: client sent data before handshake is complete")
		return
	}

	logger.Debug("Hijacked HTTP connection")
	ss.rsyncServer.StartDaemon(netConn, brw)
}
