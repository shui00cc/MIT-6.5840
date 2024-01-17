# MIT 6.5840（原6.824）

以下是本人的一些中文笔记，英文原文请访问原lab官网。

MapReduce论文原文：[rfeet.qrk (mit.edu)](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)

MapReduce论文中文翻译：[MIT 6.5840 Lab1 - MapReduce-CSDN博客](https://blog.csdn.net/weixin_51322383/article/details/132068745)

## 环境搭建

按照实验要求使用类Unix系统，我使用WSL+Goland进行实验，步骤笔记：[goland+wsl配置过程 - shui00cc - 博客园 (cnblogs.com)](https://www.cnblogs.com/shui00cc/p/17960072)

拉取实验代码：`git clone git://g.csail.mit.edu/6.5840-golabs-2023 6.5840`

## Lab1: MapReduce

### 题目信息

Lab1官网：[6.5840 Lab 1: MapReduce (mit.edu)](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)

#### 例子

计数器应用

```powershell
cd src/main
go build -buildmode=plugin ../mrapps/wc.go
rm mr-out*
go run mrsequential.go wc.so pg*.txt
more mr-out-0
```

#### 解释

- 输入文件: pg-xxx.txt
- 输出文件: mr-out-0
- mrapps/wc.go 定义了 Map 和 Reduce 函数，通过go builg加载成插件文件（`.so` 文件），这两个函数在 mrsequential.go 被加载和调用   
- "It runs the maps and reduces one at a time, in a **single process**."

#### 任务

- 分布式的 MapReduce, 由 coordinate 和 worker 组成；

- 一个 coordinate 进程和一个或多个并行执行的 worker 进程；

- 通过 RPC 对话；

- 每个 worker 进程将向 coordinate 请求一个任务，从一个或多个文件中读取任务的输入，执行任务，并将任务的输出写入一个或多个文件；

- coordinate 需要注意一个 worker 如果在10s内没有完成它的任务，则将相同的任务分配给其他的 worker 。    

- 主进程代码： main/mrcoordinator.go main/mrworker.go (仅使用，不作修改)

- 完成： mr/coordinator.go mr/worker.go mr/rpc.go

- 运行：

  ```powershell
  go build -buildmode=plugin ../mrapps/wc.go
  rm mr-out*
  go run mrcoordinator.go pg-*.txt
  ---< In one or more other windows, run some workers: >---
  go run mrworker.go wc.so
  cat mr-out-* | sort | more
  ```

- 测试：

  ```powershell
  cd ~/6.5840/src/main
  bash test-mr.sh
  ```

#### 一些规则

- 映射阶段应将中间键分成多个桶，供 nReduce 减缩任务使用，其中 nReduce 是减缩任务的数量，也就是 main/mrcoordinator.go 传递给 MakeCoordinator() 的参数。每个映射器应创建 nReduce 中间文件，供还原任务使用。
- Worker 实现应将第 X 个还原任务的输出放到 mr-out-X 文件中。
- mr-out-X 文件应包含每个还原函数输出的一行。该行应该以 Go `"%v %v"` 格式生成，并以键和值调用。请查看 main/mrsequential.go 中 "this is the correct format "的一行注释。如果您的实现与此格式偏差过大，测试脚本就会失败。
  您可以修改 mr/worker.go、mr/coordinator.go 和 mr/rpc.go。
- Worker 应将中间 Map 输出放到当前目录下的文件中，这样 Worker 之后就可以将它们作为 Reduce 任务的输入进行读取。
- main/mrcoordinator.go 希望 mr/coordinator.go 实现一个 Done() 方法，当 MapReduce 作业全部完成时返回 true；此时，mrcoordinator.go 将退出。
- 当作业完全完成时，工作进程也应退出。实现这一点的简单方法是使用 call() 的返回值：如果 worker 无法与 coordinator 取得联系，它可以认为协调器已经退出，因为作业已经完成，所以 worker 也终止。根据您的设计，您可能还会发现，coordinator 向worker 下达一个 "请退出 "的伪任务也很有帮助。

####  提示

- [指导页面](https://pdos.csail.mit.edu/6.824/labs/guidance.html)有一些关于开发和调试的提示。
  开始的一种方法是修改 mr/worker.go 的 Worker() 以向 coordinator 发送 RPC，请求任务。然后修改 coordinator ，以尚未启动的Map任务的文件名作为回应。然后修改 worker，读取该文件并调用应用程序的 Map 函数，如 mrsequential.go 所示

- 应用程序的 Map 和 Reduce 函数是在运行时使用 Go 插件包从名称以 .so 结尾的文件中加载的

- 如果更改了 mr/ 目录中的任何内容，可能需要重新构建使用的 MapReduce 插件，如 `go build -buildmode=plugin ../mrapps/wc.go`

- 此lab依赖于 workers 共享一个文件系统。当所有 Worker 都在同一台机器上运行时，共享文件系统很简单，但如果 Worker 运行在不同的机器上，则需要类似 GFS 的全局文件系统

- 中间文件的合理命名约定是 mr-X-Y，其中 X 是Map任务编号，Y 是Reduce任务编号。

- worker 的 Map 任务代码需要一种方法来将中间键/值对存储到文件中，以便在 Reduce 任务中正确读回。一种方法是使用 Go 的`encoding/json` 包。将 JSON 格式的键/值对写入打开的文件：

  ```go
  enc := json.NewEncoder(file)
  for _, kv := ... {
      err := enc.Encode(&kv)
  ```

  并读回这样的文件：

  ```go
  dec := json.NewDecoder(file)
  for {
      var kv KeyValue
      if err := dec.Decode(&kv); err != nil {
          break
      }
      kva = append(kva, kv)
  }
  ```

- worker的map部分可以使用`ihash(key)`函数（在worker. go中）来选择给定键的reduce任务。

- coordinator 作为RPC服务器，将是并发的；不要忘记锁定共享数据。

- 使用 Go 的竞赛检测器，即 go run -race。test-mr.sh 开头的注释会告诉你如何使用 -race 运行它。

- worker 有时需要等待，例如，直到最后一个Map完成，Reduce才能启动。一种可能是 worker 定期向 coordinator 询问，在每个请求之间休眠`time.Sleep()`。另一种可能是 coordinator 中相关的RPC处理程序有一个等待的循环，进行休眠`time.Sleep()`或者`sync.Cond`。Go在自己的线程中为每个RPC运行处理程序，因此一个正在等待的处理程序不需要阻止 coordinator 处理其他RPC。

- 为了测试崩溃恢复，你可以使用mrapps/crash.go应用插件。它在Map和Reduce函数中随机退出。

- 为了确保在崩溃情况下没有人能观察到部分写入的文件，MapReduce 论文提到了使用临时文件并在完全写入后以原子方式重命名的技巧。你可以使用`ioutil.TempFile`创建临时文件，并使用`os.Rename`以原子方式重命名它。

- test-mr.sh在子目录mr-tmp中运行其所有进程，因此如果发生错误并且您想查看中间或输出文件，请查看那里。可以随时修改test-mr.sh，在失败的测试后退出，这样脚本就不会继续测试（并覆盖输出文件）。

- test-mr-many.sh 连续运行 test-mr.sh 多次，这可能有助于发现低概率的错误。它接受一个参数，即运行测试的次数。不应该同时运行多个 test-mr.sh 实例，因为协调器将重用相同的套接字，导致冲突。

- Go RPC只发送名称以大写字母开头的结构体字段。子结构体也必须具有大写字段名称。

- 在调用 RPC 的 `call()` 函数时，回复结构体应包含所有默认值。RPC 调用应该如下所示：

  ```go
  Copy codereply := SomeType{}
  call(..., &reply)
  ```

  在调用之前不设置 reply 的任何字段。如果传递具有非默认字段的 reply 结构体，RPC 系统可能会在不提醒的情况下返回不正确的值。

### 我的实现

#### rpc.go

首先在rpc.go中定义RPC通信需要的结构体字段，分别是args和reply

```go
type WorkerArgs struct {
	MapTaskNumber    int // finished map task number
	ReduceTaskNumber int // finished reduce task number
}

type WorkerReply struct {
	TaskType         int    // 0:map task 1:reduce task 2:waiting 3:finished
	NMap             int    // total num of map task
	NReduce          int    // total num of reduce task
	MapTaskNumber    int    // number of map task
	ReduceTaskNumber int  	// number of reduce task 
	Filename         string // filename for worker to map
}
```

#### coordinator.go

为了避免"magic number"(在代码中直接使用未解释的常数值)，可以为上面的WorkerReply中的TaskType定义常量：

```go
const (
	TaskMap int = iota
	TaskReduce
	TaskWaiting
	TaskFinished
)
```

然后我们需要实现 Coordinator 结构体，我设置了如下字段：

```go
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
```

上面的 xxxTaskStatus 数组用于维持每个任务的状态，定义以下状态的常量：

```go
const (
	StatusNotAllocated int = iota
	StatusWaiting
	StatusFinished
)
```

接着，我们需要实现RPC处理函数，提供给 worker 调用。首先是任务完成的函数，注意加锁以及defer解锁：

```go
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
```

最后是分配任务的RPC处理函数，代码逻辑如下：

1. 当map任务尚未全部完成时，分配map任务

   a. 通过mapTaskStatus[]寻找尚未分配的map任务并分配。如果任务全部已分配，则返回*TaskWaiting*状态

   b. 分配任务后，**启动一个goroutine监听10s后任务是否完成**，如果仍未完成则重新分配此任务

2. 当map任务全部完成后分配reduce，逻辑与上面的map相同

3. 如果所有任务都已完成，返回*TaskFinished*状态

```go
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
```

最后补全MakeCoordinator()函数，其提供了nReduce作为 reduce task 的数量：

```go
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
```

在Done中补充判断，以在mrcoordinator.go中退出时使用：

```go
func (c *Coordinator) Done() bool {
    //ret := false
    
    // Your code here.
    c.mu.Lock()
    defer c.mu.Unlock()
    ret := c.nReduce == c.reduceFinished
    return ret
}
```

#### worker.go

首先和例子中wc.go一样，为我们自定义的ByKey结构体实现`sort.Sort(interface{})`需要的三个方法，以便后续排序直接使用：

```go
// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }
```

然后实现最核心的Worker()方法，代码逻辑如下：

首先通过RPC的call()调用 coordinator 的任务分配函数，获得回复后对`reply.TaskType`进行判断

1. *TaskFinished*，任务全部完成，直接退出

2. *TaskMap*

   2.1 首先读取分配的文件并调用mapf函数，生成键值数组`var intermediate []KeyValue`

   2.2 然后定义二维数组`buckets := make([][]KeyValue, reply.NReduce)`，将`intermediate`的内容append到指定序号的buckets[i]，使用`ihash(kva.Key)%reply.NReduce`作为分配的reduce任务序号i

   2.3 **为了确保在整个写入操作完成之前，其他任务不会读取到不完整的文件**，我们使用提示中给出的`ioutil.TempFile`和`json.NewEncoder`来生成临时文件，临时文件写入完成后再重命名为中间文件mr-X-Y，X是当前map任务的序号，Y是当前的buckets[i]的i，即reduce的序号。

   2.4 最后call 调用finishedxxx函数告诉 coordinator 此map任务已完成

3. *TaskReduce*，从中间文件mr-X-Y中读取数据，临时文件操作同上，进行reduce操作的过程参考wc.go，最后也需调用finish函数

4. TaskWaiting，执行`time.Sleep(time.*Second*)`等待

```go
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
```
