package simulation


import (
	raftkv "kvraft"
	"log"
	"raft"
	"sync"
)

// import "bytes"
// import "labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//

const (
	Leader    = 0
	Follower  = 1
	Candidate = 2
)

type RequestVoteArgs struct {
	mterm        int
	msource int
	mlastLogIndex int
	mlastLogTerm  int
}

type RequestVoteReply struct {
	mterm        int
	mvoteGranted bool
}

type AppendEntriesArgs struct {
	mterm     int
	msource int
	mprevLogIndex int
	mprevLogTerm int
	mentries      []Command
	mcommitIndex int
}

type AppendEntriesReply struct {
	mterm     int
	msuccess   bool
	mmatchIndex int
}

type Command struct {
	CommandIndex int
	Command      Op
	CommandTerm  int
}

type Op struct {
	Op        string
	Key       string
	Value     string
	ClientId  int64
	ReqId     int
}

type GetReply struct {
	Success bool
	Value string
}

type PutAppendReply struct {
	Success bool
}

//
// A Go object implementing a single Raft peer.
//
type Simulation struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	me        int
	State       int
	CurrentTerm int
	VotedFor    int
	Votes       int
	VotesResponded []int
	Log []Command
	CommitIndex int
	mmatchIndex []int
}
//helper function
func (s *Simulation) getLastCommandIndex() int {
	return s.Log[len(s.Log)-1].CommandIndex
}

func (s *Simulation) getLastCommandTerm() int {
	return s.Log[len(s.Log)-1].CommandTerm
}

func (s *Simulation) getCommandTerm(commandIndex int) int {
	return 0
}

func (s *Simulation) getLogSlice(prevIndex int, length int) []Command {
	return []Command{}
}

//call from the upper layer
//checker function: check whether the messages are valid before actually send them out
func (s *Simulation) CheckRequestMessage(svcMeth string, args interface{})  {
	valid := true
	switch svcMeth {
	case "Raft.RequestVote":
		valid = s.RequestVoteRequestChecker(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
	case "Raft.AppendEntries":
		valid = s.AppendEntryRequestChecker(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
	}
	if !valid {
		log.Fatalf("check request message error! server: %d, method: %s, args: %+v\n", s.me, svcMeth, args)
	}
}

func (s *Simulation) CheckReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	valid := true
	switch svcMeth {
	case "Raft.RequestVote":
		valid = s.RequestVoteReplyChecker(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)), s.RequestVoteReplyInterpreter(reply.(*raft.RequestVoteReply)))
	case "Raft.AppendEntries":
		valid = s.AppendEntryReplyChecker(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)), s.AppendEntryReplyInterpreter(reply.(*raft.AppendEntriesReply)))
	case "KVServer.Get":
		s.GetReplyChecker(s.GetRequestInterpreter(args.(*raftkv.GetArgs)), s.GetReplyInterpreter(reply.(*raftkv.GetReply)))
	case "KVServer.PutAppend":
		s.PutReplyChecker(s.PutRequestInterpreter(args.(*raftkv.PutAppendArgs)), s.PutReplyInterpreter(reply.(*raftkv.PutAppendReply)))
	}
	if !valid {
		log.Fatalf("check reply message error! server: %d, method: %s, args: %+v, reply: %+v\n", s.me, svcMeth, args, reply)
	}
}

//handler function: update the simulation state once receives the requests
func (s *Simulation) HandleRequestMessage(svcMeth string, args interface{})  {
	switch svcMeth {
	case "Raft.RequestVote":
		s.RequestVoteRequestHandler(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
	case "Raft.AppendEntries":
		s.AppendEntryRequestHandler(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
	case "KVServer.Get":
		s.ClientRequestHandler(s.GetRequestInterpreter(args.(*raftkv.GetArgs)))
	case "KVServer.PutAppend":
		s.ClientRequestHandler(s.PutRequestInterpreter(args.(*raftkv.PutAppendArgs)))
	}
}

func (s *Simulation) HandleReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	switch svcMeth {
	case "Raft.RequestVote":
		s.RequestVoteReplyHandler(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)), s.RequestVoteReplyInterpreter(reply.(*raft.RequestVoteReply)))
	case "Raft.AppendEntries":
		s.AppendEntryReplyHandler(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)), s.AppendEntryReplyInterpreter(reply.(*raft.AppendEntriesReply)))
	}
}

//Interpreter: interpret the message to simulation
func (s *Simulation) CommandInterpreter(command []raft.Command) []Command {
	return []Command{}
}

func (s *Simulation) GetRequestInterpreter(args *raftkv.GetArgs) Op {
	return Op{Op: "Get", Key: args.Key, ClientId: args.ClientId, ReqId: args.ReqId}
}

