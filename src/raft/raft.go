package raft

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
	"Common"
	"sync"
)
import "labrpc"
import "time"
import (
	"bytes"
	"labgob"
	"math/rand"
	"simulation"
	"sort"
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
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

const (
	Leader    = 0
	Follower  = 1
	Candidate = 2

	HEARTBEAT    = 500
	ELECTION_MIN = 3000
	ELECTION_MAX = 10000
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	State       int
	CurrentTerm int
	VotedFor    int
	Votes       int

	Timer *time.Timer

	AppendEntryCh chan struct{} //signal, inform the server that the log has been appended
	RequestVoteCh chan struct{} //signal, inform the server that vote successfully

	//part B
	Log []Common.Command

	CommitIndex int
	LastApplied int

	NextIndex  []int
	MatchIndex []int

	ApplyCh chan ApplyMsg

	Process bool
	sl *simulation.Simulation

	//control election
	block_elec bool
}

func inSlice(s []int, e int) (bool, int) {
	length := len(s)
	for i := 0; i < length; i++ {
		if s[i] == e {
			return true, i
		}
	}
	return false, -1
}

func (rf *Raft) BlockElec()  {
	rf.block_elec = true
}

func (rf *Raft) UnblockElec()  {
	rf.block_elec = false
}

func (rf *Raft) Peers() []*labrpc.ClientEnd {
	return rf.peers
}

func (rf *Raft) SetPeers(p []*labrpc.ClientEnd) {
	rf.mu.Lock()
	total := len(p)
	rf.peers = p
	rf.MatchIndex = make([]int, total)
	rf.NextIndex = make([]int, total)
	for i := 0; i < total; i++ {
		rf.MatchIndex[i] = -1
		rf.NextIndex[i] = rf.getLastCommandIndex() + 1
	}
	rf.Votes = 0
	rf.mu.Unlock()
}

func (rf *Raft) calculateRealIndex(raw int) int {
	offset := rf.Log[0].CommandIndex
	return raw + offset
}

func (rf *Raft) calculateLocalIndex(real int) int {
	offset := rf.Log[0].CommandIndex
	return real - offset
}

func (rf *Raft) generate_rand() time.Duration {
	////////fmt.Println("enter: generate_rand");
	return time.Duration((rand.Int()%(ELECTION_MAX-ELECTION_MIN) + ELECTION_MIN)) * time.Millisecond
}

func (rf *Raft) getTerm() int {
	rf.mu.Lock()
	term := rf.CurrentTerm
	rf.mu.Unlock()
	return term
}
func (rf *Raft) getState() int {
	////////fmt.Println("enter: getState");
	rf.mu.Lock()
	state := rf.State
	rf.mu.Unlock()
	return state
}

func (rf *Raft) setState(state int) {
	rf.mu.Lock()
	rf.State = state
	rf.mu.Unlock()
}

func (rf *Raft) getLastCommandIndex() int {
	return rf.Log[len(rf.Log)-1].CommandIndex
}

func (rf *Raft) getLastCommandTerm() int {
	return rf.Log[len(rf.Log)-1].CommandTerm
}

func (rf *Raft) GetRaftStateSize() int {
	return rf.persister.RaftStateSize()
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	////////fmt.Println("enter: GetState");
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term := rf.CurrentTerm
	state := rf.State
	if state == Leader {
		return term, true
	} else {
		return term, false
	}
}

//return the index which the majority of servers have appended to local log
func (rf *Raft) findMajority() int {
	////////fmt.Println("enter: findMajority");
	matchIndex := append([]int{}, rf.MatchIndex...)
	if !sort.IntsAreSorted(matchIndex) {
		sort.Ints(matchIndex)
	}
	length := len(matchIndex)
	////fmt.Printf("leader %d, matchindex: %+v\n", rf.me, rf.MatchIndex)
	return matchIndex[(length-1)/2]
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.Log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	d.Decode(&rf.CurrentTerm)
	d.Decode(&rf.VotedFor)
	d.Decode(&rf.Log)
	rf.CommitIndex = rf.Log[0].CommandIndex
	rf.LastApplied = rf.CommitIndex
}

func (rf *Raft) readSnapshot(data []byte) {
	if data == nil || len(data) < 1 {
		return
	}

	msg := ApplyMsg{UseSnapshot: true, Snapshot: data}

	go func() {
		rf.ApplyCh <- msg
	}()
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *Common.RequestVoteArgs, reply *Common.RequestVoteReply) {
	r := simulation.RequestVoteReply{}
	if rf.sl != nil {
		r = rf.sl.HandleRequestMessage("Raft.RequestVote", args).(simulation.RequestVoteReply)
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	defer rf.sl.CheckReplyMessage("Raft.RequestVote", reply, r)

	reply.From = rf.me
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.State = Follower
		rf.VotedFor = -1
		rf.persist()
	}
	reply.Term = rf.CurrentTerm

	if args.LastLogTerm < rf.getLastCommandTerm() {
		reply.VoteGranted = false
		return
	} else if args.LastLogTerm == rf.getLastCommandTerm() {
		if args.LastLogIndex < rf.getLastCommandIndex() {
			//////fmt.Printf("server %d reject the vote request from server %d because the log is shorter\n", rf.me, args.CandidateId)
			reply.VoteGranted = false
			return
		}
	}

	if rf.VotedFor == -1 {
		rf.VotedFor = args.CandidateId
		reply.VoteGranted = true
		go func() {
			rf.RequestVoteCh <- struct{}{}
		}()
		return
	}
	reply.VoteGranted = false
}

//
// example AppendEntries RPC arguments structure.
// field names must start with capital letters!
//

func (rf *Raft) getIndexBeforeTerm(realIndex int) int {

	index := rf.calculateLocalIndex(realIndex)
	if index <= 0 {
		return realIndex
	}

	term := rf.Log[index].CommandTerm

	var i int
	for i = index; i >= 0; i-- {
		if rf.Log[i].CommandTerm == term {
			continue
		} else {
			break
		}
	}

	return rf.calculateRealIndex(i + 1)
}

func (rf *Raft) testStaleRPC(argsPrevLogIndex int, argsPrevLogTerm int, entries []Common.Command) bool {
	index := rf.calculateLocalIndex(argsPrevLogIndex)
	if index >= 0 {
		if rf.Log[index].CommandTerm != argsPrevLogTerm {
			return false
		}
	}
	var theSame bool
	theSame = true
	for i := 0; i < len(entries); i++ {
		if index+i+1 < 0 {
			continue
		} else {
			if rf.Log[index+i+1] != entries[i] {
				theSame = false
				break
			}
		}
	}
	return theSame
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) AppendEntries(args *Common.AppendEntriesArgs, reply *Common.AppendEntriesReply) {
	//fmt.Printf("server %d : APPENDENTRIES from server %d local term %d args term %d args.prevlogindex %d, commitindex %d leader commitindex %d last index %d entries: %+v\n", rf.me, args.LeaderId, rf.CurrentTerm, args.Term, args.PrevLogIndex, rf.CommitIndex, args.LeaderCommit, rf.getLastCommandIndex(), args.Entries)
	//////fmt.Println("enter: AppendEntries");
	r := simulation.AppendEntriesReply{}
	if rf.sl != nil {
		r = rf.sl.HandleRequestMessage("Raft.AppendEntries", args).(simulation.AppendEntriesReply)
	}

	reply.Success = false
	reply.From = rf.me
	rf.mu.Lock()
	defer rf.sl.CheckReplyMessage("Raft.AppendEntries", reply, r)
	defer rf.mu.Unlock()
	defer rf.persist()
	//rf.checkIndx()

	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
	}

	reply.Term = rf.CurrentTerm

	//check if the RPC is a stale RPC
	if args.PrevLogIndex+len(args.Entries) <= rf.getLastCommandIndex() {
		if rf.testStaleRPC(args.PrevLogIndex, args.PrevLogTerm, args.Entries) {
			//update commitIndex if necessary
			//local highest index
			highestIndex := rf.getLastCommandIndex()
			//update
			if args.LeaderCommit > rf.CommitIndex {
				if args.LeaderCommit > highestIndex {
					rf.CommitIndex = highestIndex
				} else {
					rf.CommitIndex = args.LeaderCommit
				}
			}
			rf.CurrentTerm = args.Term
			reply.Success = true
			go func() {
				rf.AppendEntryCh <- struct{}{}
			}()
			return
		}
	}

	//the log of the receiver isn't long enough.
	if rf.getLastCommandIndex() < args.PrevLogIndex {
		reply.Success = false
		reply.NextIndex = rf.getLastCommandIndex()
		return
	}

	matchIndex := rf.calculateLocalIndex(args.PrevLogIndex)

	//the log of the receiver doesn't match.
	if matchIndex > -1 && rf.Log[matchIndex].CommandTerm != args.PrevLogTerm {
		//////fmt.Printf("last index: %d, matchindex: %d, length: %d log: %+v\n", rf.getLastCommandIndex(), matchIndex, len(rf.Log), rf.Log[matchIndex])
		reply.Success = false
		reply.NextIndex = rf.getIndexBeforeTerm(args.PrevLogIndex)
		//////fmt.Printf("server %d refuses the APPEND because it is unmatched, local term: %+v, prevlogterm: %+v\n", rf.me, rf.Log[matchIndex], args.PrevLogTerm)
		rf.Log = rf.Log[0:matchIndex]
		return
	} else {
		//add the log entries to local log
		if matchIndex+1 > 0 {
			//if the local log is empty
			rf.Log = rf.Log[0 : matchIndex+1]
			rf.Log = append(rf.Log, args.Entries...)
		} else {
			rf.Log = args.Entries[-matchIndex-1:]
		}

		//update commitIndex if necessary
		//local highest index
		highestIndex := rf.getLastCommandIndex()
		//update
		if args.LeaderCommit > rf.CommitIndex {
			if args.LeaderCommit > highestIndex {
				rf.CommitIndex = highestIndex
			} else {
				rf.CommitIndex = args.LeaderCommit
			}
		}

		//update current term
		//rf.CurrentTerm = args.Term

		//reply success
		reply.Success = true

		//////fmt.Printf("APPENDENTRIES from server %d to server: %d append successfully at Index %d length %d entries: %+v\n", args.LeaderId, rf.me, args.PrevLogIndex + 1, len(args.Entries), args.Entries)
		go func() {
			rf.AppendEntryCh <- struct{}{}
		}()
	}
	return
}

