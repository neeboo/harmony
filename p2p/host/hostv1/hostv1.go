package hostv1

import (
	"io"
	"net"
	"time"

	"github.com/harmony-one/harmony/log"
	"github.com/harmony-one/harmony/p2p"
)

// HostV1 is the version 1 p2p host, using direct socket call.
type HostV1 struct {
	self     p2p.Peer
	listener net.Listener
	quit     chan bool
}

// New creates a HostV1
func New(self p2p.Peer) *HostV1 {
	h := &HostV1{
		self: self,
		quit: make(chan bool, 1),
	}
	return h
}

// GetSelfPeer gets self peer
func (host *HostV1) GetSelfPeer() p2p.Peer {
	return host.self
}

// BindHandlerAndServe Version 0 p2p. Going to be deprecated.
func (host *HostV1) BindHandlerAndServe(handler p2p.StreamHandler) {
	port := host.self.Port
	addr := net.JoinHostPort(host.self.IP, port)
	var err error
	host.listener, err = net.Listen("tcp4", addr)
	if err != nil {
		log.Error("Socket listen port failed", "addr", addr, "err", err)
		return
	}
	if host.listener == nil {
		log.Error("Listen returned nil", "addr", addr)
		return
	}
	backoff := p2p.NewExpBackoff(250*time.Millisecond, 15*time.Second, 2.0)
	for { // Keep listening
		select {
		case <-host.quit:
			return
		default:
			{
				conn, err := host.listener.Accept()
				if err != nil {
					log.Error("Error listening on port.", "port", port,
						"err", err)
					backoff.Sleep()
					continue
				}
				// log.Debug("Received New connection", "local", conn.LocalAddr(), "remote", conn.RemoteAddr())
				go handler(conn)
			}
		}
	}
}

// SendMessage sends message to peer
func (host *HostV1) SendMessage(peer p2p.Peer, message []byte) (err error) {
	addr := net.JoinHostPort(peer.IP, peer.Port)
	conn, err := net.Dial("tcp", addr)
	// log.Debug("Dial from local to remote", "localID", net.JoinHostPort(host.self.IP, host.self.Port), "local", conn.LocalAddr(), "remote", addr)

	if err != nil {
		log.Warn("Dial() failed", "from", net.JoinHostPort(host.self.IP, host.self.Port), "to", addr, "error", err)
		return
	}
	defer conn.Close()

	nw, err := conn.Write(message)
	if err != nil {
		log.Warn("Write() failed", "addr", conn.RemoteAddr(), "error", err)
		return
	}
	if nw < len(message) {
		log.Warn("Write() returned short count",
			"addr", conn.RemoteAddr(), "actual", nw, "expected", len(message))
		return io.ErrShortWrite
	}

	// No ack (reply) message from the receiver for now.
	return
}

// Close closes the host
func (host *HostV1) Close() error {
	host.quit <- true
	return host.listener.Close()
}