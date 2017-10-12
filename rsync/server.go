package rsync

import (
	"net"
	"os/exec"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// Server manages multiple rsync daemons.
type Server struct {
	daemons  sync.WaitGroup
	shutdown chan interface{}
}

// CreateServer creates a new rsync server, which in turn can create
// an rsync daemon for incoming connections.
func CreateServer() *Server {
	log.Info("Starting rsync server")

	return &Server{
		daemons:  sync.WaitGroup{},
		shutdown: make(chan interface{}),
	}
}

// Shutdown gracefully shuts down the server by refusing to create new daemons
// and waiting until all running daemons have stopped.
func (rss *Server) Shutdown() {
	log.Info("rsync server: shutting down, waiting for rsync daemons to stop")
	rss.daemons.Wait()
	log.Info("rsync server: shut down")
}

// StartDaemon starts a new daemon connected to the given network connection.
func (rss *Server) StartDaemon(conn net.Conn) {
	rss.daemons.Add(1)
	go rss.work(conn)
}

func (rss *Server) work(conn net.Conn) {
	defer rss.cleanup(conn)
	defer rss.daemons.Done()

	logfields := log.Fields{
		"remote_addr": conn.RemoteAddr(),
	}

	tcpConn, ok := conn.(*net.TCPConn)
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
		rss.logCmdError(logfields, err)
		return
	}
	log.WithFields(logfields).Info("rsync ran OK, closing connection")
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

	// TODO: remove this daemon from the server list of daemons
}
