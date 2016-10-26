// +build ignore

package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/unicok/snet"
)

func main() {
	go StartServer(false, false, "10010")
	go StartServer(false, true, "10011")
	go StartServer(true, false, "10012")
	go StartServer(true, true, "10013")

	if pid := syscall.Getpid(); pid != 1 {
		ioutil.WriteFile("test_server.pid", []byte(strconv.Itoa(pid)), 0644)
		defer os.Remove("test_server.pid")
	}

	sigTERM := make(chan os.Signal, 1)
	signal.Notify(sigTERM, syscall.SIGTERM)
	<-sigTERM

	log.Println("test server killed")
}

func StartServer(unstable, enableCrypt bool, port string) {
	config := snet.Config{
		EnableCrypt:        enableCrypt,
		HandshakeTimeout:   time.Second * 5,
		RewriterBufferSize: 1024,
		ReconnWaitTimeout:  time.Minute * 5,
	}

	listener, err := snet.Listen(config, func() (net.Listener, error) {
		l, err := net.Listen("tcp", "127.0.0.1:"+port)
		if err != nil {
			return nil, err
		}
		return &unstableListener{l}, nil
	})
	if err != nil {
		log.Fatalf("listen failed: %s", err.Error())
		return
	}
	log.Println("server start:", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("accept failed: %s", err.Error())
			return
		}
		log.Println("new client")
		go func() {
			buf := make([]byte, 1024)
			uconn := &unstableConn{nil, unstable}
			for {
				if unstable {
					conn.(*snet.Conn).WrapBaseForTest(func(base net.Conn) net.Conn {
						if base != uconn {
							uconn.Conn = base
							return uconn
						}
						return base
					})
				}
				n, err := conn.Read(buf)
				if err != nil {
					break
				}
				_, err = conn.Write(buf[:n])
				if err != nil {
					break
				}
			}
			conn.Close()
			log.Println("connnection closed")
		}()
	}
}

type unstableListener struct {
	net.Listener
}

func (l *unstableListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &unstableConn{Conn: conn}, nil
}

type unstableConn struct {
	net.Conn
	enable bool
}

func (c *unstableConn) Write(b []byte) (int, error) {
	if c.enable {
		if rand.Intn(10000) < 500 {
			c.Conn.Close()
		}
	}
	return c.Conn.Write(b)
}

func (c *unstableConn) Read(b []byte) (int, error) {
	if c.enable {
		if rand.Intn(10000) < 100 {
			c.Conn.Close()
		}
	}
	return c.Conn.Read(b)
}