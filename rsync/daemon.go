package rsync

import (
	"bufio"
	"net"

	log "github.com/sirupsen/logrus"
)

// Manages a single daemon.
type rsyncDaemon struct {
	conn net.Conn
	brw  *bufio.ReadWriter
}

func (rsd *rsyncDaemon) work() {
	logger := log.WithFields(log.Fields{
		"remote_addr": rsd.conn.RemoteAddr(),
	})
	logger.Debug("rsync daemon: starting")

	rsd.conn.Write([]byte("je moeder"))
}
