package rsync

import (
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"time"

	"github.com/armadillica/flamenco-sync-server/servertools"
	log "github.com/sirupsen/logrus"
)

// Server manages multiple rsync daemons.
type Server struct {
	closable servertools.Closable
}

// CreateServer creates a new rsync server, which in turn can create
// an rsync daemon for incoming connections.
func CreateServer() *Server {
	log.Info("Starting rsync server")

	return &Server{
		closable: servertools.MakeClosable(),
	}
}

// Shutdown waits until all running daemons have stopped.
func (rss *Server) Shutdown() {
	log.Info("rsync server: shutting down, waiting for rsync daemons to stop")
	rss.closable.ClosableCloseAndWait()
	log.Info("rsync server: shut down")
}

// StartDaemon starts a new daemon connected to the given network connection.
func (rss *Server) StartDaemon(conn net.Conn) {
	rss.closable.ClosableAdd(1)
	go rss.work(conn)
}

func (rss *Server) work(conn net.Conn) {
	defer rss.closable.ClosableDone()

	logfields := log.Fields{
		"remote_addr": conn.RemoteAddr(),
	}
	logger := log.WithFields(logfields)

	startTime := time.Now().UTC()
	defer func() {
		endTime := time.Now().UTC()
		duration := endTime.Sub(startTime)
		logger.WithField("duration", duration).Info("rsync daemon finished")
	}()
	defer rss.cleanup(conn)

	// If this is a straight TCP/IP connection we can pass it to rsync, otherwise we need
	// to make sure it is one, and since we don't need this in production (the SSL termination
	// is done somewhere else) I won't implement this now.
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		fmt.Fprint(conn, "@RSYNCD: 31.0\n")
		fmt.Fprint(conn, "@ERROR: not a TCP/IP connection, here be dragons.\n")
		logger.Fatal("not a TCP/IP connection, here be dragons.")
	}

	fileConn, err := tcpConn.File()
	if err != nil {
		panic(err)
	}

	// Start the RSync process, connecting it to the network connection.
	cmd := exec.Command("./rsync-server", "--daemon", "--config", "./rsyncd.conf")
	cmd.Stdin = fileConn
	// RSync will close its stdout and stderr file descriptors because stdin is a socket.
	logger.Debug("rsync daemon: starting")

	if err := cmd.Run(); err != nil {
		rss.logCmdError(logfields, err)
		return
	}
}

func (rss *Server) logCmdError(logfields log.Fields, err error) {
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

func (rss *Server) cleanup(conn net.Conn) {
	logger := log.WithFields(log.Fields{
		"remote_addr": conn.RemoteAddr(),
	})

	if err := conn.Close(); err != nil {
		logger.WithError(err).Warning("rsync daemon cleanup: unable to close connection")
	} else {
		logger.Debug("Connection closed")
	}
}
