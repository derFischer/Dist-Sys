from z3 import *
Type, (follower, candidate, leader) = EnumSort("all_states", ["follower", "candidate", "leader"])
log_entry, mk_log_entry, (EntryTerm, EntryContent) = TupleSort('logentry', [IntSort(), IntSort()])
state, mk_states, (Term, State, Voted, Log, LogLength, VotesGranted, CommitIndex, MatchIndex) = TupleSort('states', [IntSort(), Type, IntSort(), ArraySort(IntSort(), log_entry), IntSort(), SetSort(IntSort()), IntSort(), ArraySort(IntSort(), IntSort())])

requestVoteReqMsg, mk_requestVoteReqMsg, (RequestVoteReqMsgTerm, RequestVoteReqMsgFrom, RequestVoteReqMsgLastLogTerm, RequestVoteReqMsgLastLogIndex) = TupleSort('requestVoteReqMsg', [IntSort(), IntSort(), IntSort(), IntSort()])
requestVoteResponseMsg, mk_requestVoteResponseMsg, (RequestVoteResponseMsgTerm, RequestVoteResponseMsgFrom, RequestVoteResponseMsgGranted) = TupleSort('requestVoteResponseMsg', [IntSort(), IntSort(), BoolSort()])
appendEntriesReqMsg, mk_appendEntriesReqMsg, (AppendEntriesReqMsgTerm, AppendEntriesReqMsgFrom, AppendEntriesReqMsgPrevLogIndex, AppendEntriesReqMsgPrevLogTerm, AppendEntriesReqMsgEntry, AppendEntriesReqMsgCommitIndex) \
    = TupleSort('appendEntriesReqMsg', [IntSort(), IntSort(), IntSort(), IntSort(), log_entry, IntSort()])
appendEntriesResponseMsg, mk_appendEntriesResponseMsg, (AppendEntriesResponseMsgTerm, AppendEntriesResponseMsgFrom, AppendEntriesResponseMsgSuccess, AppendEntriesResponseMsgMatchIndex) = TupleSort('appendEntriesResponseMsg', [IntSort(), IntSort(), BoolSort(), IntSort()])

TERM = 1
STATE = 2
VOTED = 3
LOG = 4
LOGLENGTH = 5
VOTESGRANTED = 6
COMMITINDEX = 7
MATCHINDEX = 8


# helper functions:
def getAllStates(current_states):
    current_term = Term(current_states)
    current_state = State(current_states)
    current_voted = Voted(current_states)
    current_log = Log(current_states)
    current_log_length = LogLength(current_states)
    current_votes_granted = VotesGranted(current_states)
    current_commit_index = CommitIndex(current_states)
    current_match_index = MatchIndex(current_states)
    return current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index

def updateOneState(current_states, state_no, new_state):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    if state_no == TERM:
        return mk_states(new_state, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index)
    if state_no == STATE:
        return mk_states(current_term, new_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index)
    if state_no == VOTED:
        return mk_states(current_term, current_state, new_state, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index)
    if state_no == LOG:
        return mk_states(current_term, current_state, current_voted, new_state, current_log_length, current_votes_granted, current_commit_index, current_match_index)
    if state_no == LOGLENGTH:
        return mk_states(current_term, current_state, current_voted, current_log, new_state, current_votes_granted, current_commit_index, current_match_index)
    if state_no == VOTESGRANTED:
        return mk_states(current_term, current_state, current_voted, current_log, current_log_length, new_state, current_commit_index, current_match_index)
    if state_no == COMMITINDEX:
        return mk_states(current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, new_state, current_match_index)
    if state_no == MATCHINDEX:
        return mk_states(current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, new_state)

def updateStates(current_states, state_nos, new_states):
    for i, val in enumerate(state_nos):
        current_states = updateOneState(current_states, val, new_states[i])
    return current_states

def serverVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    return current_term, current_state, current_voted