type InstallSnapshotArgs struct {
	Term          int
	LeaderId      int
	SnapshotIndex int
	SnapshotData  []byte

	LastSnapshotCommand Common.Command
}

type InstallSnapshotReply struct {
	Term    int
	Success bool
}

func (rf *Raft) StartSnapshot(snapshot []byte, lastCommand ApplyMsg) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	/*defer func() {
		//////fmt.Printf("applyIndex: %d\n", rf.LastApplied)
	}()*/

	baseIndex := rf.Log[0].CommandIndex
	lastIndex := rf.getLastCommandIndex()
	index := lastCommand.CommandIndex
	if index <= baseIndex || index > lastIndex {
		return
	}

	var newLogEntries []Common.Command

	localIndex := rf.calculateLocalIndex(index)
	newLogEntries = append(newLogEntries, rf.Log[localIndex:]...)
	rf.Log = newLogEntries
	rf.persist()
	//////fmt.Printf("startSnapshot: server %d, applyMSG: %+v, log start: %+v\n", rf.me, lastCommand, rf.Log[0])
	w := new(bytes.Buffer)

	data := w.Bytes()
	data = append(data, snapshot...)
	rf.persister.SaveSnapshot(data)
}

func (rf *Raft) ReadSnapshotData() []byte {
	return rf.persister.ReadSnapshot()
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	/*defer func() {
		//////fmt.Printf("applyIndex: %d\n", rf.LastApplied)
	}()*/
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		reply.Success = false
		return
	}
	if args.SnapshotIndex < rf.Log[0].CommandIndex {
		reply.Success = false
		return
	}

	go func() {
		rf.AppendEntryCh <- struct{}{}
	}()

	rf.persister.SaveSnapshot(args.SnapshotData)

	rf.Log = make([]Common.Command, 1)
	rf.Log[0] = args.LastSnapshotCommand
	rf.LastApplied = args.LastSnapshotCommand.CommandIndex
	rf.CommitIndex = args.LastSnapshotCommand.CommandIndex
	rf.persist()

	reply.Success = true

	msg := ApplyMsg{UseSnapshot: true, Snapshot: args.SnapshotData}
	go func() {
		rf.ApplyCh <- msg
	}()
}

