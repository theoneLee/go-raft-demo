package model

/*
	modify by  6.824/src/raft/config.go
	git clone git://g.csail.mit.edu/6.824-golabs-2020 6.824
	https://pdos.csail.mit.edu/6.824/labs/lab-raft.html
*/

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"runtime"
	"sync"
	"testing"
	"time"
)

var arr = []string{"11111", "11112", "11113", "11114", "11115", "11116", "11117"}

type config struct {
	mu sync.Mutex
	t  *testing.T
	//net       *labrpc.Network

	n         int
	rafts     []*RaftNode
	applyErr  []string // from apply channel readers
	connected []bool   // whether each server is on the net
	saved     []*Persister
	endnames  [][]string    // the port file names each sends to
	logs      []map[int]int // copy of each server's committed entries
	start     time.Time     // time at which make_config() was called
	// begin()/end() statistics
	t0        time.Time // time at which test_test.go called cfg.begin()
	rpcs0     int       // rpcTotal() at start of test
	cmds0     int       // number of agreements
	maxIndex  int
	maxIndex0 int
}

func make_config(t *testing.T, n int, unreliable bool) *config {
	runtime.GOMAXPROCS(4)
	cfg := &config{}
	cfg.t = t
	//cfg.net = labrpc.MakeNetwork()
	cfg.n = n
	cfg.applyErr = make([]string, cfg.n)
	cfg.rafts = make([]*RaftNode, cfg.n)
	cfg.connected = make([]bool, cfg.n)
	cfg.saved = make([]*Persister, cfg.n)
	cfg.endnames = make([][]string, cfg.n)
	cfg.logs = make([]map[int]int, cfg.n)
	cfg.start = time.Now()

	//cfg.setunreliable(unreliable)

	//cfg.net.LongDelays(true)

	MakeRaftNodeList(cfg.n)

	for i := 0; i < cfg.n; i++ {
		go NewServer(":" + arr[i])
	}
	// create a full set of Rafts.
	for i := 0; i < cfg.n; i++ {
		cfg.logs[i] = map[int]int{}
		cfg.start1(i)
	}

	//// connect everyone
	//for i := 0; i < cfg.n; i++ {
	//	cfg.connect(i)
	//}

	return cfg
}

// shut down a Raft server but save its persistent state.
func (cfg *config) crash1(i int) {
	//cfg.disconnect(i)
	//cfg.net.DeleteServer(i) // disable client connections to the server.

	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// a fresh persister, in case old instance
	// continues to update the Persister.
	// but copy old persister's content so that we always
	// pass Make() the last persisted state.
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	}

	rf := cfg.rafts[i]
	if rf != nil {
		cfg.mu.Unlock()
		rf.Kill()
		cfg.mu.Lock()
		cfg.rafts[i] = nil
	}

	if cfg.saved[i] != nil {
		raftlog := cfg.saved[i].ReadRaftState()
		cfg.saved[i] = &Persister{}
		cfg.saved[i].SaveRaftState(raftlog)
	}
}

// start or re-start a Raft.
// if one already exists, "kill" it first.
// allocate new outgoing port file names, and a new
// state persister, to isolate previous instance of
// this server. since we cannot really kill it.
//
func (cfg *config) start1(i int) {
	cfg.crash1(i)

	//// a fresh set of outgoing ClientEnd names.
	//// so that old crashed instance's ClientEnds can't send.
	//cfg.endnames[i] = make([]string, cfg.n)
	//for j := 0; j < cfg.n; j++ {
	//	cfg.endnames[i][j] = randstring(20)
	//}

	// a fresh set of ClientEnds.
	ends := make([]*rpc.Client, cfg.n)
	for j := 0; j < cfg.n; j++ {
		//ends[j] = cfg.net.MakeEnd(cfg.endnames[i][j])
		//cfg.net.Connect(cfg.endnames[i][j], j)
		ends[j] = NewClient(":" + arr[j])
	}

	cfg.mu.Lock()

	// a fresh persister, so old instance doesn't overwrite
	// new instance's persisted state.
	// but copy old persister's content so that we always
	// pass Make() the last persisted state.
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = MakePersister()
	}

	cfg.mu.Unlock()

	// listen to messages from Raft indicating newly committed messages.
	applyCh := make(chan ApplyMsg)
	go func() {
		for m := range applyCh {
			err_msg := ""
			if m.CommandValid == false {
				// ignore other types of ApplyMsg
			} else if v, ok := (m.Command).(int); ok {
				cfg.mu.Lock()
				for j := 0; j < len(cfg.logs); j++ {
					if old, oldok := cfg.logs[j][m.CommandIndex]; oldok && old != v {
						// some server has already committed a different value for this entry!
						err_msg = fmt.Sprintf("commit index=%v server=%v %v != server=%v %v",
							m.CommandIndex, i, m.Command, j, old)
					}
				}
				_, prevok := cfg.logs[i][m.CommandIndex-1]
				cfg.logs[i][m.CommandIndex] = v
				if m.CommandIndex > cfg.maxIndex {
					cfg.maxIndex = m.CommandIndex
				}
				cfg.mu.Unlock()

				if m.CommandIndex > 1 && prevok == false {
					err_msg = fmt.Sprintf("server %v apply out of order %v", i, m.CommandIndex)
				}
			} else {
				err_msg = fmt.Sprintf("committed command %v is not an int", m.Command)
			}

			if err_msg != "" {
				log.Fatalf("apply error: %v\n", err_msg)
				cfg.applyErr[i] = err_msg
				// keep reading after error so that Raft doesn't block
				// holding locks...
			}
		}
	}()

	rf := Make(ends, i, cfg.saved[i], applyCh)

	cfg.mu.Lock()
	cfg.rafts[i] = rf
	cfg.mu.Unlock()

	//svc := labrpc.MakeService(rf)
	//srv := labrpc.MakeServer()
	//srv.AddService(svc)
	//cfg.net.AddServer(i, srv)
	//NewServer(":"+arr[i])
}