def serverVarsEqual(s1, s2):
    current_term1, current_state1, current_voted1 = serverVars(s1)
    current_term2, current_state2, current_voted2 = serverVars(s2)
    return z3.If(And(current_term1 == current_term2, And(current_state1 == current_state2, current_voted1 == current_voted2)), True, False)

def logVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    return current_log, current_log_length, current_commit_index

def logVarsEqual(s1, s2):
    current_log1, current_log_length1, current_commit_index1 = logVars(s1)
    current_log2, current_log_length2, current_commit_index2 = logVars(s2)
    return z3.If(And(current_log1 == current_log2, And(current_log_length1 == current_log_length2, current_commit_index1 == current_commit_index2)), True, False)

def candidateVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    return current_votes_granted

def candidateVarsEqual(s1, s2):
    current_votes_granted1 = candidateVars(s1)
    current_votes_granted2 = candidateVars(s2)
    return z3.If(current_votes_granted1 == current_votes_granted2, True, False)

def leaderVars(current_states):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
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

def matchIndexUpdate(current_states, serverID, matchIndex):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    new_matchIndex = Store(current_match_index, serverID, matchIndex)
    return z3.If(current_match_index[serverID] < matchIndex, updateStates(current_states, [MATCHINDEX], [new_matchIndex]), current_states)

def logUpdate(current_states, prevIndex, entry):
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
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
    message_last_log_term = RequestVoteReqMsgLastLogTerm(msg)
    message_last_log_index = RequestVoteReqMsgLastLogIndex(msg)

    current_states = updateTerm(current_states, message_term)
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)

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
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)

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
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)

    logOK = Or(message_prevLogIndex == 0, And(message_prevLogIndex > 0, And(message_prevLogIndex <= current_log_length, message_prevLogTerm == EntryTerm(current_log[message_prevLogIndex]))))
    return z3.If(message_term != current_term, current_states, \
                z3.If(logOK != True, current_states, \
                    z3.If(current_commit_index > message_commitIndex, logUpdate(current_states, message_prevLogIndex, message_entry), \
                    updateStates(logUpdate(current_states, message_prevLogIndex, message_entry), [COMMITINDEX], [message_commitIndex]))))


def appendEntryResponseHandler(current_states, msg):
    message_term = AppendEntriesResponseMsgTerm(msg)
    message_from = AppendEntriesResponseMsgFrom(msg)
    message_success = AppendEntriesResponseMsgSuccess(msg)
    message_matchIndex = AppendEntriesResponseMsgMatchIndex(msg)

    current_states = updateTerm(current_states, message_term)
    current_term, current_state, current_voted, current_log, current_log_length, current_votes_granted, current_commit_index, current_match_index = getAllStates(current_states)
    return z3.If(message_term != current_term, current_states, \
                z3.If(message_success == False, current_states, matchIndexUpdate(current_states, message_from, message_matchIndex)))

states = mk_states(Int('term'), Const('state', Type), Int('voted'), Array('log', IntSort(), log_entry), Int('loglength'), Const('votesgranted', SetSort(IntSort())), Int('commitindex'), Array('matchIndex', IntSort(), IntSort()))

handlers = [requestVoteReqHandler, requestVoteResponseHandler, appendEntryReqHandler, appendEntryResponseHandler]
m1 = [mk_requestVoteReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_last_log_term'), Int('m1_last_log_index')), \
      mk_requestVoteResponseMsg(Int('m1_term'), Int('m1_from'), Bool('m1_granted')), \
      mk_appendEntriesReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_prelogindex'), Int('m1_prelogterm'), Const('entry1', log_entry), Int('m1_commitindex')), \
      mk_appendEntriesResponseMsg(Int('m1_term'), Int('m1_from'), Bool('m1_success'), Int('m1_matchindex'))]