//
// update the state of server. if the state hasn't been changed, do nothing
//
func (rf *Raft) updateState(state int) {
	if rf.State == state {
		return
	}

	switch state {
	case Follower:
		rf.State = Follower
		rf.VotedFor = -1
		rf.persist()
		return
	case Leader:
		rf.State = Leader
		rf.initNextIndexAndMatchIndex()
		rf.Start(0)
		return
	}
}

//
// start election

// broadcast requestVote
func (rf *Raft) broadcastRequestVote(args *Common.RequestVoteArgs) {
	rf.mu.Lock()
	term := rf.CurrentTerm
	rf.mu.Unlock()
	//fmt.Printf("server %d enter: broadcastRequestVote, current term: %d, current state: %v\n", rf.me, rf.CurrentTerm, rf.State)
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		} else {
			go rf.sendSingleRequestVote(i, args, term)
		}
	}
	//fmt.Printf("server %d finishes: broadcastRequestVote\n", rf.me)
	//rf.mu.Unlock()
}

func (rf *Raft) sendSingleRequestVote(server int, args *Common.RequestVoteArgs, term int) {

	if rf.getState() != Candidate || rf.getTerm() != term {
		return
	}
	var reply Common.RequestVoteReply
	//////fmt.Printf("server %d wants to send REQUEST VOTE RPC to server %d with args %+v\n", rf.me, server, args)
	if rf.getState() == Candidate && rf.getTerm() == term && rf.sendRequestVote(server, args, &reply) {
		//////fmt.Printf("server %d receives the reply from server %d success %t\n", rf.me, server, reply.VoteGranted)
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.State != Candidate || rf.CurrentTerm != term {
			return
		}
		if reply.VoteGranted == false {
			////fmt.Printf("server %d from server %d : FAILED TO GET ONE VOTE !!!!!!!\n", rf.me, server)
		}
		if reply.VoteGranted == true {
			rf.Votes += 1
			//fmt.Printf("server %d from server %d : GET ONE VOTE !!!!!!! current vote: %d\n", rf.me, server, rf.Votes)
			rf.testLeader()
		} else if reply.Term > rf.CurrentTerm {
			rf.CurrentTerm = reply.Term
			rf.updateState(Follower)
		}
	}
}

