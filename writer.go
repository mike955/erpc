package erpc

import (
	"encoding/binary"
	"io"
)

type writer struct {
	msgHeadSize int
	buf         []byte
	w           io.Writer
}

func newWriter(w io.Writer, headSize int) (wr *writer) {
	wr = &writer{
		w:           w,
		msgHeadSize: headSize,
		buf:         make([]byte, headSize),
	}
	return
}

func (w *writer) writePacket(packet []byte) (n int, err error) {
	// 将数据包大小写入头部
	n, err = w.writeHead(len(packet))
	if err != nil {
		return 0, err
	}
	// 发送数据包
	return w.writeBody(packet)
}

func (w *writer) writeHead(plen int) (n int, err error) {
	return w.writeUint32BE(uint32(plen))
}

func (w *writer) writeUint32BE(v uint32) (n int, err error) {
	binary.BigEndian.PutUint32(w.buf[:w.msgHeadSize], v)
	return w.w.Write(w.buf[:w.msgHeadSize])
}

func (w *writer) writeBody(body []byte) (n int, err error) {
	return w.w.Write(body)
}

func (w *writer) writeClose(body []byte) (n int, err error) {
	return w.w.Write(body)
}
