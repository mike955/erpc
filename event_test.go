package erpc

import (
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	event := NewServer("test", LoopNum(1))
	go func() {
		event.Serve()
	}()
	time.Sleep(time.Second)

	t.Run("tcp", func(t *testing.T) {
		conn, err := net.Dial("tcp", "127.0.0.1:9000")
		if err != nil {
			t.Errorf("dial server error(%s)", err)
			return
		}
		defer conn.Close()
		var msg = "hello world"
		_, err = conn.Write([]byte(msg))
		if err != nil {
			t.Errorf("send msg error(%s)", err)
			return
		}
		var data = make([]byte, 100)
		_, err = conn.Read(data)
		if err != nil {
			t.Errorf("receive msg error(%s)", err)
			return
		}
		fmt.Println(string(data))
		if strings.Compare(string(data), msg) == 0 {
			t.Errorf("server error")
		}
		return
	})
}

func BenchmarkBasic(b *testing.B) {
	event := NewServer("test", LoopNum(1))
	go func() {
		event.Serve()
	}()
	time.Sleep(time.Second)
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		b.Errorf("dial server error(%s)", err)
		return
	}
	defer conn.Close()
	var msg = "hello world"
	for i := 0; i < b.N; i++ {
		_, err = conn.Write([]byte(msg))
		if err != nil {
			b.Errorf("send msg error(%s)", err)
			return
		}
		var data = make([]byte, 100)
		_, err = conn.Read(data)
		if err != nil {
			b.Errorf("receive msg error(%s)", err)
			return
		}
		if strings.Compare(string(data), msg) == 0 {
			b.Errorf("server error")
		}
	}
}

func BenchmarkNetpollBasic(b *testing.B) {
	go func() {
		ln, err := net.Listen("tcp", ":8080")
		if err != nil {
			b.Errorf("list tcp error(%s)", err)
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				b.Errorf("accept connect error(%s)", err)
			}
			go func(conn net.Conn) {
				var data = make([]byte, 100)
				_, err := io.ReadFull(conn, data)
				if err != nil {
					b.Errorf("receive msg error(%s)", err)
					return
				}
				_, err = conn.Write(data)
				if err != nil {
					b.Errorf("send msg error(%s)", err)
					return
				}
			}(conn)
		}
	}()
	time.Sleep(time.Second)
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		b.Errorf("dial server error(%s)", err)
		return
	}
	defer conn.Close()
	var msg = "hello world"
	for i := 0; i < b.N; i++ {
		_, err = conn.Write([]byte(msg))
		if err != nil {
			b.Errorf("send msg error(%s)", err)
			return
		}
		var data = make([]byte, 100)
		_, err = conn.Read(data)
		if err != nil {
			b.Errorf("receive msg error(%s)", err)
			return
		}
		if strings.Compare(string(data), msg) == 0 {
			b.Errorf("server error")
		}
	}
}