//broadcast appendEntriesRPC or heartbeat
func (rf *Raft) broadcastAppendEntries() {
	//////fmt.Printf("leader %d enter: broadcastAppendEntries", rf.me);
	rf.mu.Lock()
	//fmt.Printf("server %d enter: broadcastAppendEntries\n", rf.me)
	term := rf.CurrentTerm
	rf.mu.Unlock()

	for i, _ := range rf.peers {
		if i == rf.me {
			rf.MatchIndex[rf.me] = rf.getLastCommandIndex()
			continue
		} else {
			if rf.getState() != Leader {
				break
			}
			go func(server int) {
				for {
					if rf.sendSingleAppendEntries(server, term) {
						break
					}
				}
			}(i)
		}
	}
	//fmt.Printf("server %d finishes: broadcastAppendEntires\n", rf.me)
	//rf.mu.Unlock()
}

//send appendEntriesRPC, return true means that do not need to retry
func (rf *Raft) sendSingleAppendEntries(server int, term int) bool {
	////////fmt.Println("enter: sendSingleAppendEntries");
	rf.mu.Lock()
	if rf.State != Leader || rf.CurrentTerm != term {
		rf.mu.Unlock()
		return true
	}
	if rf.NextIndex[server] <= rf.Log[0].CommandIndex {
		var args InstallSnapshotArgs
		args.LeaderId = rf.me
		args.Term = term
		args.SnapshotData = rf.persister.ReadSnapshot()
		args.LastSnapshotCommand = rf.Log[0]
		rf.mu.Unlock()
		rf.sendSingleInstallSnapshot(server, &args, term)
		return true
	}
	var args Common.AppendEntriesArgs
	args.Term = term
	args.LeaderId = rf.me
	args.LeaderCommit = rf.CommitIndex
	args.PrevLogIndex = rf.NextIndex[server] - 1
	localPrevLogIndex := rf.calculateLocalIndex(args.PrevLogIndex)
	if localPrevLogIndex >= 0 {
		//////fmt.Printf("prevlogindex: %d local prev log index:%d baseindex: %d\n", args.PrevLogIndex, localPrevLogIndex, rf.Log[0].CommandIndex)
		args.PrevLogTerm = rf.Log[localPrevLogIndex].CommandTerm
	}
	lastLogIndex := len(rf.Log) - 1
	if lastLogIndex > localPrevLogIndex {
		args.Entries = rf.Log[localPrevLogIndex+1:]
		//////fmt.Printf("server %d to server %d : sends APPEND ENTRY RPC with args %+v\n", rf.me, server, args)
	}
	rf.mu.Unlock()

	var reply Common.AppendEntriesReply
	if rf.getState() == Leader && rf.getTerm() == term && rf.sendAppendEntries(server, &args, &reply) {
		//handle the reply should acquire the lock
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.State != Leader || rf.CurrentTerm != term {
			return true
		}
		if reply.Success == false {
			//if during the server waits for the reply, the leader has been changed
			if reply.Term > rf.CurrentTerm {
				//////fmt.Printf("leader %d receive a reply with larger term \n", rf.me)
				rf.CurrentTerm = reply.Term
				rf.State = Follower
				rf.VotedFor = -1
				rf.persist()
				return true
			} else {
				//decrement the next index and retry
				tmp := rf.getIndexBeforeTerm(args.PrevLogIndex)
				if tmp < reply.NextIndex {
					rf.NextIndex[server] = tmp
				} else {
					rf.NextIndex[server] = reply.NextIndex
				}
				//////fmt.Printf("leader %d wants to decrement follower %d 's nextIndex %d\n", rf.me, server, rf.NextIndex[server])
				return false
			}
		} else {
			//rf.NextIndex[server] += len(args.Entries)
			tmp := args.PrevLogIndex + len(args.Entries) + 1
			if rf.NextIndex[server] < tmp {
				rf.NextIndex[server] = tmp
			}
			rf.MatchIndex[server] = rf.NextIndex[server] - 1
			////fmt.Printf("leader %d wants to update array of followers %d matchIndex %d reply: %+v args: %+v\n", rf.me, server, rf.MatchIndex[server], reply, args)
			return true
		}
	}
	return true
}

