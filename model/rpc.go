package model

import (
	"log"
	"net"
	"net/rpc"
)

type RpcMethod int

func NewClient(port string) *rpc.Client {
	serverAddress := "localhost"
	client, err := rpc.Dial("tcp", serverAddress+port) //":1234"
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return client
}

//func RpcCall(){
//	// Synchronous call
//	args := &service.Args{7, 8}
//	var reply int
//	err = client.Call("Arith.Multiply", args, &reply)
//	if err != nil {
//		log.Fatal("arith error:", err)
//	}
//	fmt.Printf("Arith: %d*%d=%d", args.A, args.B, reply)
//}

func NewServer(port string) {
	r := new(RpcMethod)
	rpc.Register(r)
	//获取tcpaddr
	tcpaddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1"+port)
	if err != nil {
		log.Fatal(err)
	}
	//监听端口
	tcplisten, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		log.Fatal(err)
	}
	//死循环处理连接请求
	for {
		conn, err3 := tcplisten.Accept()
		if err3 != nil {
			continue
		}
		//使用goroutine单独处理rpc连接请求
		go rpc.ServeConn(conn)
	}
}

func (rf *RaftNode) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	args.RequestServerId = server
	err := rf.peers[server].Call("RpcMethod.RequestVote", args, reply)
	if err != nil {
		panic(err)
		//return false
	}
	if reply.VoteGranted {
		return true
	}
	return false
}

func (rf *RaftNode) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	args.RequestServerId = server
	err := rf.peers[server].Call("RpcMethod.AppendEntries", args, reply)
	if err != nil {
		panic(err)
		//return false
	}
	if reply.Success {
		return true
	}
	return false
}
