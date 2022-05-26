package main

import (
	"github.com/mike955/erpc"
)

func main() {
	event := erpc.NewServer("test", erpc.LoopNum(1))
	event.Serve()
}
