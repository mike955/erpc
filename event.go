package erpc

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/mike955/erpc/internal"
	"golang.org/x/sys/unix"
)

var (
	defaultName    = "eventloop-rpc"
	defaultLoopNum = runtime.NumCPU()
	defaultNetWork = "tcp"
	defaultAddr    = "0.0.0.0:9000"
)

type EventLoopOption func(e *EventLoop)

type listener struct {
	fd   int
	addr net.Addr
	ln   net.Listener
	file *os.File
}

type Handler func(req []byte) (resp []byte)

var defaultHandler Handler = func(req []byte) (resp []byte) {
	resp = req
	return
}

type EventLoop struct {
	name    string
	numLoop int
	network string
	addr    string

	lis     *listener
	loops   []*loop
	handler Handler
	codec   Codec
	lb      *balance

	cond *sync.Cond
	wg   *sync.WaitGroup
}

func Name(name string) EventLoopOption {
	return func(e *EventLoop) {
		e.name = name
	}
}

func LoopNum(num int) EventLoopOption {
	if num <= 0 {
		num = 1
	}
	return func(e *EventLoop) {
		e.numLoop = num
	}
}

func Network(network string) EventLoopOption {
	return func(e *EventLoop) {
		e.network = network
	}
}

func Address(addr string) EventLoopOption {
	return func(e *EventLoop) {
		e.addr = addr
	}
}

func Codecer(codec Codec) EventLoopOption {
	return func(e *EventLoop) {
		e.codec = codec
	}
}

func LoadBalance(balance BalanceType) EventLoopOption {
	return func(e *EventLoop) {
		e.lb = NewBalance(balance)
	}
}

func NewServer(app string, opts ...EventLoopOption) (eventLoop *EventLoop) {
	eventLoop = &EventLoop{
		name:    defaultName,
		network: defaultNetWork,
		addr:    defaultAddr,
		codec:   dc,
		handler: defaultHandler,
		wg:      &sync.WaitGroup{},
		lb:      NewBalance(RoundRobin),
	}
	for _, o := range opts {
		o(eventLoop)
	}
	return eventLoop
}

func (e *EventLoop) RegisterHandler(handler Handler) {
	e.handler = handler
	return
}

func (e *EventLoop) Serve() (err error) {
	lis, err := e.createListener()
	if err != nil {
		panic(err)
	}
	e.lis = lis
	err = e.createLoop()
	if err != nil {
		panic(err)
	}
	err = e.serve()
	return
}

func (e *EventLoop) createListener() (lis *listener, err error) {
	ln, err := net.Listen(e.network, e.addr)
	if err != nil {
		return
	}
	lis = &listener{}
	lis.ln = ln
	lis.addr = ln.Addr()
	switch netln := lis.ln.(type) {
	case *net.TCPListener:
		lis.file, err = netln.File()
	case *net.UnixListener:
		lis.file, err = netln.File()
	default:
		err = fmt.Errorf("net(%s) type is not support", netln)
	}
	if err != nil {
		return
	}
	lis.fd = int(lis.file.Fd())
	return
}

func (e *EventLoop) createLoop() (err error) {
	for i := 0; i < e.numLoop; i++ {
		pool, err := internal.CreatePoll()
		if err != nil {
			return err
		}
		l := &loop{
			idx:     int32(i),
			poll:    pool,
			packet:  make([]byte, 0xFFFF),
			fdConns: make(map[int]*conn),
			codec:   e.codec,
			handler: e.handler,
		}
		l.poll.AddRead(e.lis.fd)
		e.loops = append(e.loops, l)
	}
	return
}

func (e *EventLoop) serve() (err error) {
	e.wg.Add(len(e.loops))
	for _, l := range e.loops {
		go e.loopRun(l)
	}

	mainPoll, err := internal.CreatePoll()
	if err != nil {
		return
	}
	if err = mainPoll.AddRead(e.lis.fd); err != nil {
		return
	}
	mainPoll.Wait(func(fd int, filter int16) (err error) {
		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			if err == syscall.EAGAIN {
				return nil
			}
			return err
		}
		if err = unix.SetNonblock(nfd, true); err != nil {
			err = os.NewSyscallError("set fs not block error: ", err)
			return
		}
		conn := &conn{
			sa:    sa,
			fd:    nfd,
			laddr: e.lis.addr,
			radd:  e.socketAddrToNetAdr(sa),
		}
		loop := e.selectLoop()
		loop.addFd(nfd, conn)
		e.lb.addBalance(loop.idx, conn.fd)
		return nil
	})

	return
}

func (e *EventLoop) selectLoop() (loop *loop) {
	load := e.lb.next()
	if e.loops[load] == nil {
		panic("select loop error")
	}
	loop = e.loops[load]
	return
}

func (e *EventLoop) loopRun(l *loop) {
	defer func() {
		e.wg.Done()
		l.closeAllConns()
	}()
	l.poll.Wait(func(fd int, filter int16) (err error) {
		if conn, exist := l.fdConns[fd]; exist {
			switch filter {
			case unix.EVFILT_READ, unix.EVFILT_WRITE:
				err = l.loopRead(fd)
			}
			if err != nil {
				l.closeConn(conn.fd)
				e.lb.removeBalance(l.idx, conn.fd)
			}
		}
		return
	})
}

func (e *EventLoop) socketAddrToNetAdr(sa unix.Sockaddr) (addr net.Addr) {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		addr = &net.TCPAddr{
			IP:   sa.Addr[0:],
			Port: sa.Port,
		}
	case *unix.SockaddrInet6:
		v6Addr := &net.TCPAddr{
			IP:   sa.Addr[0:],
			Port: sa.Port,
		}
		if sa.ZoneId != 0 {
			if ifi, err := net.InterfaceByIndex(int(sa.ZoneId)); err == nil {
				v6Addr.Zone = ifi.Name
			}
		}
		addr = v6Addr
	case *unix.SockaddrUnix:
		return &net.UnixAddr{Name: sa.Name, Net: "unix"}
	}
	return nil
}