func (cfg *config) checkTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

//
//func (cfg *config) cleanup() {
//	for i := 0; i < len(cfg.rafts); i++ {
//		if cfg.rafts[i] != nil {
//			cfg.rafts[i].Kill()
//		}
//	}
//	//cfg.net.Cleanup()
//
//
//	cfg.checkTimeout()
//}

// start a Test.
// print the Test message.
// e.g. cfg.begin("Test (2B): RPC counts aren't too high")
func (cfg *config) begin(description string) {
	fmt.Printf("%s ...\n", description)
	cfg.t0 = time.Now()
	//cfg.rpcs0 = cfg.rpcTotal()
	cfg.cmds0 = 0
	cfg.maxIndex0 = cfg.maxIndex
}

// check that there's exactly one leader.
// try a few times in case re-elections are needed.
func (cfg *config) checkOneLeader() int {
	for iters := 0; iters < 10; iters++ {
		ms := 450 + (rand.Int63() % 100)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		leaders := make(map[int][]int)
		for i := 0; i < cfg.n; i++ {
			//if cfg.connected[i] {
			if term, leader := cfg.rafts[i].GetState(); leader {
				leaders[term] = append(leaders[term], i)
			}
			//}
		}

		lastTermWithLeader := -1
		for term, leaders := range leaders {
			if len(leaders) > 1 {
				cfg.t.Fatalf("term %d has %d (>1) leaders", term, len(leaders))
			}
			if term > lastTermWithLeader {
				lastTermWithLeader = term
			}
		}

		if len(leaders) != 0 {
			return leaders[lastTermWithLeader][0]
		}
	}
	cfg.t.Fatalf("expected one leader, got none")
	return -1
}

// check that everyone agrees on the term.
func (cfg *config) checkTerms() int {
	term := -1
	for i := 0; i < cfg.n; i++ {
		//if cfg.connected[i] {
		xterm, _ := cfg.rafts[i].GetState()
		if term == -1 {
			term = xterm
		} else if term != xterm {
			cfg.t.Fatalf("servers disagree on term")
		}
		//}
	}
	return term
}

// detach server i from the net.
func (cfg *config) disconnect(i int) {
	// fmt.Printf("disconnect(%d)\n", i)

	//cfg.connected[i] = false
	//
	//// outgoing ClientEnds
	//for j := 0; j < cfg.n; j++ {
	//	if cfg.endnames[i] != nil {
	//		endname := cfg.endnames[i][j]
	//		cfg.net.Enable(endname, false)
	//	}
	//}
	//
	//// incoming ClientEnds
	//for j := 0; j < cfg.n; j++ {
	//	if cfg.endnames[j] != nil {
	//		endname := cfg.endnames[j][i]
	//		cfg.net.Enable(endname, false)
	//	}
	//}
	//todo 关闭server
}

// attach server i to the net.
func (cfg *config) connect(i int) {
	// fmt.Printf("connect(%d)\n", i)

	//cfg.connected[i] = true
	//
	//// outgoing ClientEnds
	//for j := 0; j < cfg.n; j++ {
	//	if cfg.connected[j] {
	//		endname := cfg.endnames[i][j]
	//		cfg.net.Enable(endname, true)
	//	}
	//}
	//
	//// incoming ClientEnds
	//for j := 0; j < cfg.n; j++ {
	//	if cfg.connected[j] {
	//		endname := cfg.endnames[j][i]
	//		cfg.net.Enable(endname, true)
	//	}
	//}
	//todo 重新连接 server
}

// check that there's no leader
func (cfg *config) checkNoLeader() {
	for i := 0; i < cfg.n; i++ {
		if cfg.connected[i] {
			_, is_leader := cfg.rafts[i].GetState()
			if is_leader {
				cfg.t.Fatalf("expected no leader, but %v claims to be leader", i)
			}
		}
	}
}
