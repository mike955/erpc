package erpc

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type TcpConnection struct {
	id          string
	conn        net.Conn
	msgHeadSize int

	reader    *reader
	writer    *writer
	sendMutex sync.Mutex
	recvMutex sync.Mutex
}

func newConnection(id string, conn net.Conn, headSize int, keepaliveTimeout int) (connection *TcpConnection, err error) {
	connection = &TcpConnection{
		id:          id,
		conn:        conn,
		msgHeadSize: headSize,
		reader:      newReader(conn, headSize),
		writer:      newWriter(conn, headSize),
	}
	if keepaliveTimeout > 0 {
		if err = connection.conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
			err = fmt.Errorf("set keepalive error(%s)", err.Error())
			return
		}
		if err = connection.conn.(*net.TCPConn).SetKeepAlivePeriod(time.Second * time.Duration(keepaliveTimeout)); err != nil {
			err = fmt.Errorf("set keepalive period error(%s)", err.Error())
			return
		}
	}
	return
}

func (c *TcpConnection) RemoteAddr() (addr string) {
	addr = c.conn.RemoteAddr().String()
	return
}

func (c *TcpConnection) LocalAddr() (addr string) {
	addr = c.conn.LocalAddr().String()
	return
}

func (c *TcpConnection) Receive() (msg []byte, err error) {
	c.recvMutex.Lock()
	defer c.recvMutex.Unlock()
	if msg, err = c.reader.readPacket(); err != nil {
		c.conn.Close()
	}
	return
}

func (c *TcpConnection) Send(msg []byte) (n int, err error) {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	if n, err = c.writer.writePacket(msg); err != nil {
		c.Close()
	}
	return
}

func (c *TcpConnection) Response(resp []byte) (n int, err error) {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	if n, err = c.writer.writePacket(resp); err != nil {
		c.conn.Close()
	}
	return
}

func (c *TcpConnection) Close() {
	c.conn.Close()
}
