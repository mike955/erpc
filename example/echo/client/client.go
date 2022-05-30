package main

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mike955/erpc"
	"github.com/mike955/erpc/example/echo"
)

func main() {
	client, err := erpc.NewTcpClient("127.0.0.1", 8090, 10)
	if err != nil {
		fmt.Println(err)
		return
	}
	client.ClientHandler(newEchoClient())
	request := echo.EchoMessage{
		Session:     uuid.NewString(),
		MessageType: echo.HEALTH_TYPE_REQUEST,
		Payload:     []byte(""),
	}
	req, _ := json.Marshal(request)
	client.SendTCPRequest(req, 0)
	client.Close()
}

type EchoClient struct {
}

func newEchoClient() (client *EchoClient) {
	client = &EchoClient{}
	return
}

func (c EchoClient) SetMessageReadCallback(conn *erpc.TcpConnection, req []byte) {
	var reqMessage echo.EchoMessage
	// var rpsMessage echo.EchoMessage
	json.Unmarshal(req, &reqMessage)

	switch reqMessage.MessageType {
	case echo.HEALTH_TYPE_RESPONSE:
		_ = c.healthHandleResp(reqMessage)
	case echo.HELLO_TYPE_RESPONSE:
		_ = c.helloHandleResp(reqMessage)
	default:
		_ = c.notFoundHandleResp(reqMessage)
	}
	// respData, _ := json.Marshal(rpsMessage)
	// conn.Response(respData)
}

func (c EchoClient) healthHandleResp(req echo.EchoMessage) (resp echo.EchoMessage) {
	fmt.Printf("health %+v\n", req)
	fmt.Println(string(req.Payload))
	return
}

func (c EchoClient) helloHandleResp(req echo.EchoMessage) (resp echo.EchoMessage) {
	fmt.Printf("hello %+v\n", req)
	fmt.Println(string(req.Payload))
	return
}

func (c EchoClient) notFoundHandleResp(req echo.EchoMessage) (resp echo.EchoMessage) {
	fmt.Printf("not found %+v\n", req)
	fmt.Println(string(req.Payload))
	return
}