m2 = [mk_requestVoteReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_last_log_term'), Int('m2_last_log_index')), \
      mk_requestVoteResponseMsg(Int('m2_term'), Int('m2_from'), Bool('m2_granted')), \
      mk_appendEntriesReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_prelogindex'), Int('m2_prelogterm'), Const('entry2', log_entry), Int('m2_commitindex')), \
      mk_appendEntriesResponseMsg(Int('m2_term'), Int('m2_from'), Bool('m2_success'), Int('m2_matchindex'))]



message_names = ['requestVoteReqMsg', 'requestVoteResponseMsg', 'appendEntriesReqMsg', 'appendEntriesResponseMsg']

parameters  = [[RequestVoteReqMsgTerm, RequestVoteReqMsgFrom, RequestVoteReqMsgLastLogTerm, RequestVoteReqMsgLastLogIndex], \
               [RequestVoteResponseMsgTerm, RequestVoteResponseMsgFrom, RequestVoteResponseMsgGranted], \
               [AppendEntriesReqMsgTerm, AppendEntriesReqMsgFrom, AppendEntriesReqMsgPrevLogIndex, AppendEntriesReqMsgPrevLogTerm, AppendEntriesReqMsgEntry, AppendEntriesReqMsgCommitIndex], \
               [AppendEntriesResponseMsgTerm, AppendEntriesResponseMsgFrom, AppendEntriesResponseMsgSuccess, AppendEntriesResponseMsgMatchIndex]]

TOTAL_MESSAGE_TYPE = 4

REQUESTVOTEREQTERM = 0
REQUESTVOTEREQFROM = 1
REQUESTVOTELASTLOGTERM = 2
REQUESTVOTELASTLOGINDEX = 3

REQUESTVOTERESPONSETERM = 0
REQUESTVOTERESPONSEFROM = 1
REQUESTVOTERESPONSEGRANTED = 2

APPENDENTRIESREQTERM = 0
APPENDENTRIESREQFROM = 1
APPENDENTRIESREQPREVLOGINDEX = 2
APPENDENTRIESREQPREVLOGTERM = 3
APPENDENTRIESREQENTRY = 4
APPENDENTRIESCOMMITINDEX = 5

APPENDENTRIESRESPONSETERM = 0
APPENDENTRIESRESPONSEFROM = 1
APPENDENTRIESRESPONSESUCCESS = 2
APPENDENTRIESRESPONSEMATCHINDEX = 3


parameter_names = [['Term', 'From', 'LastLogTerm', 'LastLogIndex'], \
                   ['Term', 'From', 'Granted'], \
                   ['Term', 'From', 'PrevLogIndex', 'PrevLogTerm', 'Entry', 'CommitIndex'], \
                   ['Term', 'From', 'Suceess', 'MatchIndex']]

possible_equivalence_REQUESTVOTEREQ_REQUESTVOTEREQ = set([(REQUESTVOTEREQTERM, REQUESTVOTEREQTERM), (REQUESTVOTEREQFROM, REQUESTVOTEREQFROM), (REQUESTVOTELASTLOGTERM, REQUESTVOTELASTLOGTERM), (REQUESTVOTELASTLOGINDEX, REQUESTVOTELASTLOGINDEX)])
possible_equivalence_REQUESTVOTEREQ_REQUESTVOTERESP = set([(REQUESTVOTEREQTERM, REQUESTVOTERESPONSETERM), (REQUESTVOTEREQFROM, REQUESTVOTERESPONSEFROM)])
possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESREQ = set([(REQUESTVOTEREQTERM, APPENDENTRIESREQTERM), (REQUESTVOTEREQFROM, APPENDENTRIESREQFROM)])
possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESRESP = set([(REQUESTVOTEREQTERM, APPENDENTRIESRESPONSETERM), (REQUESTVOTEREQFROM, APPENDENTRIESRESPONSEFROM)])

