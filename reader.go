package erpc

import (
	"encoding/binary"
	"fmt"
	"io"
)

type reader struct {
	msgHeadSize int
	buf         []byte
	r           io.Reader
}

func newReader(r io.Reader, headSize int) (rd *reader) {
	rd = &reader{
		r:           r,
		msgHeadSize: headSize,
		buf:         make([]byte, headSize),
	}
	return
}

func (r *reader) readPacket() (packet []byte, err error) {
	n, err := r.readHeader()
	if err != nil {
		return
	}
	packet, err = r.readBody(n)
	if err != nil {
		return
	}
	return
}

func (r *reader) readHeader() (hlen int, err error) {
	n, err := r.readUint32BE()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *reader) readUint32BE() (n uint32, err error) {
	_, err = io.ReadFull(r.r, r.buf[:r.msgHeadSize])
	if err != nil {
		return 0, err
	}
	n = binary.BigEndian.Uint32(r.buf[:r.msgHeadSize])
	return
}

func (r *reader) readBody(blen int) (buf []byte, err error) {
	buf = make([]byte, blen)
	n, err := io.ReadFull(r.r, buf)
	if err != nil {
		return nil, err
	}
	if n != blen {
		return nil, fmt.Errorf("readBody expect:%d, but have %d byte", blen, n)
	}
	return buf, nil
}
