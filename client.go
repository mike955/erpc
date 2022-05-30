package erpc

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	DEFAULT_HEAD_SIZE                       = 4
	MAX_CONNECT_RETRIES                     = 3
	DEFAULT_CONNECT_TIMEOUT   uint32        = 10
	DEFAULT_KEEP_ALIVE_PERIOD time.Duration = 10 * time.Second
	DEFAULT_SET_DEAD_LINE     time.Duration = 10 * time.Second
)

var (
	GlobalTcpClientPoolMu = sync.RWMutex{}
	GlobalTcpClientPool   = make(map[string]*tcpClient)
)

type ClientHandler interface {
	SetMessageReadCallback(conn *TcpConnection, msg []byte)
}

type tcpClient struct {
	key         string
	msgHeadSize int
	handler     ClientHandler
	conn        *TcpConnection

	err           chan error
	processingNum int64
}

func NewTcpClient(addr string, port int, timeout uint32) (client *tcpClient, err error) {
	var key = fmt.Sprintf("%s:%d:%d", addr, port, timeout)
	if timeout < 0 {
		timeout = DEFAULT_CONNECT_TIMEOUT
	}
	GlobalTcpClientPoolMu.RLock()
	if client, ok := GlobalTcpClientPool[key]; ok {
		GlobalTcpClientPoolMu.RUnlock()
		return client, nil
	}
	GlobalTcpClientPoolMu.RUnlock()

	client, err = newTcpClient(key, addr, port, timeout, DEFAULT_MSG_HEAD_SIZE)
	if err == nil {
		go client.clientHandler()
	}
	return
}

func (client *tcpClient) SendTCPRequest(req []byte, timeout uint32) (err error) {
	_, err = client.conn.Send(req)
	if err != nil {
		err = fmt.Errorf("send req data error(%s)", err.Error())
	}
	atomic.AddInt64(&client.processingNum, 1)
	return
}

func (client *tcpClient) clientHandler() {
	for {
		resp, err := client.conn.Receive()
		if err != nil {
			client.err <- fmt.Errorf("receive msg from %s error(%s)", client.conn.RemoteAddr(), err.Error())
			client.conn.Close()
			return
		}
		client.handler.SetMessageReadCallback(client.conn, resp)
		atomic.AddInt64(&client.processingNum, -1)
	}
}

func (client *tcpClient) ClientHandler(handler ClientHandler) {
	client.handler = handler
}

// wait 10 second
func (client *tcpClient) Close() {
	ticker := time.NewTicker(time.Second * 1)
	num := 10
	for _ = range ticker.C {
		if client.processingNum == 0 || num == 0 {
			break
		}
		num--
	}

	client.conn.Close()
}

func newTcpClient(key, addr string, port int, timeout uint32, msgHeadSize int) (client *tcpClient, err error) {
	address := fmt.Sprintf("%s:%d", addr, port)
	conn, err := net.DialTimeout("tcp", address, time.Second*time.Duration(timeout))
	if err != nil {
		err = fmt.Errorf("dial tcp error(%s)", err.Error())
		return
	}

	// TODO(mike.cai): maybe can create multi connection
	tcpConn, err := newConnection(uuid.NewString(), conn, DEFAULT_MSG_HEAD_SIZE, DEFAULT_KEEP_ALIVE_TIMEOUT)
	err = tcpConn.conn.(*net.TCPConn).SetDeadline(time.Time{})
	if err != nil {
		err = fmt.Errorf("set conn deadline error(%s)", err.Error())
		return
	}
	client = &tcpClient{
		key:         key,
		msgHeadSize: msgHeadSize,
		conn:        tcpConn,
	}
	GlobalTcpClientPoolMu.Lock()
	defer GlobalTcpClientPoolMu.Unlock()
	GlobalTcpClientPool[key] = client
	return
}
