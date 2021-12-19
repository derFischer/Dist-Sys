package simulation

import (
	"Common"
	"log"
	"reflect"
	"sort"
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
	From int
	mterm        int
	msource int
	mlastLogIndex int
	mlastLogTerm  int
}

type RequestVoteReply struct {
	From int
	mterm        int
	mvoteGranted bool
}

type AppendEntriesArgs struct {
	From int
	mterm     int
	msource int
	mprevLogIndex int
	mprevLogTerm int
	mentries      []Command
	mcommitIndex int
}

type AppendEntriesReply struct {
	From int
	mterm     int
	msuccess   bool
	//mmatchIndex int
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
	Votes       []bool
	Log []Command
	CommitIndex int
	mmatchIndex []int
	peers int
}

func NewSL(id int, total int) *Simulation {
	votes := make([]bool, total)
	for i :=0 ; i < total; i++ {
		votes[i] = false
	}
	matchIndex := make([]int, total)
	for i := 0; i  <  total; i++ {
		matchIndex[i] = -1
	}
	log := make([]Command, 1)
	log[0] = Command{CommandIndex: 0}
	return &Simulation{
		me: id,
		State: Follower,
		CurrentTerm: 0,
		VotedFor: -1,
		Votes: votes,
		Log: log,
		CommitIndex: 0,
		mmatchIndex: matchIndex,
		peers: total,
	}
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
	//////fmt.Printf("print commands: %v, %v\n", arg1, arg2)
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
		if (!(c1.CommandIndex == c2.CommandIndex && c1.CommandTerm == c2.CommandTerm && reflect.DeepEqual(c1.Command, c2.Command))) {
			//////fmt.Printf("print command: %v, %v, %v\n", c1, c2, reflect.DeepEqual(c1.Command, c2.Command))
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
		result = append(result, s.getCommand(prevIndex + i))
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
		if command.Command.Key == key && command.Command.Op == "PUT" {
			found = true
			value = command.Command.Value
		}
	}
	return found, value
}

func (s *Simulation) appendEntries(entries []Command)  {
	for _, command := range entries {
		current_index := s.getCurrentIndex(command.CommandIndex)
		if current_index == TRUNCATED {
			continue
		}
		if current_index == NONEXIST {
			s.Log = append(s.Log, command)
			continue
		}
		s.Log[current_index] = command
	}
}

//Interpreter: interpret the message to simulation

func CommandInterpreter(command Common.Command) Command {
	op := command.Command.(Common.Op)
	return Command{CommandIndex: command.CommandIndex, CommandTerm: command.CommandTerm, Command: Op{Op: op.Op, Key: op.Key, Value: op.Value, ClientId: op.ClientId, ReqId: op.ReqId}}
}

func CommandsInterpreter(commands []Common.Command) []Command {
	var result []Command
	for _, command := range commands {
		result = append(result, CommandInterpreter(command))
	}
	return result
}

func GetRequestInterpreter(args *Common.GetArgs) Op {
	return Op{Op: "GET", Key: args.Key, ClientId: args.ClientId, ReqId: args.ReqId}
}

func PutRequestInterpreter(args *Common.PutAppendArgs) Op {
	return Op{Op: "PUT", Key: args.Key, Value: args.Value, ClientId: args.ClientId, ReqId: args.ReqId}
}

func GetReplyInterpreter(args *Common.GetReply) GetReply {
	return GetReply{Value: args.Value, Success: args.Err == Common.OK}
}

func PutReplyInterpreter(args *Common.PutAppendReply) PutAppendReply {
	return PutAppendReply{Success: args.Err == Common.OK}
}

func RequestVoteRequestInterPreter(args *Common.RequestVoteArgs) RequestVoteArgs {
	return RequestVoteArgs{From: args.From, mterm: args.Term, msource: args.CandidateId, mlastLogIndex: args.LastLogIndex, mlastLogTerm: args.LastLogTerm}
}

func AppendEntryRequestInterPreter(args *Common.AppendEntriesArgs) AppendEntriesArgs {
	return AppendEntriesArgs{From: args.From, mterm: args.Term, msource: args.LeaderId, mprevLogIndex: args.PrevLogIndex, mprevLogTerm: args.PrevLogTerm, mentries: CommandsInterpreter(args.Entries), mcommitIndex: args.LeaderCommit}
}

