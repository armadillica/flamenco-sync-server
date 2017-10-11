package rsync

import (
	"bufio"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Server manages multiple rsync daemons.
type Server struct {
	daemons []*rsyncDaemon
	mutex   sync.Mutex
}

// CreateServer creates a new rsync server, which in turn can create
// an rsync daemon for incoming connections.
func CreateServer() *Server {
	log.Info("Starting rsync server")

	return &Server{
		daemons: make([]*rsyncDaemon, 0),
		mutex:   sync.Mutex{},
	}
}

// StartDaemon starts a new daemon connected to the given network connection.
func (rss *Server) StartDaemon(conn net.Conn, brw *bufio.ReadWriter) {
	rss.mutex.Lock()
	defer rss.mutex.Unlock()

	daemon := rsyncDaemon{conn, brw}
	rss.daemons = append(rss.daemons, &daemon)

	go daemon.work()
}
