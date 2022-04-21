from z3 import *
Type, (follower, candidate, leader) = EnumSort("all_states", ["follower", "candidate", "leader"])
log_entry, mk_log_entry, (EntryTerm, EntryContent) = TupleSort('logentry', [IntSort(), IntSort()])
state, mk_states, (Term, State, Voted, Log, LogLength, VotesGranted, CommitIndex) = TupleSort('states', [IntSort(), Type, IntSort(), ArraySort(IntSort(), log_entry), IntSort(), SetSort(IntSort()), IntSort()])

requestVoteReqMsg, mk_requestVoteReqMsg, (RequestVoteReqMsgTerm, RequestVoteReqMsgFrom, LastLogTerm, LastLogIndex) = TupleSort('requestVoteReqMsg', [IntSort(), IntSort(), IntSort(), IntSort()])
requestVoteResponseMsg, mk_requestVoteResponseMsg, (RequestVoteResponseMsgTerm, RequestVoteResponseMsgFrom, RequestVoteResponseMsgGranted) = TupleSort('requestVoteResponseMsg', [IntSort(), IntSort(), BoolSort()])
appendEntriesReqMsg, mk_appendEntriesReqMsg, (AppendEntriesReqMsgTerm, AppendEntriesReqMsgFrom, AppendEntriesReqMsgPrevLogIndex, AppendEntriesReqMsgPrevLogTerm, AppendEntriesReqMsgEntry, AppendEntriesReqMsgCommitIndex) \
    = TupleSort('appendEntriesReqMsg', [IntSort(), IntSort(), IntSort(), IntSort(), log_entry, IntSort()])


TERM = 1
STATE = 2
VOTED = 3
LOG = 4
LOGLENGTH = 5
VOTESGRANTED = 6
COMMITINDEX = 7


# helper functions:
def getAllStates(current_states):
    current_term = Term(current_states)
    current_state = State(current_states)
    current_voted = Voted(current_states)
    current_log = Log(current_states)
    current_log_length = LogLength(current_states)
    current_votes_granted = VotesGranted(current_states)
    current_commit_index = CommitIndex(current_states)
    return current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index

def updateOneState(current_states, state_no, new_state):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    if state_no == TERM:
        return mk_states(new_state, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index)
    if state_no == STATE:
        return mk_states(current_term, new_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index)
    if state_no == VOTED:
        return mk_states(current_term, current_state, new_state, current_log, current_log_length, current_votes_granted, current_commit_index)
    if state_no == LOG:
        return mk_states(current_term, current_state, current_voted, new_state, current_log_length, current_votes_granted, current_commit_index)
    if state_no == LOGLENGTH:
        return mk_states(current_term, current_state, current_voted, current_log, new_state, current_votes_granted, current_commit_index)
    if state_no == VOTESGRANTED:
        return mk_states(current_term, current_state, current_voted, current_log, current_log_length, new_state, current_commit_index)
    if state_no == COMMITINDEX:
        return mk_states(current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, new_state)

def updateStates(current_states, state_nos, new_states):
    for i, val in enumerate(state_nos):
        current_states = updateOneState(current_states, val, new_states[i])
    return current_states

def serverVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    return current_term, current_state, current_voted

def serverVarsEqual(s1, s2):
    current_term1, current_state1, current_voted1 = serverVars(s1)
    current_term2, current_state2, current_voted2 = serverVars(s2)
    return z3.If(And(current_term1 == current_term2, And(current_state1 == current_state2, current_voted1 == current_voted2)), True, False)

def logVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    return current_log, current_log_length, current_commit_index

def logVarsEqual(s1, s2):
    current_log1, current_log_length1, current_commit_index1 = logVars(s1)
    current_log2, current_log_length2, current_commit_index2 = logVars(s2)
    return z3.If(And(current_log1 == current_log2, And(current_log_length1 == current_log_length2, current_commit_index1 == current_commit_index2)), True, False)

def candidateVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    return current_votes_granted

def candidateVarsEqual(s1, s2):
    current_votes_granted1 = candidateVars(s1)
    current_votes_granted2 = candidateVars(s2)
    return z3.If(current_votes_granted1 == current_votes_granted2, True, False)

def leaderVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    return 


def followerEqual(s1, s2):
    return z3.If(And(serverVarsEqual(s1, s2), logVarsEqual(s1, s2)), True, False)

def candidateEqual(s1, s2):
    return z3.If(And(And(serverVarsEqual(s1, s2), logVarsEqual(s1, s2)), candidateVarsEqual(s1, s2)), True, False)

def leaderEqual(s1, s2):
    return z3.If(And(serverVarsEqual(s1, s2), logVarsEqual(s1, s2)), True, False)

def stateEqual(s1, s2):
    return z3.If(State(s1) != State(s2), False, \
                z3.If(Or(And(State(s1) == follower, followerEqual(s1, s2)), \
                    Or(And(State(s1) == candidate, candidateEqual(s1, s2)), \
                        And(State(s1) == leader, leaderEqual(s1, s2)))), True, False))

# def stateEqual(s1, s2):
#     return s1 == s2

def logUpdate(current_states, prevIndex, entry):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)
    new_log = Store(current_log, prevIndex + 1, entry)
    return z3.If(current_log_length == prevIndex, updateStates(current_states, [LOG, LOGLENGTH], [new_log, current_log_length + 1]), updateStates(current_states, [LOG], [new_log]))

# msg handlers
def updateTerm(current_states, msg_term):
    term = Term(current_states)
    return z3.If(term < msg_term, updateStates(current_states, [TERM, STATE, VOTED], [msg_term, follower, 0]), current_states)
    # return z3.If(term < msg_term, updateStates(current_states, [TERM, STATE, VOTED, VOTESGRANTED], [msg_term, follower, 0, EmptySet(IntSort())]), current_states)