func RequestVoteReplyInterpreter(reply *Common.RequestVoteReply) RequestVoteReply {
	return RequestVoteReply{From: reply.From, mterm: reply.Term, mvoteGranted: reply.VoteGranted}
}

func AppendEntryReplyInterpreter(reply *Common.AppendEntriesReply) AppendEntriesReply {
	return AppendEntriesReply{From: reply.From, mterm: reply.Term, msuccess: reply.Success}
}


//call From the upper layer
//checker function: check whether the messages are valid before actually send them out
func (s *Simulation) CheckRequestMessage(svcMeth string, args interface{})  {
	valid := true
	switch svcMeth {
	case "Raft.RequestVote":
		valid = s.RequestVoteRequestChecker(RequestVoteRequestInterPreter(args.(*Common.RequestVoteArgs)))
	case "Raft.AppendEntries":
		valid = s.AppendEntryRequestChecker(AppendEntryRequestInterPreter(args.(*Common.AppendEntriesArgs)))
	case "Timeout":
		valid = s.TimeoutChecker()
	}
	if !valid {
		log.Fatalf("check request message error! server: %d, method: %s, args: %+v\n local state: %+v\n", s.me, svcMeth, args, s)
	}
}

func (s *Simulation) CheckReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	valid := true
	switch svcMeth {
	//case "Raft.RequestVote":
	//	valid = s.RequestVoteReplyChecker(RequestVoteRequestInterPreter(args.(*Common.RequestVoteArgs)), RequestVoteReplyInterpreter(reply.(*Common.RequestVoteReply)))
	//case "Raft.AppendEntries":
	//	valid = s.AppendEntryReplyChecker(AppendEntryRequestInterPreter(args.(*Common.AppendEntriesArgs)), AppendEntryReplyInterpreter(reply.(*Common.AppendEntriesReply)))
	case "Raft.RequestVote":
		valid = s.RequestVoteReplyChecker(RequestVoteReplyInterpreter(args.(*Common.RequestVoteReply)), reply.(RequestVoteReply))
	case "Raft.AppendEntries":
		valid = s.AppendEntryReplyChecker(AppendEntryReplyInterpreter(args.(*Common.AppendEntriesReply)), reply.(AppendEntriesReply))
	case "KVServer.Get":
		valid = s.GetReplyChecker(GetRequestInterpreter(args.(*Common.GetArgs)), GetReplyInterpreter(reply.(*Common.GetReply)))
	case "KVServer.PutAppend":
		valid = s.PutReplyChecker(PutRequestInterpreter(args.(*Common.PutAppendArgs)), PutReplyInterpreter(reply.(*Common.PutAppendReply)))
	}
	if !valid {
		log.Fatalf("check reply message error! server: %d, method: %s, args: %+v, reply: %+v\n local state: %+v\n", s.me, svcMeth, args, reply, s)
	}
}

//handler function: update the simulation state once receives the requests
func (s *Simulation) HandleRequestMessage(svcMeth string, args interface{}) interface{} {
	switch svcMeth {
	case "Raft.RequestVote":
		//fmt.Printf("server %d receives request %s, args: %+v\n", s.me, svcMeth, args)
		return s.RequestVoteRequestHandler(RequestVoteRequestInterPreter(args.(*Common.RequestVoteArgs)))
	case "Raft.AppendEntries":
		//fmt.Printf("server %d receives request %s, args: %+v\n, self: %+v\n", s.me, svcMeth, args, s)
		return s.AppendEntryRequestHandler(AppendEntryRequestInterPreter(args.(*Common.AppendEntriesArgs)))
	case "KVServer.Get":
		//fmt.Printf("server %d receives request %s, args: %+v\n", s.me, svcMeth, args)
		s.ClientRequestHandler(GetRequestInterpreter(args.(*Common.GetArgs)))
	case "KVServer.PutAppend":
		//fmt.Printf("server %d receives request %s, args: %+v\n", s.me, svcMeth, args)
		s.ClientRequestHandler(PutRequestInterpreter(args.(*Common.PutAppendArgs)))
	case "Timeout":
		//fmt.Printf("server %d receives request %s, args: %+v\n", s.me, svcMeth)
		s.TimeoutHandler()
	}
	return nil
}

