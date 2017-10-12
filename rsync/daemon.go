package rsync

import (
	"net"
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// Manages a single daemon.
type rsyncDaemon struct {
	conn net.Conn
}

func createRsyncDaemon(conn net.Conn) *rsyncDaemon {
	daemon := rsyncDaemon{conn}
	return &daemon
}

func (rsd *rsyncDaemon) work() {
	defer rsd.cleanup()

	logfields := log.Fields{
		"remote_addr": rsd.conn.RemoteAddr(),
	}

	tcpConn, ok := rsd.conn.(*net.TCPConn)
	if !ok {
		panic("not TCP/IP")
	}
	fileConn, err := tcpConn.File()
	if err != nil {
		panic(err)
	}

	// Start the RSync process, connecting it to the network connection.
	cmd := exec.Command("/home/sybren/workspace/dpkg/rsync-3.1.1/rsync", "--daemon", "--config", "./rsyncd.conf", "--no-detach", "--verbose")
	cmd.Stdin = fileConn
	// RSync will close its stdout and stderr file descriptors because stdin is a socket.
	log.WithFields(logfields).Debug("rsync daemon: starting")

	if err := cmd.Run(); err != nil {
		rsd.logCmdError(logfields, err)
		return
	}
	log.WithFields(logfields).Info("rsync ran OK, closing connection")
}

func (rsd *rsyncDaemon) logCmdError(logfields log.Fields, err error) {
	logger := log.WithFields(logfields)

	if exitErr, ok := err.(*exec.ExitError); ok {
		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitStatus := status.ExitStatus()
			if exitStatus == 255 {
				logger.Warning("rsyncd exited due to a client-side error")
			} else {
				logger.WithField("exit_status", exitStatus).Warning("rsyncd exited with an error status")
			}
			return
		}
	}

	logger.WithError(err).Errorf("Error running rsyncd")
}

func (rsd *rsyncDaemon) cleanup() {
	logger := log.WithFields(log.Fields{
		"remote_addr": rsd.conn.RemoteAddr(),
	})

	if err := rsd.conn.Close(); err != nil {
		logger.WithError(err).Warning("rsync daemon cleanup: unable to close connection")
	} else {
		logger.Debug("Connection closed")
	}

	// TODO: remove this daemon from the server list of daemons
}
