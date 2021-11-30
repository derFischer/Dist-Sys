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

const (
	TRUNCATED = -1
	NONEXIST = -2
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
func Equal(arg1 int, arg2 int) bool {
	if arg1 == TRUNCATED || arg2 == TRUNCATED {
		return true
	}
	if arg1 == NONEXIST || arg2 == NONEXIST {
		return false
	} else {
		return arg1 == arg2
	}
}

func SliceEqual(arg1 []Command, arg2 []Command) bool {
	if len(arg1) != len(arg2) {
		return false
	}
	length := len(arg1)
	for index := 0; index < length; index++ {
		c1 := arg1[index]
		c2 := arg2[index]
		if c1.CommandIndex == NONEXIST || c2.CommandIndex == NONEXIST {
			return false
		}
		if c1.CommandIndex == TRUNCATED || c2.CommandIndex == TRUNCATED {
			continue
		}
		if (!(c1.CommandIndex == c2.CommandIndex && c1.CommandTerm == c2.CommandTerm && c1.Command == c2.Command)) {
			return false
		}
	}
	return true
}

func (s *Simulation) getLastCommandIndex() int {
	return s.Log[len(s.Log)-1].CommandIndex
}

func (s *Simulation) getLastCommandTerm() int {
	return s.Log[len(s.Log)-1].CommandTerm
}

func (s *Simulation) getCurrentIndex(commandIndex int) int {
	first_command_index := s.Log[0].CommandIndex
	current_index := commandIndex - first_command_index
	if current_index < 0 {
		return TRUNCATED
	} else if current_index > len(s.Log) - 1 {
		return NONEXIST
	}
	return current_index
}

func (s *Simulation) getCommand(commandIndex int) Command {
	localIndex := s.getCurrentIndex(commandIndex)
	if localIndex == TRUNCATED {
		return Command{CommandIndex: TRUNCATED, CommandTerm: TRUNCATED}
	} else if localIndex == NONEXIST {
		return Command{CommandIndex: NONEXIST, CommandTerm: NONEXIST}
	}
	return s.Log[localIndex]
}

func (s *Simulation) getCommandTerm(commandIndex int) int {
	return s.getCommand(commandIndex).CommandTerm
}

func (s *Simulation) getLogSlice(prevIndex int, length int) []Command {
	var result []Command
	for i := 1; i <= length; i++ {
		result = append(result, s.getCommand(prevIndex + 1))
	}
	return result
}

func (s *Simulation) entryInLog(args Op) (bool, Command) {
	for _, command := range s.Log {
		if command.Command == args {
			return true, command
		}
	}
	return false, Command{}
}

func (s *Simulation) Get(key string, commandIndex int) (bool, string) {
	found := false
	value := ""
	for _, command := range s.Log {
		if command.CommandIndex > commandIndex {
			break
		}
		if command.Command.Key == key && command.Command.Op == "Put" {
			found = true
			value = command.Command.Value
		}
	}
	return found, value
}

//call from the upper layer
//checker function: check whether the messages are valid before actually send them out
func (s *Simulation) CheckRequestMessage(svcMeth string, args interface{})  {
	valid := true
	switch svcMeth {
	case "Raft.RequestVote":
		valid = s.RequestVoteRequestChecker(RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
	case "Raft.AppendEntries":
		valid = s.AppendEntryRequestChecker(AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
	}
	if !valid {
		log.Fatalf("check request message error! server: %d, method: %s, args: %+v\n", s.me, svcMeth, args)
	}
}

func (s *Simulation) CheckReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	valid := true
	switch svcMeth {
	case "Raft.RequestVote":
		valid = s.RequestVoteReplyChecker(RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)), RequestVoteReplyInterpreter(reply.(*raft.RequestVoteReply)))
	case "Raft.AppendEntries":
		valid = s.AppendEntryReplyChecker(AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)), AppendEntryReplyInterpreter(reply.(*raft.AppendEntriesReply)))
	case "KVServer.Get":
		s.GetReplyChecker(GetRequestInterpreter(args.(*raftkv.GetArgs)), GetReplyInterpreter(reply.(*raftkv.GetReply)))
	case "KVServer.PutAppend":
		s.PutReplyChecker(PutRequestInterpreter(args.(*raftkv.PutAppendArgs)), PutReplyInterpreter(reply.(*raftkv.PutAppendReply)))
	}
	if !valid {
		log.Fatalf("check reply message error! server: %d, method: %s, args: %+v, reply: %+v\n", s.me, svcMeth, args, reply)
	}
}

//handler function: update the simulation state once receives the requests
func (s *Simulation) HandleRequestMessage(svcMeth string, args interface{})  {
	switch svcMeth {
	case "Raft.RequestVote":
		s.RequestVoteRequestHandler(RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
	case "Raft.AppendEntries":
		s.AppendEntryRequestHandler(AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
	case "KVServer.Get":
		s.ClientRequestHandler(GetRequestInterpreter(args.(*raftkv.GetArgs)))
	case "KVServer.PutAppend":
		s.ClientRequestHandler(PutRequestInterpreter(args.(*raftkv.PutAppendArgs)))
	}
}

func (s *Simulation) HandleReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	switch svcMeth {
	case "Raft.RequestVote":
		s.RequestVoteReplyHandler(RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)), RequestVoteReplyInterpreter(reply.(*raft.RequestVoteReply)))
	case "Raft.AppendEntries":
		s.AppendEntryReplyHandler(AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)), AppendEntryReplyInterpreter(reply.(*raft.AppendEntriesReply)))
	}
}