func (s *Simulation) HandleReplyMessage(svcMeth string, args interface{}, reply interface{})  {
	switch svcMeth {
	case "Raft.RequestVote":
		//fmt.Printf("server %d receives request %s, reply: %+v\n", s.me, svcMeth, reply)
		s.RequestVoteReplyHandler(RequestVoteRequestInterPreter(args.(*Common.RequestVoteArgs)), RequestVoteReplyInterpreter(reply.(*Common.RequestVoteReply)))
	case "Raft.AppendEntries":
		//fmt.Printf("server %d receives request %s, reply: %+v\n", s.me, svcMeth, reply)
		s.AppendEntryReplyHandler(AppendEntryRequestInterPreter(args.(*Common.AppendEntriesArgs)), AppendEntryReplyInterpreter(reply.(*Common.AppendEntriesReply)))
	}
}

//checker: check whether the message is correct

func (s *Simulation) RequestVoteRequestChecker(args RequestVoteArgs) bool {
	return s.State == Candidate && args.mterm == s.CurrentTerm && args.msource == s.me && args.mlastLogIndex == s.getLastCommandIndex() && args.mlastLogTerm == s.getLastCommandTerm()
}

func (s *Simulation) AppendEntryRequestChecker(args AppendEntriesArgs) bool {
	state_check := (s.State == Leader) && (args.mterm == s.CurrentTerm) && (args.msource == s.me) && (args.mcommitIndex <= s.CommitIndex)
	params_checker := Equal(args.mprevLogTerm, s.getCommandTerm(args.mprevLogIndex)) && SliceEqual(args.mentries, s.getLogSlice(args.mprevLogIndex, len(args.mentries)))
	////fmt.Printf("request args: %+v, local state: %+v\n", args, s)
	////fmt.Printf("state_check: %+v, params_check: %+v\n", state_check, SliceEqual(args.mentries, s.getLogSlice(args.mprevLogIndex, len(args.mentries))))
	return state_check && params_checker
}

func (s *Simulation) TimeoutChecker() bool {
	return s.State == Follower || s.State == Candidate
}

//func (s *Simulation) RequestVoteReplyChecker(args RequestVoteArgs, reply RequestVoteReply) bool {
//	logOK := (args.mlastLogTerm > s.getLastCommandTerm()) || (args.mlastLogTerm == s.getLastCommandTerm() && args.mlastLogIndex >= s.getLastCommandIndex())
//	grant := (args.mterm == s.CurrentTerm) && logOK && (s.VotedFor == -1 || s.VotedFor == args.msource)
//	return reply.mterm == s.CurrentTerm && reply.mvoteGranted == grant
//}
//
//func (s *Simulation) AppendEntryReplyChecker(args AppendEntriesArgs, reply AppendEntriesReply) bool {
//	logOK := (args.mprevLogIndex == 0) || (args.mprevLogIndex > 0 && args.mprevLogIndex <= s.getLastCommandIndex() && Equal(args.mprevLogTerm, s.getCommandTerm(args.mprevLogIndex)))
//	success := (args.mterm == s.CurrentTerm) && (s.State == Follower) && logOK
//	return reply.mterm == s.CurrentTerm && reply.msuccess == success
//}

func (s *Simulation) RequestVoteReplyChecker(args RequestVoteReply, reply RequestVoteReply) bool {
	return args == reply
}

func (s *Simulation) AppendEntryReplyChecker(args AppendEntriesReply, reply AppendEntriesReply) bool {
	return args == reply
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
	if s.State == Leader {
		command := Command{Command: args, CommandIndex: s.getLastCommandIndex() + 1, CommandTerm: s.CurrentTerm}
		s.Log = append(s.Log, command)
		s.mmatchIndex[s.me] = s.getLastCommandIndex()
	}
	return
}

func (s *Simulation) RequestVoteRequestHandler(args RequestVoteArgs) RequestVoteReply {
	r := RequestVoteReply{From: s.me}
	if s.DropStaleResponse(args.mterm) {
		r.mterm = s.CurrentTerm
		r.mvoteGranted = false
		return r
	}
	s.UpdateTerm(args.mterm)
	logOK := (args.mlastLogTerm > s.getLastCommandTerm()) || (args.mlastLogTerm == s.getLastCommandTerm() && args.mlastLogIndex >= s.getLastCommandIndex())
	grant := (args.mterm == s.CurrentTerm) && logOK && (s.VotedFor == -1 || s.VotedFor == args.From)
	if grant {
		s.VotedFor = args.From
	}
	r.mterm = s.CurrentTerm
	r.mvoteGranted = grant
	return r
}

