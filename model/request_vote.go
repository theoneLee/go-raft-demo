package model

import "fmt"

type RequestVoteArgs struct {
	//
	RequestServerId int
	Term            int
	CandidateId     int
}
type RequestVoteReply struct {
	//
	Term        int
	VoteGranted bool
}

func (r *RpcMethod) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	rf := getCurRaftNode(args.RequestServerId) //r.rf
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//如果 votedFor 为空或者为 candidateId，并且候选人的日志至少和自己一样新，那么就投票给他（5.2 节，5.4 节）
	if args.Term < rf.currentTerm || (args.Term == rf.currentTerm && rf.votedFor != -1) {
		fmt.Printf("%v did not vote for node %d at term %d\n", rf, args.CandidateId, args.Term)
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return nil
	}

	rf.votedFor = args.CandidateId
	rf.currentTerm = args.Term
	reply.VoteGranted = true
	//fmt.Println("RequestVote success reply:", reply)
	// 投票后重置计时器
	rf.electionTimer.Reset(randTimeDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	rf.convertTo(FOLLOWER)
	return nil
}

func getCurRaftNode(me int) *RaftNode {
	//return curRaftNode
	return raftNodeList[me]
}
