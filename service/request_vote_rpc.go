package service

import "raft-demo/model"

type RequestVoteReq struct {
	Term        int64 //候选人的任期号
	CandidateId int64 //请求选票的候选人的 Id

	LastLogIndex int64 //候选人的最后日志条目的索引值
	LastLogTerm  int64 //候选人最后日志条目的任期号

}

type RequestVoteResp struct {
	Term        int64 //当前任期号，以便于候选人去更新自己的任期号
	VoteGranted bool  //候选人赢得了此张选票时为真
}

type RequestVote int

func (t *RequestVote) RequestVoteRpc(req *RequestVoteReq, resp *RequestVoteResp) error {
	//if args.B == 0 {
	//	return errors.New("divide by zero")
	//}
	//quo.Quo = args.A / args.B
	//quo.Rem = args.A % args.B
	state := model.GetState()
	if req.Term < state.CurrentTerm {
		resp = &RequestVoteResp{
			Term:        state.CurrentTerm,
			VoteGranted: false,
		}
		return nil
	}
	if state.VotedFor == 0 || state.VotedFor == req.CandidateId {
		//如果 votedFor 为空或者为 candidateId，并且候选人的日志至少和自己一样新，那么就投票给他（5.2 节，5.4 节）
		resp = &RequestVoteResp{
			Term:        req.Term,
			VoteGranted: true,
		}
		state.VotedFor = req.CandidateId
	}

	return nil
}
