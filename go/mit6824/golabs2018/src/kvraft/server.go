package raftkv

import (
	"labgob"
	"labrpc"
	"log"
	"raft"
	"sync"
)

const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

const mydebugEnabled = false
const mylogEnabled = true

func mydebug(a ...interface{}) (n int, err error) {
	if mydebugEnabled {
		n, err = fmt.Println(a...)
	}
	return
}
func mylog(a ...interface{}) (n int, err error) {
	if mylogEnabled {
		n, err = fmt.Println(a...)
	}
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.

	Op       string // "Put" or "Append" or "Get"
	ClientId int64  // 客户端编号
	OpNo     int64  // 唯一操作编号
	Key      string
	Value    string
}

const OP_STATE_STARTED = 1 // 已经调用了raft的Start，但是没有收到响应
const OP_STATE_APPLIED = 2 // 收到了raft的apply消息

type OpResult struct {
	ClientId int64
	OpNo    int64 // 唯一操作编号
	OpState int   // 操作状态

	Term int
	CommandIndex int

	NofityCh chan int // 通知其他等待的

	// 上一次已经完成时的返回数据，包括PutAppendReply和GetReply的并集
	WrongLeader bool
	Err         Err
	Value       string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	killCh chan int

	kvMap map[string]string // 保存成功通过Raft的数据

	// clientId -> 操作结果
	// 由于一个客户端一次只会发起一次请求，但是可能重复发送
	lastClientOpMap map[int64]*OpResult
	// CommandIndex -> 请求
	lastCommandMap map[int]*OpResult
}

func (kv *KVServer) Lock() {
	kv.mu.Lock()
}
func (kv *KVServer) Unlock() {
	kv.mu.Unlock()
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.

	term, isLeader = kv.rf.GetState()

	if !isLeader {
		reply.WrongLeader = true
		reply.Err = nil
		reply.Value = nil
	} else {
		if kv.tryGet(args, reply) {
			
		}
	}
}

// 返回true说明需要等待，false说明不需要等待
func (kv *KVServer) tryGet(args *GetArgs, reply *GetReply) bool {
	kv.Lock()
	defer kv.Unlock()

	opResult, ok := kv.lastClientOpMap[args.ClientId]

	if ok && opResult.OpNo == args.OpNo {
		if opResult.OpState == STATE_ARRLIED {
			mylog("发现重复Get[", args.OpNo, "]请求，之前已经应用，直接返回")
			reply.WrongLeader = opResult.WrongLeader
			reply.Err = opResult.Err
			reply.Value = opResult.Value
			return false
		} else {
			mylog("发现重复Get[", args.OpNo, "]请求，之前还没有应用，需要等待")
			return true
		}
	} else {
		op := Op{
			Op:       "Get",
			ClientId: args.ClientId,
			OpNo:     args.OpNo,
			Key:      args.Key,
			Value:    nil,
		}
		index, term, isLeader := kv.rf.Start(op)

		if !isLeader {
			reply.WrongLeader = true
			reply.Err = nil
			reply.Value = nil
			return false
		} else {
			opRes = &OpResult{
				ClientId : args.ClientId,
				OpNo:     args.OpNo,
				OpState:  OP_STATE_STARTED,
				Term : term,
				CommandIndex : index,
				NofityCh: make(chan int),
			}
			kv.lastClientOpMap[args.ClientId] = opRes
			kv.lastCommandMap[index] = opRes
			return true
		}
	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *KVServer) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.

	go func() {
		kv.killCh <- 0
	}()
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
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.

	kv.killCh = make(chan int, 1)

	return kv
}
