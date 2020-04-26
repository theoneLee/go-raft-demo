package model

import (
	"fmt"
	"math/rand"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

var curRaftNode *RaftNode //当前raftNode，每一个state实例各有一个，测试/运行时，需要run 3个state，注意先注册rpc server，在构建peers
var raftNodeList []*RaftNode

func MakeRaftNodeList(len int) {
	raftNodeList = make([]*RaftNode, len)
}

const (
	FOLLOWER = iota
	CANDIDATE
	LEADER

	HeartbeatInterval    = time.Duration(120) * time.Millisecond
	ElectionTimeoutLower = time.Duration(300) * time.Millisecond
	ElectionTimeoutUpper = time.Duration(400) * time.Millisecond
)

type RaftNode struct {
	mu    sync.Mutex
	peers []*rpc.Client //集群上所有节点的通讯端
	db    *Persister
	me    int //当前raftnode位于peers的index

	// requestVote 和 appendEntries
	currentTerm    int
	votedFor       int
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
	state          int
}

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

func (rf RaftNode) String() string {
	return fmt.Sprintf("[node(%d), state(%v), term(%d), votedFor(%d)]",
		rf.me, rf.state, rf.currentTerm, rf.votedFor)
}

//返回当前状态机的currentTerm和state是否是leader
func (rf *RaftNode) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	var term int
	var isleader bool
	//
	term = rf.currentTerm
	isleader = rf.state == LEADER
	return term, isleader
}

func (rf *RaftNode) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *RaftNode) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

func (rf *RaftNode) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// 2B

	return index, term, isLeader
}
func (rf *RaftNode) Kill() {
}

func (rf *RaftNode) broadcastHeartbeat() {
	args := AppendEntriesArgs{
		Term:     rf.currentTerm,
		LeaderId: rf.me,
	}

	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(server int) {
			var reply AppendEntriesReply
			if rf.sendAppendEntries(server, &args, &reply) {
				rf.mu.Lock()
				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.convertTo(FOLLOWER)
				}
				rf.mu.Unlock()
			} else {
				fmt.Printf("broadcastHeartbeat %v send request vote to %d failed\n", rf, server)
			}
		}(i)
	}
}

func (rf *RaftNode) startElection() {
	rf.currentTerm += 1
	rf.electionTimer.Reset(randTimeDuration(ElectionTimeoutLower, ElectionTimeoutUpper))

	args := RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
	}

	var voteCount int32

	for i := range rf.peers {
		if i == rf.me {
			rf.votedFor = rf.me
			atomic.AddInt32(&voteCount, 1)
			continue
		}
		go func(server int) {
			var reply RequestVoteReply
			if rf.sendRequestVote(server, &args, &reply) {
				rf.mu.Lock()

				if reply.VoteGranted && rf.state == CANDIDATE {
					atomic.AddInt32(&voteCount, 1)
					if atomic.LoadInt32(&voteCount) > int32(len(rf.peers)/2) {
						rf.convertTo(LEADER)
					}
				} else {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.convertTo(FOLLOWER)
					}
				}
				rf.mu.Unlock()
			} else {
				fmt.Printf("startElection %v send request vote to %d failed\n", rf, server)
			}
		}(i)
	}
}

func Make(peers []*rpc.Client, me int, db *Persister, applyCh chan ApplyMsg) *RaftNode {
	rf := &RaftNode{}
	//curRaftNode = rf
	raftNodeList[me] = rf
	rf.peers = peers
	rf.db = db
	rf.me = me

	rf.currentTerm = 0
	rf.votedFor = -1
	rf.heartbeatTimer = time.NewTimer(HeartbeatInterval)
	rf.electionTimer = time.NewTimer(randTimeDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	rf.state = FOLLOWER

	go func(node *RaftNode) {
		for {
			select {
			case <-rf.electionTimer.C:
				rf.mu.Lock()
				if rf.state == FOLLOWER {
					// 竞选计时器到期，follower转canidate，向各节点 requestVote
					rf.convertTo(CANDIDATE)
				} else {
					rf.startElection()
				}
				rf.mu.Unlock()

			case <-rf.heartbeatTimer.C:
				rf.mu.Lock()
				if rf.state == LEADER { //leader给各follower节点发送心跳
					rf.broadcastHeartbeat()
					rf.heartbeatTimer.Reset(HeartbeatInterval)
				}
				rf.mu.Unlock()
			}
		}
	}(rf)

	// 在崩溃前，从db获取数据进行初始化
	rf.readPersist(db.ReadRaftState())

	return rf
}

func randTimeDuration(lower, upper time.Duration) time.Duration {
	num := rand.Int63n(upper.Nanoseconds()-lower.Nanoseconds()) + lower.Nanoseconds()
	return time.Duration(num) * time.Nanosecond
}

func (rf *RaftNode) convertTo(s int) {
	if s == rf.state {
		return
	}
	fmt.Printf("Term %d: server %d convert from %v to %v\n",
		rf.currentTerm, rf.me, rf.state, s)
	rf.state = s
	switch s {
	case FOLLOWER:
		rf.heartbeatTimer.Stop()
		rf.electionTimer.Reset(randTimeDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
		rf.votedFor = -1

	case CANDIDATE:
		rf.startElection()

	case LEADER:
		rf.electionTimer.Stop()
		rf.broadcastHeartbeat()
		rf.heartbeatTimer.Reset(HeartbeatInterval)
	}
}
