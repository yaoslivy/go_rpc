package main

import (
	"encoding/json"
	"fmt"
	"go_rpc"
	"go_rpc/codec"
	"log"
	"net"
	"time"
)

//实现了一个消息的编解码器GobCodec，并且客户端与服务端实现了简单的协议交换;
// 即允许客户端使用不同的编码方式，同时实现了服务端的雏形，建立连接，读取，处理并回复客户端的请求

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	go_rpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)
	//确保服务端监听成功后，客户端再发起请求
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()

	time.Sleep(time.Second)
	//客户端发送options进行协议交换，
	_ = json.NewEncoder(conn).Encode(go_rpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	//发送请求和接收响应
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		//发送消息头和消息体
		_ = cc.Write(h, fmt.Sprintf("go rpc req %d", h.Seq))
		// 解析服务端的响应reply
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)

	}
}
