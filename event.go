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
	defaultName    = "erpc-event-loop"
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

type loop struct {
	idx     int
	poll    *internal.Poll
	packet  []byte
	fdConns map[int]*conn
	count   int32
	// handler Hanlder
}

type conn struct {
	idx   int
	sa    unix.Sockaddr
	fd    int
	data  []byte
	laddr net.Addr
	radd  net.Addr
}

type EventLoop struct {
	name    string
	numLoop int
	network string
	addr    string

	lis   *listener
	loops []*loop
	codec Codec

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

func NewServer(app string, opts ...EventLoopOption) (eventLoop *EventLoop) {
	eventLoop = &EventLoop{
		name:    defaultName,
		network: defaultNetWork,
		addr:    defaultAddr,
		codec:   dc,

		wg: &sync.WaitGroup{},
	}
	for _, o := range opts {
		o(eventLoop)
	}
	return eventLoop
}

// func (e *EventLoop) RegisterService(err error) {
// 	listener, err := net.Listen("tcp", e.addr)
// 	if err != nil {
// 		panic(err)
// 	}
// 	e.listener = listener
// 	err = e.serve()
// 	return
// }

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
			idx:     i,
			poll:    pool,
			packet:  make([]byte, 0xFFFF),
			fdConns: make(map[int]*conn),
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
		}
		loop := e.selectLoop()
		loop.poll.AddReadWrite(nfd)
		loop.fdConns[nfd] = conn
		return nil
	})

	return
}

func (e *EventLoop) selectLoop() (loop *loop) {
	loop = e.loops[0]
	return
}

func (e *EventLoop) loopRun(l *loop) {
	defer func() {
		//fmt.Println("-- loop stopped --", l.idx)
		// s.signalShutdown()
		e.wg.Done()
	}()
	l.poll.Wait(func(fd int, filter int16) (err error) {
		// time.Sleep(time.Minute * 1)
		if conn, exist := l.fdConns[fd]; exist {
			switch filter {
			case unix.EVFILT_READ:
				l.loopRead(fd, conn)
			case unix.EVFILT_WRITE:
				l.loopWrite(fd, conn)
			}
		}
		return
	})
}

func (l *loop) loopRead(fd int, c *conn) (err error) {
	n, err := unix.Read(c.fd, l.packet)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		delete(l.fdConns, fd)
		unix.Close(c.fd)
		return nil
	}
	var in = l.packet[:n]
	fmt.Println("in", string(in))
	var out []byte
	out = append(out[:0], in...)
	if len(out) != 0 {
		unix.Write(c.fd, out)
	}
	return nil
}

func (l *loop) loopWrite(fd int, c *conn) {

}