func (s *Simulation) AppendEntryRequestHandler(args AppendEntriesArgs) AppendEntriesReply {
	r := AppendEntriesReply{From: s.me}
	if s.DropStaleResponse(args.mterm) {
		r.mterm = s.CurrentTerm
		r.msuccess = false
		return r
	}
	s.UpdateTerm(args.mterm)
	logOK := (args.mprevLogIndex == 0) || (args.mprevLogIndex > 0 && args.mprevLogIndex <= s.getLastCommandIndex() && Equal(args.mprevLogTerm, s.getCommandTerm(args.mprevLogIndex)))
	if args.mterm == s.CurrentTerm && s.State == Follower && !logOK {
		r.mterm = s.CurrentTerm
		r.msuccess = false
		return r
	}
	if args.mterm == s.CurrentTerm && s.State == Candidate {
		s.State = Follower
	}
	if args.mterm == s.CurrentTerm && s.State == Follower && logOK {
		s.appendEntries(args.mentries)
		s.CommitIndex = max(s.CommitIndex, args.mcommitIndex)
	}
	r.mterm = s.CurrentTerm
	r.msuccess = (args.mterm == s.CurrentTerm) && (s.State == Follower) && logOK
	return r
}

func (s *Simulation) RequestVoteReplyHandler(args RequestVoteArgs, reply RequestVoteReply) {
	if s.DropStaleResponse(reply.mterm) {
		return
	}
	s.UpdateTerm(reply.mterm)
	if reply.mvoteGranted {
		s.Votes[reply.From] = true
		s.testLeader()
	}
	return
}

func (s *Simulation) AppendEntryReplyHandler(args AppendEntriesArgs, reply AppendEntriesReply) {
	//fmt.Printf("appendentry args, %+v, reply: %+v\n", args, reply)
	if s.DropStaleResponse(reply.mterm) {
		return
	}
	s.UpdateTerm(reply.mterm)
	if reply.msuccess {
		s.mmatchIndex[reply.From] = max(args.mprevLogIndex + len(args.mentries), s.mmatchIndex[reply.From])
		s.updateCommitIndex()
	}
	return
}

func (s *Simulation) TimeoutHandler()  {
	s.Candidate()
}



//other functions
func (s *Simulation) DropStaleResponse(mterm int) bool {
	if mterm < s.CurrentTerm {
		return true
	}
	return false
}

func (s *Simulation) UpdateTerm(mterm int) {
	if mterm > s.CurrentTerm {
		s.CurrentTerm = mterm
		s.Follower()
	}
}

func (s *Simulation) testLeader() {
	if s.State != Candidate {
		return
	}
	votes := 0
	var i int
	for i = 0; i < s.peers; i++ {
		if s.Votes[i] == true {
			votes += 1
		}
	}
	if votes > s.peers/2 {
		s.Leader()
	}
}

func (s *Simulation) Candidate()  {
	s.State = Candidate
	s.CurrentTerm = s.CurrentTerm + 1
	s.VotedFor = s.me
	for i := 0; i < s.peers; i++ {
		s.Votes[i] = false
	}
	s.Votes[s.me] = true
}

func (s *Simulation) Follower()  {
	s.State = Follower
	s.VotedFor = -1
}

func (s *Simulation) Leader()  {
	var i int
	s.State = Leader
	for i = 0; i < s.peers; i++ {
		s.mmatchIndex[i] = 0
	}
	s.mmatchIndex[s.me] = s.getLastCommandIndex()
	//fmt.Printf("server %d becomes leader, state: %+v\n", s.me, s)
}

func (s *Simulation) updateCommitIndex()  {
	majority := findMajority(s.mmatchIndex)
	if majority > s.CommitIndex {
		command := s.getCommand(majority)
		if command.CommandTerm == s.CurrentTerm {
			s.CommitIndex = max(command.CommandIndex, s.CommitIndex)
		}
	}
}

func (s *Simulation) Reconfig(peers int)  {
	s.peers = peers
}

//helper function
func max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func findMajority(array []int) int {
	if !sort.IntsAreSorted(array) {
		sort.Ints(array)
	}
	length := len(array)
	return array[(length - 1) / 2]
}