possible_equivalence_REQUESTVOTERESP_REQUESTVOTERESP = set([(REQUESTVOTERESPONSETERM, REQUESTVOTERESPONSETERM), (REQUESTVOTERESPONSEFROM, REQUESTVOTERESPONSEFROM), (REQUESTVOTERESPONSEGRANTED, REQUESTVOTERESPONSEGRANTED)])
possible_equivalence_REQUESTVOTERESP_APPENDENTRIESREQ = set([(REQUESTVOTERESPONSETERM, APPENDENTRIESREQTERM), (REQUESTVOTERESPONSEFROM, APPENDENTRIESREQFROM)])
possible_equivalence_REQUESTVOTERESP_APPENDENTRIESRESP = set([(REQUESTVOTERESPONSETERM, APPENDENTRIESRESPONSETERM), (REQUESTVOTERESPONSEFROM, APPENDENTRIESRESPONSEFROM)])

possible_equivalence_APPENDENTRIESREQ_APPENDENTRIESREQ = set([(APPENDENTRIESREQTERM, APPENDENTRIESREQTERM), (APPENDENTRIESREQFROM, APPENDENTRIESREQFROM), (APPENDENTRIESREQPREVLOGINDEX, APPENDENTRIESREQPREVLOGINDEX), (APPENDENTRIESREQPREVLOGTERM, APPENDENTRIESREQPREVLOGTERM), (APPENDENTRIESREQENTRY, APPENDENTRIESREQENTRY), (APPENDENTRIESCOMMITINDEX, APPENDENTRIESCOMMITINDEX)])
possible_equivalence_APPENDENTRIESREQ_APPENDENTRIESRESP = set([(APPENDENTRIESREQTERM, APPENDENTRIESRESPONSETERM), (APPENDENTRIESREQFROM, APPENDENTRIESRESPONSEFROM)])

possible_equivalence_APPENDENTRIESRESP_APPENDENTRIESRESP = set([(APPENDENTRIESRESPONSETERM, APPENDENTRIESRESPONSETERM), (APPENDENTRIESRESPONSEFROM, APPENDENTRIESRESPONSEFROM), (APPENDENTRIESRESPONSESUCCESS, APPENDENTRIESRESPONSESUCCESS), (APPENDENTRIESRESPONSEMATCHINDEX, APPENDENTRIESRESPONSEMATCHINDEX)])

possible_equivalence = [[possible_equivalence_REQUESTVOTEREQ_REQUESTVOTEREQ, possible_equivalence_REQUESTVOTEREQ_REQUESTVOTERESP, possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESREQ,\
                        possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESRESP], \
                        [possible_equivalence_REQUESTVOTEREQ_REQUESTVOTERESP, possible_equivalence_REQUESTVOTERESP_REQUESTVOTERESP, possible_equivalence_REQUESTVOTERESP_APPENDENTRIESREQ, possible_equivalence_REQUESTVOTERESP_APPENDENTRIESRESP], \
                        [possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESREQ, possible_equivalence_REQUESTVOTERESP_APPENDENTRIESREQ, possible_equivalence_APPENDENTRIESREQ_APPENDENTRIESREQ, possible_equivalence_APPENDENTRIESREQ_APPENDENTRIESRESP], \
                        [possible_equivalence_REQUESTVOTEREQ_APPENDENTRIESRESP, possible_equivalence_REQUESTVOTERESP_APPENDENTRIESRESP, possible_equivalence_APPENDENTRIESREQ_APPENDENTRIESRESP, possible_equivalence_APPENDENTRIESRESP_APPENDENTRIESRESP]]

EQUAL = 0
UNEQUAL = 1