func (s *Simulation) PutRequestInterpreter(args *raftkv.PutAppendArgs) Op {
	return Op{Op: "Put", Key: args.Key, Value: args.Value, ClientId: args.ClientId, ReqId: args.ReqId}
}

func (s *Simulation) GetReplyInterpreter(args *raftkv.GetReply) GetReply {
	return GetReply{Value: args.Value, Success: args.Err == raftkv.OK}
}

func (s *Simulation) PutReplyInterpreter(args *raftkv.PutAppendReply) PutAppendReply {
	return PutAppendReply{Success: args.Err == raftkv.OK}
}

func (s *Simulation) RequestVoteRequestInterPreter(args *raft.RequestVoteArgs) RequestVoteArgs {
	return RequestVoteArgs{mterm: args.Term, msource: args.CandidateId, mlastLogIndex: args.LastLogIndex, mlastLogTerm: args.LastLogTerm}
}

func (s *Simulation) AppendEntryRequestInterPreter(args *raft.AppendEntriesArgs) AppendEntriesArgs {
	return AppendEntriesArgs{mterm: args.Term, msource: args.LeaderId, mprevLogIndex: args.PrevLogIndex, mprevLogTerm: args.PrevLogTerm, mentries: s.CommandInterpreter(args.Entries), mcommitIndex: args.LeaderCommit}
}

func (s *Simulation) RequestVoteReplyInterpreter(reply *raft.RequestVoteReply) RequestVoteReply {
	return RequestVoteReply{mterm: reply.Term, mvoteGranted: reply.VoteGranted}
}

func (s *Simulation) AppendEntryReplyInterpreter(reply *raft.AppendEntriesReply) AppendEntriesReply {
	return AppendEntriesReply{mterm: reply.Term, msuccess: reply.Success, mmatchIndex: reply.NextIndex - 1}
}


//checker: check whether the message is correct
//func (s *Simulation) ReplyToClient(args string) bool {
//	return true
//}

func (s *Simulation) RequestVoteRequestChecker(args RequestVoteArgs) bool {

	return s.State == Candidate && args.mterm == s.CurrentTerm && args.msource == s.me && args.mlastLogIndex == s.getLastCommandIndex() && args.mlastLogTerm == s.getLastCommandTerm()
}

func (s *Simulation) AppendEntryRequestChecker(args AppendEntriesArgs) bool {
	state_check := (s.State == Leader) && (args.mterm == s.CurrentTerm) && (args.msource == s.me) && (args.mcommitIndex == s.CommitIndex)
	params_checker := (args.mprevLogTerm == s.getCommandTerm(args.mprevLogIndex)) && (args.mentries == s.getLogSlice(args.mprevLogIndex, len(args.mentries)))
	return state_check && params_checker
}

func (s *Simulation) RequestVoteReplyChecker(args RequestVoteArgs, reply RequestVoteReply) bool {
	logOK := (args.mlastLogTerm > s.getLastCommandTerm()) || (args.mlastLogTerm == s.getLastCommandTerm() && args.mlastLogIndex >= s.getLastCommandIndex())
	grant := (args.mterm == s.CurrentTerm) && logOK && (s.VotedFor == -1 || s.VotedFor == args.msource)
	return reply.mterm == s.CurrentTerm && reply.mvoteGranted == grant
}

func (s *Simulation) AppendEntryReplyChecker(args AppendEntriesArgs, reply AppendEntriesReply) bool {
	logOK := (args.mprevLogIndex == 0) || (args.mprevLogIndex > 0 && args.mprevLogIndex < s.getLastCommandIndex() && args.mprevLogTerm == s.getCommandTerm(args.mprevLogIndex))
	success := (args.mterm == s.CurrentTerm) && (s.State == Follower) && logOK
	return reply.mterm == s.CurrentTerm && reply.msuccess == success && (!success || reply.mmatchIndex == args.mprevLogIndex + len(args.mentries))
}

func (s *Simulation) GetReplyChecker(args Op, reply GetReply) bool {
	return true
}

func (s *Simulation) PutReplyChecker(args Op, reply PutAppendReply) bool {
	return true
}



//Handler:
func (s *Simulation) ClientRequestHandler(args Op) {
	return
}

func (s *Simulation) RequestVoteRequestHandler(args RequestVoteArgs) {
	return
}

func (s *Simulation) AppendEntryRequestHandler(args AppendEntriesArgs) {
	return
}

func (s *Simulation) RequestVoteReplyHandler(args RequestVoteArgs, reply RequestVoteReply) {
	return
}

func (s *Simulation) AppendEntryReplyHandler(args AppendEntriesArgs, reply AppendEntriesReply) {
	return
}

