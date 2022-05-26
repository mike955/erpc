package main

import (
	"fmt"
	"io/ioutil"
	"net"
)

func main() {
	// ln, err := net.Listen("tcp", "127.0.0.1:8080")
	// if err != nil {
	// 	fmt.Printf("list tcp error(%s)", err)
	// 	return
	// }
	// for {
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		fmt.Printf("accept connect error(%s)", err)
	// 		return
	// 	}
	// 	fmt.Println("--------1")
	// 	go func(conn net.Conn) {
	// 		fmt.Println("--------2")
	// 		// var data = make([]byte, 100)
	// 		data, err := ioutil.ReadAll(conn)
	// 		if err != nil {
	// 			fmt.Printf("receive msg error(%s)", err)
	// 			return
	// 		}
	// 		fmt.Println(data)
	// 		_, err = conn.Write(data)
	// 		if err != nil {
	// 			fmt.Printf("send msg error(%s)", err)
	// 			return
	// 		}
	// 	}(conn)
	// }

	// go func() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("list tcp error(%s)", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("accept connect error(%s)", err)
		}
		go func(conn net.Conn) {
			// var rechead = make([]byte, 4)
			data, err := ioutil.ReadAll(conn)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("---------------")
			fmt.Println(data)
			// _, err = io.ReadFull(conn, rechead[:4])
			// if err != nil {
			// 	fmt.Printf("receive msg error(%s)", err)
			// 	return
			// }
			// fmt.Println("head:", string(rechead))
			// n := binary.BigEndian.Uint32(rechead[:4])
			// var msg = make([]byte, n)
			// _, err = io.ReadFull(conn, msg[:n])
			// fmt.Println("msg:", string(msg))

			// var respHead = make([]byte, 4)
			// binary.BigEndian.PutUint32(respHead[:4], uint32(len(msg)))
			// _, err = conn.Write(respHead[:4])
			// if err != nil {
			// 	// return fmt.Errorf("write head error: %s", err.Error())
			// }
			// _, err = conn.Write(msg)
			// if err != nil {
			// 	fmt.Printf("send msg error(%s)", err)
			// 	return
			// }
		}(conn)
	}
	// }()
	// time.Sleep(time.Second)
	// conn, err := net.Dial("tcp", ":8080")
	// if err != nil {
	// 	fmt.Printf("dial server error(%s)", err)
	// 	return
	// }
	// defer conn.Close()
	// var msg = []byte("hello world")
	// var sendhead = make([]byte, 4)
	// // for i := 0; i < fmt.N; i++ {
	// binary.BigEndian.PutUint32(sendhead[:4], uint32(len(msg)))
	// _, err = conn.Write(sendhead[:4])
	// if err != nil {
	// 	fmt.Printf("send msg error(%s)", err)
	// 	return
	// }
	// _, err = conn.Write(msg)
	// // fmt.Println("===================")
	// var rechead = make([]byte, 4)
	// _, err = io.ReadFull(conn, rechead[:4])
	// if err != nil {
	// 	fmt.Printf("receive msg error(%s)", err)
	// 	return
	// }
	// n := binary.BigEndian.Uint32(rechead[:4])
	// var data = make([]byte, n)
	// _, err = io.ReadFull(conn, data[:n])
	// fmt.Println("data:", string(data))
	// if strings.Compare(string(data), string(msg)) == 0 {
	// 	fmt.Printf("server error")
	// }
	// }
}
