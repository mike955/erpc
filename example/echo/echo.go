package echo

import (
	"encoding/json"
	"fmt"

	"github.com/mike955/erpc"
)

const (
	UNKNOWN_TYPE_REQUEST = iota
	HEALTH_TYPE_REQUEST
	HELLO_TYPE_REQUEST
	UNKNOWN_TYPE_RESPONSE
	HEALTH_TYPE_RESPONSE
	HELLO_TYPE_RESPONSE
)

type EchoMessage struct {
	Session     string `json:"session"`
	MessageType int    `json:"msgType"`
	Payload     []byte `json:"payload"`
}

type EchoServer struct {
}

func NewEchoServer() (server *EchoServer) {
	server = &EchoServer{}
	return
}

func (s *EchoServer) OnConnectionSuccess(conn *erpc.TcpConnection) {
	fmt.Printf("connection from %s success\n", conn.RemoteAddr())
}

func (s *EchoServer) OnConnectionClosed(conn *erpc.TcpConnection) {
	fmt.Printf("connection from %s success\n", conn.RemoteAddr())
}

func (s *EchoServer) SetMessageReadCallback(conn *erpc.TcpConnection, req []byte) {
	var reqMessage EchoMessage
	var rpsMessage EchoMessage
	json.Unmarshal(req, &reqMessage)

	switch reqMessage.MessageType {
	case HEALTH_TYPE_REQUEST:
		rpsMessage = s.healthHandleReq(reqMessage)
	case HELLO_TYPE_REQUEST:
		rpsMessage = s.helloHandleReq(reqMessage)
	default:
		rpsMessage = s.notFoundHandleReq(reqMessage)
	}
	respData, _ := json.Marshal(rpsMessage)
	conn.Response(respData)
}

func (s *EchoServer) healthHandleReq(req EchoMessage) (resp EchoMessage) {
	resp.Session = req.Session
	resp.MessageType = HEALTH_TYPE_RESPONSE
	resp.Payload = []byte("OK")
	return
}

func (s *EchoServer) helloHandleReq(req EchoMessage) (resp EchoMessage) {
	resp.Session = req.Session
	resp.MessageType = UNKNOWN_TYPE_RESPONSE
	resp.Payload = []byte(fmt.Sprintf("hello %s", string(req.Payload)))
	return
}

func (s *EchoServer) notFoundHandleReq(req EchoMessage) (resp EchoMessage) {
	resp.Session = req.Session
	resp.MessageType = UNKNOWN_TYPE_RESPONSE
	resp.Payload = []byte("NOT FOUND")
	return
}

func (s *EchoServer) SetMessageWriteCallback(conn *erpc.TcpConnection, req []byte) {

}
