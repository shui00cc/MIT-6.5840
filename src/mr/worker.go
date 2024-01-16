package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
	for {
		args := WorkerArgs{}
		reply := WorkerReply{}
		ok := call("Coordinator.AllocateTask", &args, &reply)
		if !ok || reply.TaskType == TaskFinished {
			// coordinator died || job finished
			break
		}
		if reply.TaskType == TaskMap {
			// map task
			intermediate := []KeyValue{}
			file, err := os.Open(reply.Filename)
			if err != nil {
				log.Fatalf("cannot open %v: %v", reply.Filename, err)
			}
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v: %v", reply.Filename, err)
			}
			file.Close()
			kva := mapf(reply.Filename, string(content))
			intermediate = append(intermediate, kva...)

			// hash into buckets
			buckets := make([][]KeyValue, reply.NReduce)
			for i := range buckets {
				buckets[i] = []KeyValue{}
			}
			for _, kva := range intermediate {
				buckets[ihash(kva.Key)%reply.NReduce] = append(buckets[ihash(kva.Key)%reply.NReduce], kva)
			}

			// write into intermediate files
			for i := range buckets {
				oname := fmt.Sprintf("mr-%d-%d", reply.MapTaskNumber, i)
				ofile, _ := ioutil.TempFile("", oname+"*")
				enc := json.NewEncoder(ofile)
				for _, kva := range buckets[i] {
					err := enc.Encode(&kva)
					if err != nil {
						log.Fatalf("cannot write into %v: %v", oname, err)
					}
				}
				os.Rename(ofile.Name(), oname)
				ofile.Close()
			}

			// call to send finish msg
			finishedArgs := WorkerArgs{reply.MapTaskNumber, -1}
			finishedReply := WorkerReply{}
			call("Coordinator.ReceiveFinishedMap", &finishedArgs, &finishedReply)
		} else if reply.TaskType == TaskReduce {
			// reduce task
			// collect Key-Value from mr-X-Y
			intermediate := []KeyValue{}
			for i := 0; i < reply.NMap; i++ {
				iname := fmt.Sprintf("mr-%d-%d", i, reply.ReduceTaskNumber)
				file, err := os.Open(iname)
				if err != nil {
					log.Fatalf("cannot open %v: %v", iname, err)
				}
				dec := json.NewDecoder(file)
				for {
					var kv KeyValue
					if err := dec.Decode(&kv); err != nil {
						break
					}
					intermediate = append(intermediate, kv)
				}
				file.Close()
			}

			// sort by key
			sort.Sort(ByKey(intermediate))

			// output file
			oname := fmt.Sprintf("mr-out-%d", reply.ReduceTaskNumber)
			ofile, _ := ioutil.TempFile("", oname+"*")

			//
			// call Reduce on each distinct key in intermediate[],
			// and print the result to mr-out-0.
			//
			i := 0
			for i < len(intermediate) {
				j := i + 1
				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}
				output := reducef(intermediate[i].Key, values)

				// this is the correct format for each line of Reduce output.
				fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

				i = j
			}
			os.Rename(ofile.Name(), oname)
			ofile.Close()

			// delete intermediate files
			for i := 0; i < reply.NMap; i++ {
				iname := fmt.Sprintf("mr-%d-%d", i, reply.ReduceTaskNumber)
				err := os.Remove(iname)
				if err != nil {
					log.Fatalf("cannot delete %v: %v", iname, err)
				}
			}

			// call to send finish msg
			finishedArgs := WorkerArgs{-1, reply.ReduceTaskNumber}
			finishedReply := WorkerReply{}
			call("Coordinator.ReceiveFinishedReduce", &finishedArgs, &finishedReply)
		}
		// else: reply.TaskType == TaskWaiting
		time.Sleep(time.Second)
	}
	return
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
