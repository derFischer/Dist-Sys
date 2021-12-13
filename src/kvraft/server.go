package raftkv

import (
	"Common"
	"bytes"
	"labgob"
	"labrpc"
	"log"
	"raft"
	"simulation"
	"sync"
)

const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

const (
	APPEND = "APPEND"
	PUT    = "PUT"
	GET    = "GET"

	OFFLINETIME = 50
)


type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	//if the server has been stopped
	stop bool

	//record
	record  map[string]string
	history map[int64]Common.Recent

	maxraftstate int // snapshot if log grows this big
	sl *simulation.Simulation

	// Your definitions here.
}

func (kv *KVServer) AppendEntryToLog(entry Common.Op) bool {

	_, _, isLeader := kv.rf.Start(entry)
	/*select {
	case <-time.After(OFFLINETIME * time.Millisecond):
		return isLeader
	}*/
	return isLeader
}

func (kv *KVServer) isUpToDate(ClientId int64, ReqId int) bool {
	value, exist := kv.history[ClientId]
	if exist {
		return ReqId > value.ReqId
	}
	return true
}

func (kv *KVServer) Get(args *Common.GetArgs, reply *Common.GetReply) {
	//fmt.Printf("server %d receive a GET RPC %+v\n", kv.me, args)
	defer kv.sl.CheckReplyMessage("KVServer.Get", args, reply)
	if kv.sl != nil {
		kv.sl.HandleRequestMessage("KVServer.Get", args)
	}

	if !kv.isUpToDate(args.ClientId, args.ReqId) {
		tmp := kv.history[args.ClientId]
		if args.ReqId == tmp.ReqId {
			reply.Err = tmp.Err
			reply.Value = tmp.Result
			reply.WrongLeader = false
			return
		}
	}

	//var ApplyCh = make(chan recent, 1)
	//entry := Op{Op: GET, Key: args.Key, ClientId: args.ClientId, ReqId: args.ReqId, ApplyChan: ApplyCh}

	//isLeader := kv.AppendEntryToLog(entry)
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.WrongLeader = true
		return
	} else {
		reply.WrongLeader = false
		v, ok := kv.record[args.Key]
		if ok {
			reply.Value = v
			reply.Err = Common.OK
		} else {
			reply.Value = ""
			reply.Err = Common.ErrNoKey
		}

		//for {
		//	select {
		//	case tmp := <-ApplyCh:
		//		reply.Err = tmp.Err
		//		if reply.Err == OK {
		//			reply.Value = tmp.Result
		//		}
		//		return
		//	}
		//}
	}
}

func (kv *KVServer) apply(entry Common.Op, index int) {

	if !kv.isUpToDate(entry.ClientId, entry.ReqId) {
		var reply Common.Recent
		if entry.Op == GET {
			tmp := kv.history[entry.ClientId]
			if entry.ReqId == tmp.ReqId {
				reply.Err = tmp.Err
				reply.Result = tmp.Result
				return
			}
		} else {
			reply.Err = Common.OK
		}
		if entry.ApplyChan != nil {
			entry.ApplyChan <- kv.history[entry.ClientId]
		}
		return
	}

	defer func() {
		//fmt.Printf("server %d finish applying the command %+v refresh the highest req id %d of client %d\n", kv.me, entry, kv.history[entry.ClientId].ReqId, entry.ClientId)
	}()

	switch entry.Op {
	case APPEND:
		_, ok := kv.record[entry.Key]
		if !ok {
			kv.record[entry.Key] = entry.Value
		} else {
			kv.record[entry.Key] += entry.Value
		}
		kv.history[entry.ClientId] = Common.Recent{ReqId: entry.ReqId, Err: Common.OK}
	case PUT:
		kv.record[entry.Key] = entry.Value
		kv.history[entry.ClientId] = Common.Recent{ReqId: entry.ReqId, Err: Common.OK}
	case GET:
		v, ok := kv.record[entry.Key]
		if ok {
			kv.history[entry.ClientId] = Common.Recent{ReqId: entry.ReqId, Result: v, Err: Common.OK}
		} else {
			kv.history[entry.ClientId] = Common.Recent{ReqId: entry.ReqId, Err: Common.ErrNoKey}
		}
	}
	if entry.ApplyChan != nil {
		entry.ApplyChan <- kv.history[entry.ClientId]
	}
}