def stepDownToFollower(current_states, msg_term):
    term = Term(current_states)
    return z3.If(term <= msg_term, updateStates(current_states, [TERM, STATE], [msg_term, follower]), current_states)

def requestVoteReqHandler(current_states, msg):
    message_term = RequestVoteReqMsgTerm(msg)
    message_from = RequestVoteReqMsgFrom(msg)
    message_last_log_term = LastLogTerm(msg)
    message_last_log_index = LastLogIndex(msg)

    current_states = updateTerm(current_states, message_term)
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)

    logOK = Or(message_last_log_term > EntryTerm(current_log[current_log_length]), \
                And(message_last_log_term == EntryTerm(current_log[current_log_length]), \
                        message_last_log_index > current_log_length))
    grant = And(message_term == current_term, \
                And(logOK, \
                    Or(current_voted == 0, current_voted == message_from)))

    return z3.If(grant, updateStates(current_states, [VOTED], [message_from]), current_states)

def requestVoteResponseHandler(current_states, msg):
    message_term = RequestVoteResponseMsgTerm(msg)
    message_from = RequestVoteResponseMsgFrom(msg)
    message_grant = RequestVoteResponseMsgGranted(msg)

    current_states = updateTerm(current_states, message_term)
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)

    return z3.If(message_term != current_term, current_states, \
                z3.If(message_grant, updateStates(current_states, [VOTESGRANTED], [SetAdd(current_votes_granted, message_from)]), current_states))

def appendEntryReqHandler(current_states, msg):
    message_term = AppendEntriesReqMsgTerm(msg)
    message_from = AppendEntriesReqMsgTerm(msg)
    message_prevLogIndex = AppendEntriesReqMsgPrevLogIndex(msg)
    message_prevLogTerm = AppendEntriesReqMsgPrevLogTerm(msg)
    message_entry = AppendEntriesReqMsgEntry(msg)
    message_commitIndex = AppendEntriesReqMsgCommitIndex(msg)

    current_states = updateTerm(current_states, message_term)
    current_states = stepDownToFollower(current_states, message_term)
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index = getAllStates(current_states)

    logOK = Or(message_prevLogIndex == 0, And(message_prevLogIndex > 0, And(message_prevLogIndex <= current_log_length, message_prevLogTerm == EntryTerm(current_log[message_prevLogIndex]))))
    return z3.If(message_term != current_term, current_states, \
                z3.If(logOK != True, current_states, \
                    z3.If(current_commit_index > message_commitIndex, logUpdate(current_states, message_prevLogIndex, message_entry), \
                    updateStates(logUpdate(current_states, message_prevLogIndex, message_entry), [COMMITINDEX], [message_commitIndex]))))



s = Solver()
states = mk_states(Int('term'), Const('state', Type), Int('voted'), Array('log', IntSort(), log_entry), Int('loglength'), Const('votesgranted', SetSort(IntSort())), Int('commitindex'))

def test1():
    # test1: whether two requestVoteReq are commutative
    m1 = mk_requestVoteReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_last_log_term'), Int('m1_last_log_index'))
    m2 = mk_requestVoteReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_last_log_term'), Int('m2_last_log_index'))

    s.add(RequestVoteReqMsgTerm(m1) != RequestVoteReqMsgTerm(m2))
    s.add(stateEqual(requestVoteReqHandler(requestVoteReqHandler(states, m1), m2), requestVoteReqHandler(requestVoteReqHandler(states, m2), m1)) == False)

def test2():
    # test2: whether two requestVoteResponse are commutative
    m1 = mk_requestVoteResponseMsg(Int('m1_term'), Int('m1_from'), Bool('m1_granted'))
    m2 = mk_requestVoteResponseMsg(Int('m2_term'), Int('m2_from'), Bool('m2_granted'))
    s.add(z3.If(RequestVoteResponseMsgGranted(m1), RequestVoteResponseMsgTerm(m1) <= Term(states), True))
    s.add(z3.If(RequestVoteResponseMsgGranted(m2), RequestVoteResponseMsgTerm(m2) <= Term(states), True))
    s.add(stateEqual(requestVoteResponseHandler(requestVoteResponseHandler(states, m1), m2), requestVoteResponseHandler(requestVoteResponseHandler(states, m2), m1)) == False)

def test3():
    # test3: whether two appendEntriesReq are commutative
    m1 = mk_appendEntriesReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_prelogindex'), Int('m1_prelogterm'), Const('entry1', log_entry), Int('m1_commitindex'))
    m2 = mk_appendEntriesReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_prelogindex'), Int('m2_prelogterm'), Const('entry2', log_entry), Int('m2_commitindex'))
    s.add(AppendEntriesReqMsgPrevLogIndex(m1) == AppendEntriesReqMsgPrevLogIndex(m2))
    s.add(AppendEntriesReqMsgPrevLogTerm(m1) == AppendEntriesReqMsgPrevLogTerm(m2))
    s.add(AppendEntriesReqMsgEntry(m1) == AppendEntriesReqMsgEntry(m2))
    s.add(AppendEntriesReqMsgTerm(m1) == AppendEntriesReqMsgTerm(m2))
    s.add(stateEqual(appendEntryReqHandler(appendEntryReqHandler(states, m1), m2), appendEntryReqHandler(appendEntryReqHandler(states, m2), m1)) == False)

# test1()
# test2()
test3()
result = s.check()
print(result)
if result == sat:
    print(s.model())