//Interpreter: interpret the message to simulation

func CommandInterpreter(command raft.Command) Command {
	op := command.Command.(raftkv.Op)
	return Command{CommandIndex: command.CommandIndex, CommandTerm: command.CommandTerm, Command: Op{Op: op.Op, Key: op.Key, Value: op.Value, ClientId: op.ClientId, ReqId: op.ReqId}}
}

func CommandsInterpreter(commands []raft.Command) []Command {
	var result []Command
	for _, command := range commands {
		result = append(result, CommandInterpreter(command))
	}
	return result
}

func GetRequestInterpreter(args *raftkv.GetArgs) Op {
	return Op{Op: "Get", Key: args.Key, ClientId: args.ClientId, ReqId: args.ReqId}
}

func PutRequestInterpreter(args *raftkv.PutAppendArgs) Op {
	return Op{Op: "Put", Key: args.Key, Value: args.Value, ClientId: args.ClientId, ReqId: args.ReqId}
}

func GetReplyInterpreter(args *raftkv.GetReply) GetReply {
	return GetReply{Value: args.Value, Success: args.Err == raftkv.OK}
}

func PutReplyInterpreter(args *raftkv.PutAppendReply) PutAppendReply {
	return PutAppendReply{Success: args.Err == raftkv.OK}
}

func RequestVoteRequestInterPreter(args *raft.RequestVoteArgs) RequestVoteArgs {
	return RequestVoteArgs{mterm: args.Term, msource: args.CandidateId, mlastLogIndex: args.LastLogIndex, mlastLogTerm: args.LastLogTerm}
}

func AppendEntryRequestInterPreter(args *raft.AppendEntriesArgs) AppendEntriesArgs {
	return AppendEntriesArgs{mterm: args.Term, msource: args.LeaderId, mprevLogIndex: args.PrevLogIndex, mprevLogTerm: args.PrevLogTerm, mentries: CommandsInterpreter(args.Entries), mcommitIndex: args.LeaderCommit}
}

func RequestVoteReplyInterpreter(reply *raft.RequestVoteReply) RequestVoteReply {
	return RequestVoteReply{mterm: reply.Term, mvoteGranted: reply.VoteGranted}
}

func AppendEntryReplyInterpreter(reply *raft.AppendEntriesReply) AppendEntriesReply {
	return AppendEntriesReply{mterm: reply.Term, msuccess: reply.Success, mmatchIndex: reply.NextIndex - 1}
}


//checker: check whether the message is correct

func (s *Simulation) RequestVoteRequestChecker(args RequestVoteArgs) bool {

	return s.State == Candidate && args.mterm == s.CurrentTerm && args.msource == s.me && args.mlastLogIndex == s.getLastCommandIndex() && args.mlastLogTerm == s.getLastCommandTerm()
}

func (s *Simulation) AppendEntryRequestChecker(args AppendEntriesArgs) bool {
	state_check := (s.State == Leader) && (args.mterm == s.CurrentTerm) && (args.msource == s.me) && (args.mcommitIndex == s.CommitIndex)
	params_checker := Equal(args.mprevLogTerm, s.getCommandTerm(args.mprevLogIndex)) && SliceEqual(args.mentries, s.getLogSlice(args.mprevLogIndex, len(args.mentries)))
	return state_check && params_checker
}

func (s *Simulation) RequestVoteReplyChecker(args RequestVoteArgs, reply RequestVoteReply) bool {
	logOK := (args.mlastLogTerm > s.getLastCommandTerm()) || (args.mlastLogTerm == s.getLastCommandTerm() && args.mlastLogIndex >= s.getLastCommandIndex())
	grant := (args.mterm == s.CurrentTerm) && logOK && (s.VotedFor == -1 || s.VotedFor == args.msource)
	return reply.mterm == s.CurrentTerm && reply.mvoteGranted == grant
}

func (s *Simulation) AppendEntryReplyChecker(args AppendEntriesArgs, reply AppendEntriesReply) bool {
	logOK := (args.mprevLogIndex == 0) || (args.mprevLogIndex > 0 && args.mprevLogIndex < s.getLastCommandIndex() && Equal(args.mprevLogTerm, s.getCommandTerm(args.mprevLogIndex)))
	success := (args.mterm == s.CurrentTerm) && (s.State == Follower) && logOK
	return reply.mterm == s.CurrentTerm && reply.msuccess == success && (!success || reply.mmatchIndex == args.mprevLogIndex + len(args.mentries))
}

func (s *Simulation) GetReplyChecker(args Op, reply GetReply) bool {
	found, command := s.entryInLog(args)
	key_in_log, value := s.Get(args.Key, command.CommandIndex)
	commit := s.CommitIndex >= command.CommandIndex
	return !reply.Success || !found || !key_in_log || value == reply.Value && commit
}

func (s *Simulation) PutReplyChecker(args Op, reply PutAppendReply) bool {
	found, command := s.entryInLog(args)
	commit := s.CommitIndex >= command.CommandIndex
	return !reply.Success || !found ||  commit
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