func (rf *Raft) sendSingleInstallSnapshot(server int, args *InstallSnapshotArgs, term int) {
	if rf.getState() != Leader || rf.getTerm() != term {
		return
	}
	var reply InstallSnapshotReply
	//////fmt.Printf("server %d wants to send INSTALLSNAPSHOT RPC to server %d with args %+v\n", rf.me, server, args)
	if rf.getState() == Leader && rf.getTerm() == term && rf.sendInstallSnapshot(server, args, &reply) {
		//////fmt.Printf("server %d receives the reply from server %d success %t\n", rf.me, server, reply.Success)
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.State != Leader || rf.CurrentTerm != term {
			return
		}
		if reply.Success == true {
			rf.NextIndex[server] = args.LastSnapshotCommand.CommandIndex + 1
			rf.MatchIndex[server] = rf.NextIndex[server] - 1
		} else if reply.Term > rf.CurrentTerm {
			rf.CurrentTerm = reply.Term
			rf.updateState(Follower)
		}
	}
}

/*func (rf *Raft) broadcastInstallSnapshot(args *InstallSnapshotArgs) {
	rf.mu.Lock()
	term := rf.CurrentTerm
	rf.mu.Unlock()
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		} else {
			rf.sendSingleInstallSnapshot(i, args, term)
		}
	}

}*/

