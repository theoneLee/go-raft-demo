package model

const (
	FOLLOWER = iota
	CANDIDATE
	LEADER
)

type State struct {
	//todo 当前节点的id ？
	ServerId int64
	Role     int

	CurrentTerm int64       //服务器最后一次知道的任期号（初始化为 0，持续递增）
	VotedFor    int64       //在当前获得选票的候选人的 Id
	Log         []LogEntity //日志条目集；每一个条目包含一个用户状态机执行的指令，和收到时的任期号

	//所有服务器上经常变的
	CommitIndex int64
	LastApplied int64

	//在领导人里经常改变的 （选举后重新初始化）
	//NextIndex []int64
	//MatchIndex []int64
	NextIndex  map[int64]int64
	MatchIndex map[int64]int64
}

type LogEntity struct {
	Index   int64
	Term    int64
	Command string
}

var s *State

func New() *State {
	//todo
}

func GetState() *State {
	return s
}
