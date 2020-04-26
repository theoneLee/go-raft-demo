package model

type AppendEntriesArgs struct {
	RequestServerId int
	Term            int
	LeaderId        int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (r *RpcMethod) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	rf := getCurRaftNode(args.RequestServerId) //r.rf
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm { //如果 term < currentTerm 就返回 false （5.1 节）
		reply.Success = false
		reply.Term = rf.currentTerm
		return nil
	}
	reply.Success = true
	rf.currentTerm = args.Term
	rf.electionTimer.Reset(randTimeDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	rf.convertTo(FOLLOWER)
	return nil
}
