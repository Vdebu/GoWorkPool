package pools

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// CrawTask 存储任务队列
type CrawTask struct {
	URL string
	ID  int
}

// crawPool 工作池实体
type crawPool struct {
	taskQueue chan CrawTask  // 只要声明是个通道就好了之后new的时候进行初始化
	workCount int            // 工作者个数
	wg        sync.WaitGroup // WaitGroup
}

// NewWorkerPool 初始化工作池
func NewWorkerPool(workCount int, queueSize int) *crawPool {
	return &crawPool{
		workCount: workCount,                      // 初始化对应数目的工人
		taskQueue: make(chan CrawTask, queueSize), // 初始化对应数目的chan用于维护数据
	}
}

// Start 启动工作池
func (c *crawPool) Start() {
	// 根据初始化的信息创建指定改数量的工作协程
	for i := 1; i <= c.workCount; i++ {
		// Add
		c.wg.Add(1)
		workerID := i
		// 启动协程
		go func() {
			// Done
			defer c.wg.Done()
			// 不断从任务队列中获取任务并执行
			for task := range c.taskQueue {
				// 执行任务
				processURL(task, workerID)
			}
		}()
		fmt.Println("工作协程:", workerID, "已启动...")
	}
}

// AddTask 向工作池中添加任务
func (c *crawPool) AddTask(task CrawTask) {
	// 向当前工作池的任务队列添加任务
	c.taskQueue <- task
}
func processURL(task CrawTask, workerID int) {
	fmt.Printf("工作协程 #%d 开始处理任务 #%d: %s\n", workerID, task.ID, task.URL)
	// 发送http请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	// 发送get请求
	resp, err := client.Get(task.URL)
	if err != nil {
		fmt.Println("请求失败...")
		return
	}
	defer resp.Body.Close()
	// 读取响应体并输出
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应体失败...")
		return
	}
	// 输出内容
	fmt.Printf("任务 #%d 完成: %s, 状态码: %d, 内容长度: %d 字节\n",
		task.ID, task.URL, resp.StatusCode, len(body))
	//fmt.Println(string(body))
	// 模拟处理内容耗时
	time.Sleep(time.Second)
}

// Stop 等待所有任务完毕关闭工作池
func (c *crawPool) Stop() {
	// 关闭通道开始任务
	close(c.taskQueue)
	// 等待任务全部结束
	c.wg.Wait()
	fmt.Println("所有任务均已完成...")
}
