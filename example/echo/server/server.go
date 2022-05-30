package main

import (
	"fmt"

	"github.com/mike955/erpc"
	"github.com/mike955/erpc/example/echo"
)

var ()

func main() {
	server := erpc.NewTcpServer(
		erpc.Address("127.0.0.1:8090"),
		erpc.Handler(echo.NewEchoServer()),
	)
	err := server.Serve()
	fmt.Println("err: ", err.Error())
}
