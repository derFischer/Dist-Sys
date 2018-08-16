package raftkv

import "labrpc"
import "crypto/rand"
import (
	"math/big"
	"sync"
	"time"
)

const (
	WAITTIME = 100
	TRITIME  = 3
)

type Clerk struct {
	mu      sync.Mutex
	servers []*labrpc.ClientEnd
	id      int64
	reqId   int
	revId   int
	// You will have to modify this struct.
	leader int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.id = nrand()
	ck.reqId = 0
	ck.revId = 0
	ck.leader = 0
	//fmt.Printf("client %d has been made, connect to server %+v\n", ck.id, ck.servers)
	// You'll have to add code here.
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//

func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.
	var args GetArgs
	args.Key = key
	args.ClientId = ck.id
	ck.mu.Lock()
	args.ReqId = ck.reqId
	ck.reqId += 1
	ck.mu.Unlock()

	for {
		if ck.reqId == ck.revId+1 {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	for {
		var reply GetReply
		var ok bool
		tries := 0
		for {
			tries++
			//fmt.Printf("client %d wants to send GET RPC %d to server %d content: %+v tries: %d\n", ck.id, ck.reqId, ck.leader, args, tries)
			go func() {
				ok = ck.servers[ck.leader].Call("KVServer.Get", &args, &reply)
			}()
			time.Sleep(WAITTIME * time.Millisecond)
			if ok || tries >= TRITIME {
				//fmt.Printf("content: %+v tries: %d\n", args, tries)
				break
			}
		}
		//fmt.Printf("GET RPC %+v received reply %+v\n", args, reply)
		if ok && !reply.WrongLeader {
			if reply.Err == OK {
				ck.revId++
				return reply.Value
			} else if reply.Err == ErrNoKey {
				ck.revId++
				return ""
			}
		} else {
			for i := 0; i < len(ck.servers); i++ {
				var reply GetReply
				var ok bool
				tries := 0
				for {
					tries++
					//fmt.Printf("client %d wants to send GET RPC %d to server %d content: %+v tries: %d\n", ck.id, ck.reqId, ck.leader, args, tries)
					go func() {
						ok = ck.servers[i].Call("KVServer.Get", &args, &reply)
					}()
					time.Sleep(WAITTIME * time.Millisecond)
					if ok || tries >= TRITIME {
						//fmt.Printf("content: %+v tries: %d\n", args, tries)
						break
					}
				}
				//fmt.Printf("GET RPC %+v received reply %+v\n", args, reply)
				if ok && !reply.WrongLeader {
					ck.leader = i
					if reply.Err == OK {
						ck.revId++
						return reply.Value
					} else if reply.Err == ErrNoKey {
						ck.revId++
						return ""
					}
				}
			}
		}
	}
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//

func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	var args PutAppendArgs
	args.Key = key
	args.Value = value
	args.Op = op
	args.ClientId = ck.id
	ck.mu.Lock()
	args.ReqId = ck.reqId
	ck.reqId += 1
	ck.mu.Unlock()

	for {
		if ck.reqId == ck.revId+1 {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	//first try to send the RPC to the leader which was recorded locally
	for {
		var reply PutAppendReply
		var ok bool
		tries := 0
		for {
			tries++
			//fmt.Printf("client %d wants to send APPEND RPC %d to server %d content: %+v tries: %d\n", ck.id, ck.reqId, ck.leader, args, tries)
			go func() {
				ok = ck.servers[ck.leader].Call("KVServer.PutAppend", &args, &reply)
			}()
			time.Sleep(WAITTIME * time.Millisecond)
			if ok || tries >= TRITIME {
				//fmt.Printf("content: %+v tries: %d\n", args, tries)
				break
			}
		}
		//fmt.Printf("APPEND RPC %+v received reply %+v\n", args, reply)

		if ok && !reply.WrongLeader {
			if reply.Err == OK {
				ck.revId++
				return
			} else {
				ck.revId++
				return
			}
		} else {
			for i := 0; i < len(ck.servers); i++ {
				//fmt.Printf("client %d wants to send APPEND RPC %d to server %d content: %+v\n", ck.id, ck.reqId, i, args)
				var reply PutAppendReply
				var ok bool
				tries := 0
				for {
					tries++
					//fmt.Printf("client %d wants to send APPEND RPC %d to server %d content: %+v tries: %d\n", ck.id, ck.reqId, ck.leader, args, tries)
					go func() {
						ok = ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
					}()
					time.Sleep(WAITTIME * time.Millisecond)
					if ok || tries >= TRITIME {
						//fmt.Printf("content: %+v tries: %d\n", args, tries)
						break
					}
				}
				//fmt.Printf("APPEND RPC %+v received reply %+v\n", args, reply)
				if ok && !reply.WrongLeader {
					ck.leader = i
					if reply.Err == OK {
						ck.revId++
						return
					} else {
						ck.revId++
						return
					}
				}
			}
		}
	}
	//if the recorded leader has been replaced, retry RPC to other kvservers
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "PUT")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "APPEND")
}