for i in range(TOTAL_MESSAGE_TYPE):
    for j in range(i, TOTAL_MESSAGE_TYPE):
        minimum_equal_set = []
        m1_message = m1[i]
        m2_message = m2[j]
        handler1 = handlers[i]
        handler2 = handlers[j]
        print("############# the mininum equivalence conditions for " + message_names[i] + " and " + message_names[j] + " are:\n")

        def rm_a_cond(conditions, cond):
            new_conditions = set(conditions)
            new_conditions.remove(cond)
            return new_conditions

        def generate_all_possible_conditions(conditions):
            if len(conditions) == 0:
                return set()
            if len(conditions) == 1:
                for para1, para2 in iter(conditions):
                    return [set([(para1, para2, EQUAL)]), set([(para1, para2, UNEQUAL)]), set()]

            for cond in iter(conditions):
                tmp_result_set = generate_all_possible_conditions(rm_a_cond(conditions, cond))
                para1, para2 = cond
                result_set = tmp_result_set.copy()
                for conds in iter(tmp_result_set):
                    c1 = set(conds)
                    c1.add((para1, para2, EQUAL))
                    result_set.append(c1)
                    c2 = set(conds)
                    c2.add((para1, para2, UNEQUAL))
                    result_set.append(c2)
                return result_set
        
        def check_commutative_under_conditions(m1_type, m2_type, conds):
            s = Solver()
            for para1, para2, relation in iter(conds):
                if relation == EQUAL:
                    s.add(parameters[m1_type][para1](m1_message) == parameters[m2_type][para2](m2_message))
                else:
                    s.add(parameters[m1_type][para1](m1_message) != parameters[m2_type][para2](m2_message))
            s.add(stateEqual(handler2(handler1(states, m1_message), m2_message), handler1(handler2(states, m2_message), m1_message)) == False)
            return s.check() == unsat
        
        print("generating conditions: ...\n")
        all_possible_conditions = generate_all_possible_conditions(possible_equivalence[i][j])

        print("checking conditions: ...\n")
        tmp_sets = []
        for conds in iter(all_possible_conditions):
            if check_commutative_under_conditions(i, j, conds):
                tmp_sets.append(conds)
        
        print("deduplicate: ...\n")
        def subsetExists(array, element):
            for ele in iter(array):
                if ele.issubset(element) and ele != element:
                    # print("the element is: " + str(element) + " the ele is: " + str(ele))
                    return True
            return False

        for ele in iter(tmp_sets):
            if not subsetExists(tmp_sets, ele):
                minimum_equal_set.append(ele)

        
        
        case_seq = 1
        for conditions in iter(minimum_equal_set):
            if case_seq == 1:
                print("case " + str(case_seq) + ":\n")
            else:
                print("or case " + str(case_seq) + ":\n")
            case_seq = case_seq + 1
            if conditions == set():
                print("no condition required\n")
                break
            for para1, para2, relation in iter(conditions):
                name1 = parameter_names[i][para1]
                name2 = parameter_names[j][para2]
                if relation == EQUAL:
                    print(name1 + ", " + name2 + " EQUAL \n")
                else:
                    print(name1 + ", " + name2 + " UNEQUAL \n")


        # UNSATISFIED = -1
        # def rm_a_cond(conditions, cond):
        #     new_conditions = set(conditions)
        #     new_conditions.remove(cond)
        #     return new_conditions

        # def check_commutative_under_conditions(m1_type, m2_type, conditions):
        #     s = Solver()
        #     for para1, para2 in iter(conditions):
        #         s.add(parameters[m1_type][para1] == parameters[m2_type][para2])
        #         s.add(parameters[m1_type][para1] == parameters[m2_type][para2])
        #     s.add(stateEqual(handler2(handler1(states, m1_message), m2_message), handler1(handler2(states, m2_message), m1_message)) == False)
        #     return s.check() == unsat

        # def add_to_set(conditions):
        #     if conditions == UNSATISFIED:
        #         return
        #     else:
        #         minimum_equal_set.add(conditions)

        # def find_mininum_set_recursively(initial_conditions):
        #     # if not check_commutative_under_conditions(initial_conditions):
        #     #     return None
        #     if initial_conditions == None:
        #         return None
        #     keep_all = True
        #     for cond in iter(initial_conditions):
        #         tmp_set = rm_a_cond(initial_conditions, cond)
        #         if check_commutative_under_conditions(i, j, tmp_set):
        #             keep_all = False
        #             add_to_set(find_mininum_set_recursively(tmp_set))
        #     if keep_all:
        #         if check_commutative_under_conditions(i, j, initial_conditions):
        #             return initial_conditions
        #         else:
        #             return UNSATISFIED
        
        # minimum_equal_set.add(find_mininum_set_recursively(possible_equivalence[i][j]))



