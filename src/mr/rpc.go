package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
// Capital letters!
type WorkerArgs struct {
	MapTaskNumber    int // finished map task number
	ReduceTaskNumber int // finished reduce task number
}

type WorkerReply struct {
	TaskType         int    // 0:map task 1:reduce task 2:waiting 3:finished
	NMap             int    // total num of map task
	NReduce          int    // total num of reduce task
	MapTaskNumber    int    // number of map task
	ReduceTaskNumber int    // number of reduce task
	Filename         string // filename for worker to map
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
