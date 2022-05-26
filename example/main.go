package main

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type listener struct {
	fd   int
	addr net.Addr
	ln   net.Listener
	file *os.File
}

type Poll struct {
	fd      int
	changes []syscall.Kevent_t
}

type conn struct {
	fd     int              // file descriptor
	out    []byte           // write buffer
	sa     syscall.Sockaddr // remote socket address
	reuse  bool             // should reuse input buffer
	opened bool             // connection opened event fired
	// action     Action           // next user action
	ctx        interface{} // user-defined context
	addrIndex  int         // index of listening address
	localAddr  net.Addr    // local addre
	remoteAddr net.Addr    // remote addr
}

var fdConns = make(map[int]*conn)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	lis := &listener{}
	lis.ln = ln
	lis.addr = ln.Addr()

	switch netln := lis.ln.(type) {
	case *net.TCPListener:
		lis.file, err = netln.File()
	case *net.UnixListener:
		lis.file, err = netln.File()
	}
	if err != nil {
		panic(err)
	}
	lis.fd = int(lis.file.Fd())

	// err = syscall.SetNonblock(lis.fd, true)
	// if err != nil {
	// 	panic(err)
	// }
	fd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	poll := new(Poll)
	poll.fd = fd
	_, err = syscall.Kevent(poll.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}

	poll.changes = append(poll.changes, syscall.Kevent_t{
		Ident:  uint64(lis.fd),
		Flags:  syscall.EV_ADD,
		Filter: syscall.EVFILT_READ,
	})

	events := make([]syscall.Kevent_t, 10)
	for {
		n, err := syscall.Kevent(poll.fd, poll.changes, events, nil)
		if err != nil && err != syscall.EINTR {
			panic(err)
		}
		poll.changes = poll.changes[:0]
		for i := 0; i < n; i++ {
			if fd := int(events[i].Ident); fd != 0 {
				handle(fd, poll)
			}
		}
	}
}

func handle(fd int, poll *Poll) error {

	c := fdConns[fd]
	// time.Sleep(time.Minute * 1)
	switch {
	case c == nil:
		return loopAccept(fd, poll)
	// case !c.opened:
	case len(c.out) > 0:
		return loopWrite(fd, poll, c)
	default:
		loopRead(c, poll)
	}
	return nil
}

func loopRead(c *conn, poll *Poll) error {
	var in []byte
	var packet = make([]byte, 0xFFFF)
	n, err := syscall.Read(c.fd, packet)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		delete(fdConns, c.fd)
		syscall.Close(c.fd)
		return nil
	}
	in = packet[:n]
	fmt.Println(string(in))
	c.out = append(c.out[:0], in...)
	if len(c.out) != 0 {
		poll.changes = append(poll.changes, syscall.Kevent_t{
			Ident: uint64(c.fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE,
		})
	}
	return nil
}

func loopAccept(fd int, poll *Poll) error {
	nfd, sa, err := syscall.Accept(fd)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	if err := unix.SetNonblock(nfd, true); err != nil {
		return err
	}
	c := &conn{fd: nfd, sa: sa}
	c.out = nil
	fdConns[c.fd] = c
	poll.changes = append(poll.changes,
		syscall.Kevent_t{
			Ident: uint64(c.fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_READ,
		},
		syscall.Kevent_t{
			Ident: uint64(c.fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE,
		},
	)
	return nil
}

func loopWrite(fd int, poll *Poll, c *conn) error {
	n, err := syscall.Write(fd, c.out)
	if err != nil {
		if err != syscall.EAGAIN {
			return nil
		}
	}
	c.out = c.out[n:]
	return nil
}
