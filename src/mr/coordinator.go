package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

// Status constants for task status, in Coordinator struct
const (
	StatusNotAllocated int = iota
	StatusWaiting
	StatusFinished
)

// Task constants representing task types, in WorkerReply struct
const (
	TaskMap int = iota
	TaskReduce
	TaskWaiting
	TaskFinished
)

type Coordinator struct {
	// Your definitions here.
	nMap             int      // total num of map task
	nReduce          int      // total num of reduce task
	mapFinished      int      // number of finished map task
	reduceFinished   int      // number of finished reduce task
	mapTaskStatus    []int    // status array for map task
	reduceTaskStatus []int    // status array for reduce task
	files            []string // input files
	mu               sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) ReceiveFinishedMap(args *WorkerArgs, reply *WorkerReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mapFinished++
	c.mapTaskStatus[args.MapTaskNumber] = StatusFinished
	return nil
}

func (c *Coordinator) ReceiveFinishedReduce(args *WorkerArgs, reply *WorkerReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reduceFinished++
	c.reduceTaskStatus[args.ReduceTaskNumber] = StatusFinished
	return nil
}

func (c *Coordinator) AllocateTask(args *WorkerArgs, reply *WorkerReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mapFinished < c.nMap {
		// allocate new map task
		allocate := -1
		for i := 0; i < c.nMap; i++ {
			if c.mapTaskStatus[i] == StatusNotAllocated {
				allocate = i
				break
			}
		}
		if allocate == -1 {
			reply.TaskType = TaskWaiting
		} else {
			reply.NReduce = c.nReduce
			reply.TaskType = TaskMap
			reply.MapTaskNumber = allocate
			reply.Filename = c.files[allocate]
			c.mapTaskStatus[allocate] = StatusWaiting
			// 10s not finished
			go func() {
				select {
				case <-time.After(10 * time.Second):
					c.mu.Lock()
					if c.mapTaskStatus[allocate] == StatusWaiting {
						c.mapTaskStatus[allocate] = StatusNotAllocated
					}
					c.mu.Unlock()
				}
			}()
		}
	} else if c.mapFinished == c.nMap && c.reduceFinished < c.nReduce {
		// all map task finished, allocate new reduce task
		allocate := -1
		for i := 0; i < c.nReduce; i++ {
			if c.reduceTaskStatus[i] == StatusNotAllocated {
				allocate = i
				break
			}
		}
		if allocate == -1 {
			reply.TaskType = TaskWaiting
		} else {
			reply.NMap = c.nMap
			reply.TaskType = TaskReduce
			reply.ReduceTaskNumber = allocate
			c.reduceTaskStatus[allocate] = StatusWaiting
			// 10s not finished
			go func() {
				select {
				case <-time.After(10 * time.Second):
					c.mu.Lock()
					if c.reduceTaskStatus[allocate] == StatusWaiting {
						c.reduceTaskStatus[allocate] = StatusNotAllocated
					}
					c.mu.Unlock()
				}
			}()
		}
	} else {
		// all finished
		reply.TaskType = TaskFinished
	}
	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	//ret := false

	// Your code here.
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := c.nReduce == c.reduceFinished
	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.files = files
	c.nMap = len(files)
	c.nReduce = nReduce
	c.mapTaskStatus = make([]int, c.nMap)
	c.reduceTaskStatus = make([]int, c.nReduce)

	c.server()
	return &c
}
