TLA is quite **a concrete version of implementation**, but may not be used as simulation directly.

States $\stackrel{decide}{\longleftrightarrow}$ Messages

# States
|   TLA  | Simulation  |
|  :-:  | :-: |
| log  | $\checkmark$* |
| currentTerm  |  $\checkmark$ |
| state | $\checkmark$* |
| votedFor | $\checkmark$ |
| commitIndex | $\checkmark$* |
| votesResponded |  | 
| votesGranted | $\checkmark$ | 
| nextIndex | |
| matchIndex| $\checkmark$* |
*:The updated ways are different from that in the simulation.


**log**: The AppendEntriesHandler is different.

**state**: The simulation allows a leader => follower => candidate without receiving any external message. While the TLA doesn't allow leader => follower.

**commitIndex**: The AppendEntriesHandler, AppendEntriesResponseHandler are different.

**matchIndex**: The AppendEntriesResponseHandler is different.




# Messages
|   TLA  | Simulation  |
|  :-:  | :-:  |
| Client Request  | Client Request *(But support interpret Get, Put... for KV operations)* |
| Append Entries Request *(up to 1 entry)*  | Append Entries/Heartbeat Request *(support multiple entries)* |
| Append Entries Request Resp *(up to 1 entry)*  | Append Entries/Heartbeat Request Resp *(support multiple entries)* |
| Request Vote Request  | Request Vote Request |
| Request Vote Response  | Request Vote Response |
| -  | Read Index Request |
| -  | Read Index Response |

# Simulation
## Minimum States and Messages Set
The goal is to ensure the **core states (which decide the KV store)** are correct. The messages will affect the states.
Considering the requests except RO, the **core states** in a server are: **log, commitIndex**. Other states in the simulation like **currentTerm, state, votedFor, votesGranted, matchIndex** will affect **log** or **commitIndex**, and thus we should check them in the simulation.

Similarly, all the messages that affect the above states should be checked, e.g. AppendEntries, RequestVote. **Not all the parameters in these messages should be checked.** Although the **nextIndex** is attached in **AppendEntriesRequest** msg, but it doesn't affect the states.

We do not need to simulate other messages which don't affect the above states. E.g., **PreVote** messages in ETCD doesn't change any of the above states.

## How to Check? Valid States/Messages Set
Suppose a raft server is a state machine, given a sequence of input messages, its valid local states could be different (a valid set).

|   Sequence  | Pre Log State | Post Valid Log state  |
|  :-:  | :-:  | :-: |
| Client Req Q1, Q2, Q3 | \|\| | \|Q1\|Q2\|Q3\|,\|Q2\|Q1\|Q3\|,\|Q1\|Q3\|Q2\|,... |
| Client Req Q1, Q2, Q3, AppendEntryReq(PreLogIndex:0, Q1) | \|\| | \|Q1\|Q2\|Q3\|, \|Q1\|Q3\|Q2\| |

|   Sequence  | Pre CommitIndex State | Post CommitIndex state  |
|  :-:  | :-:  | :-: |
| AppendEntryResp(matchIndex: 5)  | 0 | <= 0 |
| AppendEntryResp(matchIndex: 5), AppendEntryResp(matchIndex: 5) | 0 | <= 5 |

ETCD allows the following state change: A leader can step down to a follower without receiving any external messages. Such transition is valid.
|   Sequence  | Pre State State | Post State state  |
|  :-:  | :-:  | :-: |
| nil  | Leader | Leader, Follower |

The simulation should be **generalizable enough to allow for checking more valid states/transitions/messages**. E.g. the commitIndex should be **as large as possible**.