package mocknet

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// stream implements network.Stream
type stream struct {
	env       Environment
	write     *io.PipeWriter
	read      *io.PipeReader
	conn      *conn

	reset  chan struct{}
	close  chan struct{}
	closed chan struct{}

	writeErr error

	protocol atomic.Value
	stat     network.Stat
}

var ErrReset error = errors.New("stream reset")
var ErrClosed error = errors.New("stream closed")

type transportObject struct {
	msg         []byte
	arrivalTime time.Duration
}

func NewStream(env Environment, w *io.PipeWriter, r *io.PipeReader, dir network.Direction) *stream {
	s := &stream{
		env:       env,
		read:      r,
		write:     w,
		reset:     make(chan struct{}, 1),
		close:     make(chan struct{}, 1),
		closed:    make(chan struct{}),
		stat:      network.Stat{Direction: dir},
	}

	return s
}

func (s *stream) Write(p []byte) (n int, err error) {
	l := s.conn.link
	delay := l.GetLatency() + l.RateLimit(len(p))

	// Copy it.
	cpy := make([]byte, len(p))
	copy(cpy, p)


	select {
	case <-s.closed: // bail out if we're closing.
		return 0, s.writeErr
	case sa := <-s.env.SynTask(delay):
		// TODO: instead of scheduling a task for delivery, the message could be buffered,
		//  and split in two racing tasks: schedule delivery when buffer is full, or when X time has passed
		// write this message.
		n, err = s.write.Write(cpy)
		sa <- TaskAck{}
		return
	}
}

func (s *stream) Protocol() protocol.ID {
	// Ignore type error. It means that the protocol is unset.
	p, _ := s.protocol.Load().(protocol.ID)
	return p
}

func (s *stream) Stat() network.Stat {
	return s.stat
}

func (s *stream) SetProtocol(proto protocol.ID) {
	s.protocol.Store(proto)
}

func (s *stream) Close() error {
	select {
	case s.close <- struct{}{}:
	default:
	}
	<-s.closed
	if s.writeErr != ErrClosed {
		return s.writeErr
	}
	return nil
}

func (s *stream) Reset() error {
	// Cancel any pending reads/writes with an error.
	s.write.CloseWithError(ErrReset)
	s.read.CloseWithError(ErrReset)

	select {
	case s.reset <- struct{}{}:
	default:
	}
	<-s.closed

	// No meaningful error case here.
	return nil
}

func (s *stream) teardown() {
	// at this point, no streams are writing.
	s.conn.removeStream(s)

	// Mark as closed.
	close(s.closed)

	s.conn.net.notifyAll(func(n network.Notifiee) {
		n.ClosedStream(s.conn.net, s)
	})
}

func (s *stream) Conn() network.Conn {
	return s.conn
}

func (s *stream) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "pipe", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (s *stream) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "pipe", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (s *stream) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "pipe", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (s *stream) Read(b []byte) (int, error) {
	return s.read.Read(b)
}

func (s *stream) resetWith(err error) {
	s.write.CloseWithError(err)
	s.read.CloseWithError(err)
	s.writeErr = err
}
