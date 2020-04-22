package model

import (
	"sync"
	"time"
)

const (
	FOLLOWER = iota
	CANDIDATE
	LEADER

	HBINTERVAL = 50 * time.Millisecond //heart beat interval
)

type RaftNode struct {
	mu        sync.Mutex
	peers     []*Peer //集群上所有节点的通讯端
	persister *Persister
	me        int //当前raftnode位于peers的index

	state         int // 当前节点身份
	voteCount     int
	chanCommit    chan bool
	chanHeartbeat chan bool
	chanGrantVote chan bool
	chanLeader    chan bool
	chanApply     chan ApplyMsg

	//在所有servers都要保持的
	currentTerm int
	voteFor     int
	log         []LogEntry

	//在所有servers都易变
	commitIndex int
	lastApplied int

	//在leader节点易变的
	nextIndex  []int
	matchIndex []int
}

// 已
type ApplyMsg struct {
	Index   int
	Command interface{}
	//UseSnapshot bool   //todo 快照
	//Snapshot    []byte //
}

type LogEntry struct {
	LogIndex   int
	LogTerm    int
	LogCommand interface{}
}