# def test1():
#     # test1: whether two requestVoteReq are commutative
#     sl = Solver()
#     m1_t = mk_requestVoteReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_last_log_term'), Int('m1_last_log_index'))
#     m2_t = mk_requestVoteReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_last_log_term'), Int('m2_last_log_index'))

#     sl.add(RequestVoteReqMsgLastLogIndex(m1_t) != RequestVoteReqMsgLastLogIndex(m2_t))
#     sl.add(stateEqual(requestVoteReqHandler(requestVoteReqHandler(states, m1_t), m2_t), requestVoteReqHandler(requestVoteReqHandler(states, m2_t), m1_t)) == False)
#     print(sl.check())

# test1()
# def test2():
#     # test2: whether two requestVoteResponse are commutative
#     m1 = mk_requestVoteResponseMsg(Int('m1_term'), Int('m1_from'), Bool('m1_granted'))
#     m2 = mk_requestVoteResponseMsg(Int('m2_term'), Int('m2_from'), Bool('m2_granted'))
#     s.add(z3.If(RequestVoteResponseMsgGranted(m1), RequestVoteResponseMsgTerm(m1) <= Term(states), True))
#     s.add(z3.If(RequestVoteResponseMsgGranted(m2), RequestVoteResponseMsgTerm(m2) <= Term(states), True))
#     s.add(stateEqual(requestVoteResponseHandler(requestVoteResponseHandler(states, m1), m2), requestVoteResponseHandler(requestVoteResponseHandler(states, m2), m1)) == False)

# def test3():
#     # test3: whether two appendEntriesReq are commutative
#     m1 = mk_appendEntriesReqMsg(Int('m1_term'), Int('m1_from'), Int('m1_prelogindex'), Int('m1_prelogterm'), Const('entry1', log_entry), Int('m1_commitindex'))
#     m2 = mk_appendEntriesReqMsg(Int('m2_term'), Int('m2_from'), Int('m2_prelogindex'), Int('m2_prelogterm'), Const('entry2', log_entry), Int('m2_commitindex'))
#     s.add(AppendEntriesReqMsgPrevLogIndex(m1) == AppendEntriesReqMsgPrevLogIndex(m2))
#     s.add(AppendEntriesReqMsgPrevLogTerm(m1) == AppendEntriesReqMsgPrevLogTerm(m2))
#     s.add(AppendEntriesReqMsgEntry(m1) == AppendEntriesReqMsgEntry(m2))
#     s.add(AppendEntriesReqMsgTerm(m1) == AppendEntriesReqMsgTerm(m2))
#     s.add(stateEqual(appendEntryReqHandler(appendEntryReqHandler(states, m1), m2), appendEntryReqHandler(appendEntryReqHandler(states, m2), m1)) == False)

# def test4():
#     h1 = appendEntryResponseHandler
#     h2 = appendEntryResponseHandler
#     m1 = mk_appendEntriesResponseMsg(Int('m1_term'), Int('m1_from'), Bool('m1_success'), Int('m1_matchindex'))
#     m2 = mk_appendEntriesResponseMsg(Int('m2_term'), Int('m2_from'), Bool('m2_success'), Int('m2_matchindex'))
# # test1()
# # test2()
# # test3()
# test4()
# s.add(stateEqual(h2(h1(states, m1), m2), h1(h2(states, m2), m1)) == False)
# result = s.check()
# print(result)
# if result == sat:
#     print(s.model())