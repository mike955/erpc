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
	defaultName       = "eventloop-rpc"
	defaultLoopNum    = runtime.NumCPU()
	defaultNetWork    = "tcp"
	defaultAddr       = "127.0.0.1:9000"
	defaultBufferSize = 0xFFFF
	defaultPollSize   = 128

	defaultCallback CallbackHandler = func(req []byte) (resp []byte) {
		resp = req
		return
	}
)

type EventLoopOption func(e *EventLoop)

type listener struct {
	fd   int
	addr net.Addr
	ln   net.Listener
	file *os.File
}

type CallbackHandler func(req []byte) (resp []byte)

type EventLoop struct {
	name       string
	numLoop    int
	network    string
	addr       string
	bufferSize int
	pollSize   int

	lis      *listener
	loops    []*loop
	callback CallbackHandler
	lb       *balance

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

func LoadBalance(balance BalanceType) EventLoopOption {
	return func(e *EventLoop) {
		e.lb = newBalance(balance)
	}
}

func BufferSize(size int) EventLoopOption {
	return func(e *EventLoop) {
		e.bufferSize = size
	}
}

func PollSize(size int) EventLoopOption {
	return func(e *EventLoop) {
		e.pollSize = size
	}
}

func NewServer(app string, opts ...EventLoopOption) (eventLoop *EventLoop) {
	eventLoop = &EventLoop{
		name:       defaultName,
		network:    defaultNetWork,
		addr:       defaultAddr,
		bufferSize: defaultBufferSize,
		pollSize:   defaultPollSize,
		callback:   defaultCallback,
		wg:         &sync.WaitGroup{},
		lb:         newBalance(RoundRobin),
	}
	for _, o := range opts {
		o(eventLoop)
	}
	return eventLoop
}

func (e *EventLoop) RegisterCallback(callback CallbackHandler) {
	e.callback = callback
	return
}

func (e *EventLoop) Serve() (err error) {
	lis, err := e.createListener()
	if err != nil {
		return err
	}
	e.lis = lis
	err = e.createLoop()
	if err != nil {
		return err
	}
	err = e.serve()
	return
}

func (e *EventLoop) Stop() (err error) {
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
		pool, err := internal.CreatePoll(e.pollSize)
		if err != nil {
			return err
		}
		l := &loop{
			idx:      int32(i),
			poll:     pool,
			packet:   make([]byte, e.bufferSize),
			fdConns:  make(map[int]*conn),
			callback: e.callback,
		}
		l.poll.AddRead(e.lis.fd)
		e.loops = append(e.loops, l)
	}
	return
}

func (e *EventLoop) serve() (err error) {
	e.wg.Add(len(e.loops))
	for _, l := range e.loops {
		go e.runLoop(l)
	}

	mainPoll, err := internal.CreatePoll(e.pollSize)
	if err != nil {
		return
	}
	if err = mainPoll.AddRead(e.lis.fd); err != nil {
		return
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		err = mainPoll.Wait(func(fd int, filter int16) (err error) {
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
			return
		})
	}()
	e.wg.Wait()
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

func (e *EventLoop) runLoop(l *loop) {
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
