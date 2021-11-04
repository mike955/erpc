package internal

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type Poll struct {
	fd int
	// changes []unix.Kevent_t
}

func CreatePoll() (poll *Poll, err error) {
	poll = &Poll{}
	if poll.fd, err = unix.Kqueue(); err != nil {
		poll = nil
		err = os.NewSyscallError("create kqueue error:", err)
		return
	}
	if _, err = unix.Kevent(poll.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil); err != nil {
		poll = nil
		err = os.NewSyscallError("kqueue add kevent error:", err)
	}
	return
}

func (p *Poll) Close() (err error) {
	if err = unix.Close(p.fd); err != nil {
		err = os.NewSyscallError("close kqueue fd error:", err)
	}
	return
}

func (p *Poll) AddRead(fd int) (err error) {
	read := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ}
	if _, err = unix.Kevent(p.fd, []unix.Kevent_t{read}, nil, nil); err != nil {
		err = os.NewSyscallError("kqueue add kevent error: ", err)
	}
	return
}

func (p *Poll) AddWrite(fd int) (err error) {
	write := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE}
	if _, err = unix.Kevent(p.fd, []unix.Kevent_t{write}, nil, nil); err != nil {
		err = os.NewSyscallError("kqueue add kevent error: ", err)
	}
	return
}

func (p *Poll) AddReadWrite(fd int) (err error) {
	read := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ}
	write := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE}
	if _, err = unix.Kevent(p.fd, []unix.Kevent_t{read, write}, nil, nil); err != nil {
		err = os.NewSyscallError("kqueue add kevent error: ", err)
	}
	return
}

func (p *Poll) ModRead(fd int) (err error) {
	read := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE}
	if _, err = unix.Kevent(p.fd, []unix.Kevent_t{read}, nil, nil); err != nil {
		err = os.NewSyscallError("kqueue delete kevent error: ", err)
	}
	return
}

func (p *Poll) ModReadWrite(fd int) (err error) {
	read := unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE}
	if _, err = unix.Kevent(p.fd, []unix.Kevent_t{read}, nil, nil); err != nil {
		err = os.NewSyscallError("kqueue add kevent error: ", err)
	}
	return
}

func (p *Poll) Wait(callback func(fd int, filter int16) error) error {
	events := make([]unix.Kevent_t, 128)
	var tsp *unix.Timespec
	for {
		n, err := unix.Kevent(p.fd, nil, events, tsp)
		if err != nil && err != unix.EINTR {
			if err == unix.EBADF {
				return nil
			}
		}
		for i := 0; i < n; i++ {
			event := events[i]
			if fd := int(event.Ident); fd != 0 {
				efilter := event.Filter
				// eflags := event.Flags
				if (event.Flags&unix.EV_EOF != 0) || (event.Flags&unix.EV_ERROR != 0) {
					efilter = -0xd
				}
				callback(fd, efilter)
				// switch {
				// 	case efilter = unix.con
				// case efilter == unix.EVFILT_READ && eflags&unix.EV_ENABLE != 0:
				// 	callback(fd, efilter)
				// case efilter == unix.EVFILT_WRITE && eflags&unix.EV_ENABLE != 0:
				// 	callback(fd, efilter)
				// }
				// switch err = callback(fd, efilter); err {
				// case nil:
				// case errors.New(""):
				// default:
				// 	fmt.Println("event loop error")
				// }
			}
		}
	}
}
