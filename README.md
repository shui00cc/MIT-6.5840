# MIT 6.5840（原6.824）

以下是本人的一些中文笔记，英文原文请访问原lab官网。

MapReduce论文原文：[rfeet.qrk (mit.edu)](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)

MapReduce论文中文翻译：[MIT 6.5840 Lab1 - MapReduce-CSDN博客](https://blog.csdn.net/weixin_51322383/article/details/132068745)

## 环境搭建

按照实验要求使用类Unix系统，我使用WSL+Goland进行实验，步骤笔记：[goland+wsl配置过程 - shui00cc - 博客园 (cnblogs.com)](https://www.cnblogs.com/shui00cc/p/17960072)

拉取实验代码：`git clone git://g.csail.mit.edu/6.5840-golabs-2023 6.5840`

## Lab1: MapReduce

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

- 参考： main/mrcoordinator.go main/mrworker.go

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