//check the commit index and update it if necessary
func (rf *Raft) checkCommitIndex() {
	//////fmt.Printf("leader %d enter: checkCommitIndex\n", rf.me)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	commitIndex := rf.findMajority()
	////fmt.Printf("leader: %d, majority: %d, current commit index: %d, length: %d\n", rf.me, commitIndex, rf.CommitIndex, len(rf.Log));
	if rf.CommitIndex < commitIndex {
		//localCommitIndex := rf.calculateLocalIndex(commitIndex)
		//if rf.Log[localCommitIndex].CommandTerm == rf.CurrentTerm {
		//	rf.CommitIndex = commitIndex
		//}
		//this is incorrect, since a leader can only commit the entries until its term
		rf.CommitIndex = commitIndex
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *Common.RequestVoteArgs, reply *Common.RequestVoteReply) bool {
	args.From = rf.me
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *Common.AppendEntriesArgs, reply *Common.AppendEntriesReply) bool {
	//fmt.Printf("server %d sends append entries to %d\n", rf.me, server)
	args.From = rf.me
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	term, isLeader = rf.GetState()
	if isLeader == false {
		return -1, -1, false
	} else {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		index = rf.getLastCommandIndex() + 1
		rf.Log = append(rf.Log, Common.Command{CommandIndex: index, Command: command, CommandTerm: term})
		rf.persist()
		//update local record
		rf.NextIndex[rf.me] = index + 1
		rf.MatchIndex[rf.me] = index
		go func() {
			rf.broadcastAppendEntries()
		}()
		//rf.broadcastAppendEntries()
		////fmt.Printf("leader server %d receive a new command at index: %d local term: %d, votes: %d command %+v\n", rf.me, index, rf.CurrentTerm, rf.Votes, command)
		return index, term, isLeader
	}
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
	rf.mu.Lock()
	rf.Process = true
	rf.mu.Unlock()
}

//update the server itself before it calls for a vote request
func (rf *Raft) updateBeforeRequestVote() {
	rf.mu.Lock()
	rf.CurrentTerm = rf.CurrentTerm + 1
	rf.VotedFor = rf.me
	rf.persist()
	rf.Votes = 1
	rf.Timer.Reset(rf.generate_rand())
	rf.mu.Unlock()
}

func (rf *Raft) updateToCandidate() {
	//////fmt.Printf("server %d became CANDIDATE\n", rf.me)
	if rf.getState() != Candidate {
		rf.setState(Candidate)
	}
	rf.updateBeforeRequestVote()

	var args Common.RequestVoteArgs
	rf.mu.Lock()
	args.Term = rf.CurrentTerm
	args.CandidateId = rf.me
	args.LastLogIndex = rf.getLastCommandIndex()
	args.LastLogTerm = rf.getLastCommandTerm()
	rf.mu.Unlock()
	rf.broadcastRequestVote(&args)
}

func (rf *Raft) resetTimer() {
	rf.mu.Lock()
	rf.Timer.Reset(rf.generate_rand())
	rf.mu.Unlock()
}

func (rf *Raft) updateToFollower() {
	rf.mu.Lock()
	rf.State = Follower
	rf.VotedFor = -1
	rf.mu.Unlock()
}

func (rf *Raft) testLeader() {
	//rf.mu.Lock()
	//fmt.Printf("server %d tests the leader, current term: %d, votes: %d\n", rf.me, rf.CurrentTerm, rf.Votes)
	if rf.Votes > len(rf.peers)/2 {
		////fmt.Printf("server %d becomes the leader, current term: %d\n", rf.me, rf.CurrentTerm)
		rf.State = Leader
		rf.initNextIndexAndMatchIndex()
	}
	//rf.mu.Unlock()
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//

func (rf *Raft) initNextIndexAndMatchIndex() {
	//////fmt.Printf("server %d became LEADER\n", rf.me)
	var i int
	for i = 0; i < len(rf.peers); i++ {
		rf.NextIndex[i] = rf.getLastCommandIndex() + 1
	}
	for i = 0; i < len(rf.peers); i++ {
		rf.MatchIndex[i] = 0
	}
	rf.MatchIndex[rf.me] = rf.getLastCommandIndex()
}

func (rf *Raft) checkIndx() {
	for i := 0; i < len(rf.Log)-1; i++ {
		if rf.Log[i].CommandIndex != rf.Log[i+1].CommandIndex-1 {
			//////fmt.Printf("server %d log bug: index 1: %d, index 2: %d, command: %+v\n", rf.me, rf.Log[i].CommandIndex, rf.Log[i+1].CommandIndex, rf.Log[i + 1])
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
}

func MakeWithSL(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg, sl *simulation.Simulation) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).

	rf.CurrentTerm = 0
	rf.VotedFor = -1
	rf.State = Follower
	rf.Votes = 0

	rf.AppendEntryCh = make(chan struct{})
	rf.RequestVoteCh = make(chan struct{})

	rf.CommitIndex = 0
	rf.LastApplied = 0
	rf.sl = sl
	rf.block_elec = false

	rf.Log = make([]Common.Command, 1)
	rf.Log[0] = Common.Command{CommandIndex: 0}
	rf.NextIndex = make([]int, len(rf.peers))
	rf.MatchIndex = make([]int, len(rf.peers))
	rf.initNextIndexAndMatchIndex()

	rf.ApplyCh = applyCh
	// initialize from state persisted before a crash

	rf.readPersist(persister.ReadRaftState())
	rf.readSnapshot(persister.ReadSnapshot())

	go func() {
		rf.Timer = time.NewTimer(rf.generate_rand())
		for {
			if rf.Process == true {
				break
			}
			switch rf.getState() {
			case Follower:
				select {
				case <-rf.AppendEntryCh:
					rf.resetTimer()
				case <-rf.RequestVoteCh:
					rf.resetTimer()
				case <-rf.Timer.C:
					if rf.block_elec {
						continue
					}
					if rf.sl != nil {
						rf.sl.CheckRequestMessage("Timeout", nil)
						rf.sl.HandleRequestMessage("Timeout", nil)
					}
					rf.updateToCandidate()
				}
			case Candidate:
				select {
				case <-rf.AppendEntryCh:
					rf.resetTimer()
					rf.updateToFollower()
				case <-rf.Timer.C:
					rf.Timer.Reset(rf.generate_rand())
					if rf.sl != nil {
						rf.sl.CheckRequestMessage("Timeout", nil)
						rf.sl.HandleRequestMessage("Timeout", nil)
					}
					rf.updateToCandidate()
				default:
					//rf.testLeader()
				}
			case Leader:
				//////fmt.Printf("server %d is the leader\n", rf.me)
				select {
				case <-rf.AppendEntryCh:
					rf.updateToFollower()
					rf.resetTimer()
				default:
					rf.broadcastAppendEntries()
					rf.checkCommitIndex()
					time.Sleep(HEARTBEAT * time.Millisecond)
				}
			}
			//rf.checkIndx()
			rf.mu.Lock()
			if rf.CommitIndex > rf.LastApplied {
				//////fmt.Printf("server %d commitIndex: %d, applyindex: %d\n", rf.me, rf.CommitIndex, rf.LastApplied)
				localCommitIndex := rf.calculateLocalIndex(rf.CommitIndex)
				localAppliedIndex := rf.calculateLocalIndex(rf.LastApplied)
				//////fmt.Printf("server %d localCommitIndex: %d, localAppliedIndex: %d, length of log %d\n", rf.me, localCommitIndex, localAppliedIndex, len(rf.Log))
				var i int
				for i = localAppliedIndex + 1; i <= localCommitIndex; i++ {
					var msg ApplyMsg
					msg.CommandValid = true
					msg.Command = rf.Log[i].Command
					msg.CommandIndex = rf.Log[i].CommandIndex
					applyCh <- msg
					//////fmt.Printf("server %d commit the command at index %d command: %+v \n", rf.me, rf.calculateRealIndex(i), rf.Log[i])
					rf.LastApplied++
				}
			}
			rf.mu.Unlock()

		}
	}()
	return rf
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).

	rf.CurrentTerm = 0
	rf.VotedFor = -1
	rf.State = Follower
	rf.Votes = 0

	rf.AppendEntryCh = make(chan struct{})
	rf.RequestVoteCh = make(chan struct{})

	rf.CommitIndex = 0
	rf.LastApplied = 0

	rf.Log = make([]Common.Command, 1)
	rf.Log[0] = Common.Command{CommandIndex: 0}
	rf.NextIndex = make([]int, len(rf.peers))
	rf.MatchIndex = make([]int, len(rf.peers))
	rf.initNextIndexAndMatchIndex()

	rf.ApplyCh = applyCh
	// initialize from state persisted before a crash

	rf.readPersist(persister.ReadRaftState())
	rf.readSnapshot(persister.ReadSnapshot())

	go func() {
		////fmt.Printf("start the raft server, %d", rf.me)
		rf.Timer = time.NewTimer(rf.generate_rand())
		for {
			if rf.Process == true {
				break
			}
			switch rf.getState() {
			case Follower:
				select {
				case <-rf.AppendEntryCh:
					rf.resetTimer()
				case <-rf.RequestVoteCh:
					rf.resetTimer()
				case <-rf.Timer.C:
					rf.updateToCandidate()
				}
			case Candidate:
				select {
				case <-rf.AppendEntryCh:
					rf.resetTimer()
					rf.updateToFollower()
				case <-rf.Timer.C:
					rf.Timer.Reset(rf.generate_rand())
					rf.updateToCandidate()
				default:
					//rf.testLeader()
				}
			case Leader:
				//////fmt.Printf("server %d is the leader\n", rf.me)
				select {
				case <-rf.AppendEntryCh:
					rf.updateToFollower()
					rf.resetTimer()
				default:
					rf.broadcastAppendEntries()
					rf.checkCommitIndex()
					time.Sleep(HEARTBEAT * time.Millisecond)
				}
			}
			//rf.checkIndx()
			rf.mu.Lock()
			if rf.CommitIndex > rf.LastApplied {
				//////fmt.Printf("server %d commitIndex: %d, applyindex: %d\n", rf.me, rf.CommitIndex, rf.LastApplied)
				localCommitIndex := rf.calculateLocalIndex(rf.CommitIndex)
				localAppliedIndex := rf.calculateLocalIndex(rf.LastApplied)
				//////fmt.Printf("server %d localCommitIndex: %d, localAppliedIndex: %d, length of log %d\n", rf.me, localCommitIndex, localAppliedIndex, len(rf.Log))
				var i int
				for i = localAppliedIndex + 1; i <= localCommitIndex; i++ {
					var msg ApplyMsg
					msg.CommandValid = true
					msg.Command = rf.Log[i].Command
					msg.CommandIndex = rf.Log[i].CommandIndex
					applyCh <- msg
					//////fmt.Printf("server %d commit the command at index %d command: %+v \n", rf.me, rf.calculateRealIndex(i), rf.Log[i])
					rf.LastApplied++
				}
			}
			rf.mu.Unlock()

		}
	}()

	return rf
}
