# Overview
This report tries to figure out the following questions through surveying several open-source distributed systems, including [ETCD](https://github.com/etcd-io/etcd), [Redis-Raft](https://github.com/RedisLabs/redisraft), [MongoDB](https://github.com/mongodb/mongo) and [Scylla](https://github.com/scylladb/scylla):
- A summary of some inconsistency bugs (especially, related to consensus protocol) in the real-world open-source distributed system.
  - Inconsistent behavior and the events required.
  - Why the bug/How to fix.
- Message mapping from implementation to specification.
- How to catch the inconsistency through simulation?

These kinds of inconsistency bugs are due to that the implementation doesn't follow the specification of the correct consensus algorithm, and sometimes such differences are intentional for optimizing performance.

Therefore, it becomes more difficult to detect the bugs in different systems, since they may apply different optimization mechanisms based on the original consensus algorithms. To verify the correctness of each optimization would be cumbersome and cannot scale (hard to import other optimizations later).

We aim to find out a **general way with one-time effort** to detect these bugs.

# Introduction to the systems
## ETCD
The consensus protocol of ETCD is [Raft](https://raft.github.io/raft.pdf), except that it applies some optimization mechanisms, including PreVote and ReConfig. Since these kinds of optimization are not specified in the paper and thus haven't been proved before, some of them would be **incorrect**. See **Inconsistency bugs - ETCD** section for further details.

## RedisRaft
This is a Redis module which implements [Raft](https://raft.github.io/raft.pdf) and provides strongly-consistent clusters of Redis servers.

## MongoDB
[The consensus protocol of MongoDB](https://www.usenix.org/conference/nsdi21/presentation/zhou) derives from Raft. Instead of the push-based replication in Raft, MongoDB utilized **pull-based** replication.

**Minor question here:** Map this protocol back to Raft?
**Another question:** Can we map arbitrary consensus protocol to one specification?

**Advantages**
- Flexible data transmission.
- Reuse primary-backup mechanism.
- Support many replicas.

**Design Overview**
- Adopt Raft's election rules.
- Reuse pull-based data replication (as primary-backup)
  - Replace the ***AppendEntry*** request (primary -> secondary) in Raft with a ***Pull*** request (secondary -> primary). To ensure the correctness, the secondary will reply its current term to the primary. If primary finds that the term is larger, it will step down.
- Attention: It reuses the old mechanism in MongoDB so that **a server will apply a command as soon as it appends the entry into its log**. If the entry will be overwritten later, it will roll back the updates. As soon as the op is replicated to majority, the leader will then ack the client.

<!-- <h2 id="2">Scylla</h2>
Scylla partitions the data according to the token, and in each partition, it runs a raft group seperately.

**TODO: Currently the confusing part of Scylla is that Scylla uses Raft to replicate LWT operations, but its membership protocol is based on Cassandra's gossip system, which results in several correctness problems.** -->


# Inconsistency bugs
## ETCD
### [Consistent reads are not consistent](https://github.com/etcd-io/etcd/issues/741)
**Inconsistent behavior:** A read may return a stale version.

Client sends a read request to an old leader, and the leader is partitioned from other nodes. The node will check its state, and since its state is LEADER, it believes that it maintains the most up-to-date versions, and directly returns to the client without replicating the read to MAJORITY.

**How to fix:** Replicate each read to MAJORITY instead of replying to the client directly (even though the state is LEADER).

### [Pending conf change is not handled correctly during campaign](https://github.com/etcd-io/etcd/issues/12133)
**Inconsistent behavior:** There would be two leaders in the same term.

Specifically, this bug is related to an algorithm optimization in ETCD, which is not specified in Raft.

<h4 id="1">ETCD optimization of reconfig:</h4>
In Raft, it requires that each replica should uses the latest config entry in its log as its quorum config, regardless whether the log entry is committed or not. This means that to follow the algorithm, one node needs to keep track of the latest conf entry, which is not trivial, since uncommitted entries might be overwritten.

<br />

In ETCD, instead of exactly implementing the algorithm specified by Raft, it changes the algorithm: **once there is a committed but unapplied conf change entry, a node cannot send RequestVote requests.** 

**TODO: This change hasn't been proved and the correctness still needs to be verified. Currently I think this implementation may still result in inconsistency.**

The bug here is that it fails to check one type of conf change when scanning the log.

**How to fix:** Fix the check function (add the type).

## RedisRaft
### [Total Data Loss on Failover](https://github.com/RedisLabs/redisraft/issues/14)
**Inconsistency behavior:** the data in follower nodes are completely lost even the entries are committed.

When execute the command, RedisRaft module will send back the request to Redis, trying to apply this command to database. However, the message will be sent back to RedisRaft Module (TODO: check the code? I guess this is due to that RedisRaft is an external module and it will catch all the messages send to Redis, including the message itself sends). Then, RedisRaft will raftize this message again. Since it is a follower, it will reject this request.

**How to fix:** Before send back to Redis in execute function, RedisRaft will first enter RedisModuleState. Then, in the raftize function, it will first check whether it is in RedisModuleState. If yes, it will not actually raftize this command.

### [Stale reads in normal operation](https://github.com/RedisLabs/redisraft/issues/19)
**Inconsistency behavior:** stale read.

This bug is due to that the node forgets to append a no-op command once becomes the leader. This may result in that a node whose log is not most up-to-date becomes a new leader and some committed entries would be overwritten (and the FSM will diverge). Suppose a write op is committed but the new leader knows nothing about the write. Then if the leader serves a read, it still can replicate the read to majority, and replies to the client. But since the leader hasn't applied the write before, the client will see an older version.

**How to fix:** Enforce a node to append a no-op command once becomes the leader.

### [Possible lost updates with partitions and membership changes](https://github.com/RedisLabs/redisraft/issues/17)
**Inconsistency behavior:** There would be two leaders in the same term.

Fail to identify one type of conf change (Remove Node).

**TODO: The code of conf change is also different from the specification and actually I am not sure whether it is correct. It doesn't implement joint quorum at all, but requires that there could only be at most 1 pending (unapplied) conf change (both committed or uncommitted). And each time the server appends a conf change, it will update its conf. This may affect avalability but will not affect the correctness?**


## MongoDB
### [Acknowledge writes from prior terms, which has already been rolled back.](https://jira.mongodb.org/browse/SERVER-27053)

**Inconsistency behavior:** A write in the old term has already been rolled back (discarded), but the leader acked the client that the write is successfully committed.

Once the old leader finds out that its term is out-of-date, it will step down. Given this sequence:
- A is a leader, term = 1. A receives writes R.
- C becomes candidate, term = 2. B votes for C, term = 2.
- B sync from A, send back already replicate writes R, term = 2.
- A steps down, rolls back the writes.
- Later B will sync with C and overwrites the previous write, correct.

But the bug is:
- A steps down, rolls back the writes.
- A becomes candidate again, term = 3.
- A becomes the leader, term = 3.
- A forgets to close the connection with B before steps down, and then receives the message from B that it has replicated writes R. This thread thinks both A and B replicates this write, and will ack the client.

**How to fix:** Close all the connection (both external or interal) when step down.

### [Don't sync from nodes in an older term](https://jira.mongodb.org/browse/SERVER-27149)
**Inconsistency behavior:** Lost committed entries.

In MongoDB, the follower needs to compare the OpTime (term and timestamp) to decide whether the source is more up-to-date. However, it forgets to check the term (check the timestamp only) when syncing with the source. Therefore, the follower could sync with a partitioned old leader whose log is not up-to-date but the timestamp is large (receive a lot of requests from the client).

This bug is due to that the code is based on MongoDB's previous implementation, which is based on primary-backup and uses timestamp only to decide source.

**How to fix:** Add term when check the source.

# Checking
## General algorithm of simulation checking
1. **Build a message mapping (given by the user).**
   - All the messages appear (related to) in the specification should be mapped.
   - The mapping could be **n (implementation) to 1 (specification)**.
     - E.g., both AppendEntries and HeartBeat will be mapped to AppendEntry in specification).
   - Given a mapping pair, the number of messages could be **1 (implementation) to n (specification)**.
     - E.g., the AppendEntries request in implementation is batched, and should be mapped back to $n$ AppendEntry request in specification.
2. **Run the system together with the simulation, and the simulation does the following things:**
   - For each message sent in the system:
     - Interpret it according to the mapping.
     - **Check the validity of the message.**
       - If invalid, assert an error.
   - For each message received in the system:
     - interpret it according to the mapping.
     - Execute the handler function in the simulation part.

### Check the validity of the message
**1. Whether the server is eligible to send this message.**
  - E.g., only a leader can send AppendEntry request.
  - This is specified in TLA specification.
  
**2. The timing of the message?**
  - E.g. one can only send VoteResponse msg after receiving the vote request?
  - Or the timing doesn't affect the correctness as long as 1 and 3 are satisfied?
  - Discussion 11-17-21: we assume that we can map each response back to the corresponding request.
  
**3. The content of the message.**
  - Or, whether the parameters in the message are correct?
  - Some are easy to check, e.g. term, lastLogIndex, etc...
    - These are **decided** by the current state (term, log, ...), only one value is valid.
    - Some parameters could be arbitrary value as long as it satisfies the constraints between the parameters.
      - E.g., prevLogIndex and entry (could be arbitrary value but these two should match the log).
      - Discussion 11-17-21: we assume that we have the relations between the parameters (this is specified by the user).
    - Some parameters depend on the previous messages. E.g. matchIndex in AppendEntryResponse, it cannot be inferred from the current state (log), but depends on the previous AppendEntryRequest.
      - Discussion 11-17-21: we assume that we can map each response back to the corresponding request.



## Message Mapping
### ETCD
1. **MsgProp <=> ClientRequest (1 to n mapping)**
   - to: receiver <=> i
   - from: self <=> ?
   - entries <=> v (1 to n mapping)
2. **MsgApp <=> AppendEntriesRequest (1 to n mapping)**
   - to <=> mdest
   - from <=> msource
   - term <=> mterm
   - entries <=> mentries (1 to n mapping)
   - index <=> mprevLogIndex
   - logTerm <=> prevLogTerm
   - commit <=> mcommitIndex
3. **MsgAppResp <=> AppendEntriesResponse**
   - to <=> mdest
   - from <=> msource
   - term <=> mterm
   - index <=> mmatchIndex
   - Reject <=> msuccess
   - RejectHint <=> ?
   - LogTerm <=> ?
4. **MsgVote <=> RequestVote**
   - to <=> mdest
   - from <=> msource
   - term <=> mterm
   - index <=> mlastLogIndex
   - logTerm <=> mlastLogTerm
5. **MsgVoteResp <=> RequestVoteResponse**
   - to <=> mdest
   - from <=> msource
   - term <=> mterm
   - Reject <=> mvoteGranted
6. MsgHeartBeat <=> AppendEntriesRequest
   - to <=> mdest
   - from <=> msource
   - term <=> term
   - commit <=> mcommitIndex
   - nil <=> mentries
   - nil <=> index
   - nil <=> logTerm

### RedisRaft
1. **msg_entry_t <=> ClientRequest (1 to n mapping)**
   - raft\_entry\_t <=> v (1 to n mapping)
2. **msg\_appendentries\_t <=> AppendEntriesRequest (1 to n mapping)**
   - leader\_id <=> msource
   - term <=> mterm
   - entries <=> mentries (1 to n mapping)
   - prev\_log\_idx <=> mprevLogIndex
   - prev\_log\_term <=> prevLogTerm
   - leader\_commit <=> mcommitIndex
3. **msg\_append\_entries\_response\_t <=> AppendEntriesResponse**
   - term <=> mterm
   - current\_idx <=> mmatchIndex
   - success <=> msuccess
4. **msg\_requestvote\_t <=> RequestVote**
   - prevote <=> **false**
   - candidate\_id <=> msource
   - term <=> mterm
   - last\_log\_idx <=> mlastLogIndex
   - last\_log\_term <=> mlastLogTerm
5. **msg\_requestvote\_response\_t <=> RequestVoteResponse**
   - prevote <=> **false**
   - to <=> mdest
   - request\_term <=> ?
   - term <=> mterm
   - vote\_granted <=> mvoteGranted

### MongoDB ([TLA](https://github.com/mongodb/mongo/blob/master/src/mongo/tla_plus/RaftMongo/RaftMongo.tla)) to Raft
| Function | Raft | MongoDB |
| -- | -- | -- |
| Sync log (source -> receiver) | AppendEntries **Request** | PullEntries **Response** |
| Sync log (receiver -> source) | AppendEntries **Response** | UpdatePosition **Request** |

- Sync log (source -> receiver)
  - term <=> ?
  - mentries <=> entries[2:]
  - prev\_log\_idx <=>? entries[0].timestamp
  - prev\_log\_term <=> entries[0].term
  - leader\_commit <=> commitPoint

- Sync log (receiver -> source)
  - mterm <=> term
  - current\_idx <=>? lastPosition[sender].timestamp
  - success <=> ?

Algorithm:
Raft:
- Check the term, prev\_log\_idx, and prev\_log\_term. If satisfies, append the entries.

MongoDB:
- Check the last entry of entries and local log, to determine whether the source is more up-to-date.


Difference:
- timestamp v.s. index: for the similar purpose, but need to translate timestamp back to index (not trivial).
  - E.g., given a timestamp, a follower can compute **where the index should be** based on the timestamp. But given an index, the follower cannot infer the correct index (it can just check whether the entry at this index is correct).
- MongoDB compares the log to check whether the source is more up-to-date, but Raft only compares the term:
   -  In Raft, only leader can send AppendEntries Request, and it has batched phase 1 in vote phase. Phase 2 ballot == term == term of the last entry in the log.
   -  In MongoDB, a server can pull request from any other server. **The key difference is that mapping to Raft, a follower can also send AppendEntries Request even without phase 1.** Therefore, term cannot specify whether this source is more up-to-date, and it needs to use the similar way as vote to check the log (kind of vote + append together, if vote failed, the append will be rollbacked, the difference is that it will not increment its term, and therefore it have the same term as the leader).


## Bug Detection
### [Stale reads in normal operation](https://github.com/RedisLabs/redisraft/issues/19)
The key bug is not that the server must propose a no-op once becomes the leader, but the leader can only commit until to the entries whose term == current term. To propose a no-op just aims for fast commit the entries.

### [Consistent reads are not consistent](https://github.com/etcd-io/etcd/issues/741)