func (kv *KVServer) PutAppend(args *Common.PutAppendArgs, reply *Common.PutAppendReply) {
	// Your code here.
	//fmt.Printf("server %d receive a PUTAPPEND RPC %+v local highest req id %d \n", kv.me, args, kv.history[args.ClientId].ReqId)
	defer kv.sl.CheckReplyMessage("KVServer.PutAppend", args, reply)
	if kv.sl != nil {
		kv.sl.HandleRequestMessage("KVServer.PutAppend", args)
	}

	if !kv.isUpToDate(args.ClientId, args.ReqId) {
		reply.WrongLeader = false
		reply.Err = Common.OK
		return
	}

	ApplyCh := make(chan Common.Recent, 1)
	entry := Common.Op{Op: args.Op, Key: args.Key, Value: args.Value, ClientId: args.ClientId, ReqId: args.ReqId, ApplyChan: ApplyCh}

	isLeader := kv.AppendEntryToLog(entry)
	if !isLeader {
		reply.WrongLeader = true
		return
	} else {
		reply.WrongLeader = false
		for {
			select {
			case tmp := <-ApplyCh:
				reply.Err = tmp.Err
				return
			}
		}
	}

}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *KVServer) Kill() {
	//fmt.Printf("server %d crash\n", kv.me)
	kv.mu.Lock()
	kv.rf.Kill()
	kv.stop = true
	kv.mu.Unlock()
	// Your code here, if desired.
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServerWithSL(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, sl *simulation.Simulation) *KVServer {
	labgob.Register(Common.Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.MakeWithSL(servers, me, persister, kv.applyCh, sl)
	kv.stop = false
	kv.record = map[string]string{}
	kv.history = map[int64]Common.Recent{}
	kv.sl = sl

	// You may need initialization code here.
	go func() {
		for {
			if kv.stop {
				break
			}
			select {
			case tmp := <-kv.applyCh:
				kv.mu.Lock()
				if tmp.UseSnapshot {
					r := bytes.NewBuffer(tmp.Snapshot)
					d := labgob.NewDecoder(r)

					kv.record = make(map[string]string)
					kv.history = make(map[int64]Common.Recent)

					d.Decode(&kv.record)
					d.Decode(&kv.history)
				} else if tmp.Command != nil {
					kv.apply(tmp.Command.(Common.Op), tmp.CommandIndex)
					if maxraftstate != -1 && kv.rf.GetRaftStateSize() >= (maxraftstate/2) {
						//fmt.Printf("server %d wants to trim the raft state %d\n", kv.me, kv.rf.GetRaftStateSize())
						w := new(bytes.Buffer)
						e := labgob.NewEncoder(w)
						e.Encode(kv.record)
						e.Encode(kv.history)
						data := w.Bytes()
						go kv.rf.StartSnapshot(data, tmp)
					}
				}
				kv.mu.Unlock()
			}
		}
	}()

	return kv
}

func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Common.Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.stop = false
	kv.record = map[string]string{}
	kv.history = map[int64]Common.Recent{}

	// You may need initialization code here.
	go func() {
		for {
			if kv.stop {
				break
			}
			select {
			case tmp := <-kv.applyCh:
				kv.mu.Lock()
				if tmp.UseSnapshot {
					r := bytes.NewBuffer(tmp.Snapshot)
					d := labgob.NewDecoder(r)

					kv.record = make(map[string]string)
					kv.history = make(map[int64]Common.Recent)

					d.Decode(&kv.record)
					d.Decode(&kv.history)
				} else if tmp.Command != nil {
					kv.apply(tmp.Command.(Common.Op), tmp.CommandIndex)
					if maxraftstate != -1 && kv.rf.GetRaftStateSize() >= (maxraftstate/2) {
						//fmt.Printf("server %d wants to trim the raft state %d\n", kv.me, kv.rf.GetRaftStateSize())
						w := new(bytes.Buffer)
						e := labgob.NewEncoder(w)
						e.Encode(kv.record)
						e.Encode(kv.history)
						data := w.Bytes()
						go kv.rf.StartSnapshot(data, tmp)
					}
				}
				kv.mu.Unlock()
			}
		}
	}()

	return kv
}
