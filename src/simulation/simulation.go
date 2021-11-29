package simulation

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
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
	Command      interface{}
	CommandTerm  int
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
	Log []Command
	CommitIndex int
}
//helper function
func (s *Simulation) getLastCommandIndex() int {
	return s.Log[len(s.Log)-1].CommandIndex
}

func (s *Simulation) getLastCommandTerm() int {
	return s.Log[len(s.Log)-1].CommandTerm
}


//call from the upper layer
func (s *Simulation) CheckMessage(svcMeth string, args interface{}, reply bool)  {
	switch svcMeth {
	case "Raft.RequestVote":
		if !reply {
			s.RequestVoteRequestChecker(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
		} else {
			s.RequestVoteReplyChecker(s.RequestVoteReplyInterpreter(args.(*raft.RequestVoteReply)))
		}
	case "Raft.AppendEntries":
		if !reply {
			s.AppendEntryRequestChecker(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
		} else {
			s.AppendEntryReplyChecker(s.AppendEntryReplyInterpreter(args.(*raft.AppendEntriesReply)))
		}
	}
}

func (s *Simulation) Handler(svcMeth string, args interface, reply bool)  {
	switch svcMeth {
	case "Raft.RequestVote":
		if !reply {
			s.RequestVoteRequestHandler(s.RequestVoteRequestInterPreter(args.(*raft.RequestVoteArgs)))
		} else {
			s.RequestVoteReplyHandler(s.RequestVoteReplyInterpreter(args.(*raft.RequestVoteReply)))
		}
	case "Raft.AppendEntries":
		if !reply {
			s.AppendEntryRequestHandler(s.AppendEntryRequestInterPreter(args.(*raft.AppendEntriesArgs)))
		} else {
			s.AppendEntryReplyHandler(s.AppendEntryReplyInterpreter(args.(*raft.AppendEntriesReply)))
		}
	}
}

//Interpreter: interpret the requests to simulation
func (s *Simulation) CommandInterpreter(command []raft.Command) []Command {
	return []Command{}
}

func (s *Simulation) RequestVoteRequestInterPreter(args *raft.RequestVoteArgs) RequestVoteArgs {
	return RequestVoteArgs{mterm: args.Term, msource: args.CandidateId, mlastLogIndex: args.LastLogIndex, mlastLogTerm: args.LastLogTerm}
}

func (s *Simulation) AppendEntryRequestInterPreter(args *raft.AppendEntriesArgs) AppendEntriesArgs {
	return AppendEntriesArgs{mterm: args.Term, msource: args.LeaderId, mprevLogIndex: args.PrevLogIndex, mprevLogTerm: args.PrevLogTerm, mentries: s.CommandInterpreter(args.Entries), mcommitIndex: args.LeaderCommit}
}

func (s *Simulation) RequestVoteReplyInterpreter(args *raft.RequestVoteReply) RequestVoteReply {
	return RequestVoteReply{mterm: args.Term, mvoteGranted: args.VoteGranted}
}

func (s *Simulation) AppendEntryReplyInterpreter(args *raft.AppendEntriesReply) AppendEntriesReply {
	return AppendEntriesReply{mterm: args.Term, msuccess: args.Success, mmatchIndex: args.NextIndex - 1}
}


//checker: check whether the message is correct
//func (s *Simulation) ReplyToClient(args string) bool {
//	return true
//}

func (s *Simulation) RequestVoteRequestChecker(args RequestVoteArgs) bool {

	return args.mterm == s.CurrentTerm && args.msource == s.me && args.mlastLogIndex == s.getLastCommandIndex() && args.mlastLogTerm == s.getLastCommandTerm()
}

func (s *Simulation) AppendEntryRequestChecker(args AppendEntriesArgs) bool {
	return
}

func (s *Simulation) RequestVoteReplyChecker(args RequestVoteReply) bool {
	return true
}

func (s *Simulation) AppendEntryReplyChecker(args AppendEntriesReply) bool {
	return true
}



//Handler:
func (s *Simulation) ClientRequestHandler(command Command) {
	return
}

func (s *Simulation) RequestVoteRequestHandler(args RequestVoteArgs) {
	return
}

func (s *Simulation) AppendEntryRequestHandler(args AppendEntriesArgs) {
	return
}

func (s *Simulation) RequestVoteReplyHandler(args RequestVoteReply) {
	return
}

func (s *Simulation) AppendEntryReplyHandler(args AppendEntriesReply) {
	return
}

