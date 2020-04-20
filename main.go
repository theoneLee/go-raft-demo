package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"raft-demo/service"
)

func main() {
	/*todo 不考虑 日志压缩的快照 和 集群成员变化的情况。
	因此只需要实现两个rpc：

	*/

	demo()
}

func demo() {
	arith := new(service.Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}
