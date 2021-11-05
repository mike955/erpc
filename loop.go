package erpc

import (
	"net"
	"syscall"

	"github.com/mike955/erpc/internal"
	"golang.org/x/sys/unix"
)

type conn struct {
	idx   int
	sa    unix.Sockaddr
	fd    int
	data  []byte
	laddr net.Addr
	radd  net.Addr
}

type loop struct {
	idx     int32
	poll    *internal.Poll
	packet  []byte
	fdConns map[int]*conn
	handler Handler
	codec   Codec
}

func (l *loop) loopRead(fd int) (err error) {
	conn := l.fdConns[fd]
	n, err := unix.Read(conn.fd, l.packet)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	var in = l.packet[:n]
	resp := l.handler(in)
	if len(resp) != 0 {
		unix.Write(conn.fd, resp)
	}
	return nil
}

func (l *loop) closeConn(fd int) (err error) {
	conn := l.fdConns[fd]
	err = unix.Close(conn.fd)
	if err == nil {
		delete(l.fdConns, fd)
	}
	return
}

func (l *loop) closeAllConns() {
	for fd, _ := range l.fdConns {
		l.closeConn(fd)
	}
	return
}

func (l *loop) loopWrite(fd int, c *conn) {

}

func (l *loop) addFd(fd int, conn *conn) {
	l.poll.AddReadWrite(fd)
	l.fdConns[fd] = conn
}
