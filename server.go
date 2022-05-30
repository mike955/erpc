package erpc

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
)

var (
	DEFAULT_MSG_HEAD_SIZE      = 4
	DEFAULT_KEEP_ALIVE_TIMEOUT = 10
)

type ServerHandler interface {
	OnConnectionSuccess(conn *TcpConnection)
	OnConnectionClosed(conn *TcpConnection)
	// SetThreadInitCallback()
	SetMessageReadCallback(conn *TcpConnection, msg []byte)
	SetMessageWriteCallback(conn *TcpConnection, msg []byte)
}

type ServerOption func(o *tcpServer)

type tcpServer struct {
	name        string
	address     string
	numThreads  int
	msgHeadSize int
	keepalive   bool

	listener net.Listener
	handler  ServerHandler

	connMap   map[string]*TcpConnection
	connMapMu sync.Mutex

	err chan error
}

func NewTcpServer(opts ...ServerOption) (srv *tcpServer) {
	srv = &tcpServer{
		name:        "erpc",
		address:     "127.0.0.1:9090",
		msgHeadSize: DEFAULT_MSG_HEAD_SIZE,
		keepalive:   true,
		handler:     defaultServerHandler(),
		connMap:     make(map[string]*TcpConnection),
		connMapMu:   sync.Mutex{},
	}
	for _, o := range opts {
		o(srv)
	}
	return
}

func (s *tcpServer) Serve() (err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", s.address); err != nil {
		err = fmt.Errorf("create listener(%s) error(%s)", s.address, err.Error())
		return
	}
	s.listener = listener
	for {
		c, acceeptErr := s.listener.Accept()
		if acceeptErr != nil { // TODO(mike.cai): log error
			s.err <- fmt.Errorf("listener accept error(%s)", acceeptErr.Error())
			continue
		}
		connID := uuid.NewString()
		connection, err := newConnection(connID, c, s.msgHeadSize, DEFAULT_KEEP_ALIVE_TIMEOUT)
		if err != nil {
			s.err <- fmt.Errorf("receive connection(%s) error(%s)", c.RemoteAddr(), err.Error())
			continue
		}
		s.connMapMu.Lock()
		s.connMap[connID] = connection
		s.connMapMu.Unlock()

		s.handler.OnConnectionSuccess(connection)
		// 每个连接启动一个goroutine处理
		go s.serveConnection(connection)
	}
}

func (s *tcpServer) serveConnection(c *TcpConnection) {
	for {
		req, err := c.Receive()
		if err != nil {
			s.err <- fmt.Errorf("receive msg from %s error(%s)", c.conn.RemoteAddr(), err.Error())
			s.RemoveConn(c.id)
			return
		}
		go s.handler.SetMessageReadCallback(c, req) // 每个每个请求使用一个 goroutine 处理
	}
}

func (s *tcpServer) RemoveConn(connID string) {
	s.connMapMu.Lock()
	defer s.connMapMu.Unlock()
	if conn, ok := s.connMap[connID]; ok {
		delete(s.connMap, connID)
		s.handler.OnConnectionClosed(conn)
	}
}

func (s *tcpServer) Set(numThreads int) {
	s.numThreads = numThreads
}

func (s *tcpServer) Name() (name string) {
	name = s.name
	return
}

func (s *tcpServer) Address() (address string) {
	address = s.address
	return
}

func (s *tcpServer) NumThreads() (numThreads int) {
	numThreads = s.numThreads
	return
}

func Name(name string) ServerOption {
	return func(s *tcpServer) {
		s.name = name
	}
}

func Address(addr string) ServerOption {
	return func(s *tcpServer) {
		s.address = addr
	}
}

func ThreadNums(numThreads int) ServerOption {
	return func(s *tcpServer) {
		if numThreads <= 0 {
			numThreads = 1
		}
		s.numThreads = numThreads
	}
}

func Handler(handler ServerHandler) ServerOption {
	return func(s *tcpServer) {
		s.handler = handler
	}
}

type serverHandler struct {
	logger *log.Logger
}

func defaultServerHandler() (handler *serverHandler) {
	handler = &serverHandler{
		logger: log.Default(),
	}
	return
}

func (h *serverHandler) OnConnectionSuccess(conn *TcpConnection) {
	h.logger.Printf("connection from %s success", conn.conn.RemoteAddr().String())
}

func (h *serverHandler) OnConnectionClosed(conn *TcpConnection) {
	h.logger.Printf("connection from %s closed", conn.conn.RemoteAddr().String())
}

func (h *serverHandler) SetThreadInitCallback() {

}

func (h *serverHandler) SetMessageReadCallback(conn *TcpConnection, msg []byte) {
	req := string(msg)
	h.logger.Printf("receive mesaage from %s, req:%s", conn.conn.RemoteAddr().String(), req)
}

func (h *serverHandler) SetMessageWriteCallback(conn *TcpConnection, msg []byte) {

}
