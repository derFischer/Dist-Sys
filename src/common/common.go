package Common

const (
	OK       = "OK"
	ErrNoKey = "ErrNoKey"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId int64
	ReqId    int
}

type PutAppendReply struct {
	WrongLeader bool
	Err         Err
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	ClientId int64
	ReqId    int
}

type GetReply struct {
	WrongLeader bool
	Err         Err
	Value       string
}

type Recent struct {
	ReqId  int
	Result string
	Err    Err
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Op        string
	Key       string
	Value     string
	ClientId  int64
	ReqId     int
	ApplyChan chan Recent
}


//raft structures
type Command struct {
	CommandIndex int
	Command      interface{}
	CommandTerm  int
}

type RequestVoteArgs struct {
	From 		int
	Term        int
	CandidateId int

	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	From		int
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	From	 int
	Term     int
	LeaderId int

	//part B
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Command
	LeaderCommit int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type AppendEntriesReply struct {
	From	  int
	Term      int
	Success   bool
	NextIndex int
}