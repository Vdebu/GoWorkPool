package pools

import (
	"fmt"
	"sync"
	"time"
)

// ImageTask 图片处理任务
type ImageTask struct {
	FileName string
	ID       int
	Size     int // 大小KB
}

// ImageProcessorResult 处理结果
type ImageProcessorResult struct {
	TaskID     int           // 任务ID
	FileName   string        // 文件名
	ResultSize int           // 结果大小
	Duration   time.Duration // 处理用时
}

// ImageProcessorPool 图像处理工作池
type ImageProcessorPool struct {
	workerCount int
	taskChan    chan ImageTask
	resultChan  chan ImageProcessorResult
	wg          sync.WaitGroup
	done        chan struct{}
}

// NewImageProcessorPool 初始化工作池
func NewImageProcessorPool(workerCount int, queueSize int) *ImageProcessorPool {
	return &ImageProcessorPool{
		workerCount: workerCount,                                // worker数量
		taskChan:    make(chan ImageTask, queueSize),            // 队列大小
		resultChan:  make(chan ImageProcessorResult, queueSize), // 存储结果
		done:        make(chan struct{}),
	}
}

// Start 启动图片处理工作池
func (ip *ImageProcessorPool) Start() {
	// 启动工作协程
	for i := 1; i <= ip.workerCount; i++ {
		// ADD
		ip.wg.Add(1)
		workerID := i
		go func() {
			// Done
			defer ip.wg.Done()
			for {
				// 启动一个循环监听任务(默认阻塞) -> 获取并执行
				select {
				// 尝试获取任务
				case task, ok := <-ip.taskChan:
					if !ok {
						// 没有取成功说明任务全做完了 -> 结束协程回收资源
						fmt.Printf("图片处理工作协程 #%d 退出\n", workerID)
						return
					}
					// 开始处理任务
					result := processImage(task, workerID)
					// 将任务结果进行存储
					ip.resultChan <- result
				case <-ip.done:
					// 接收到退出信号退出协程
					fmt.Printf("图片处理工作协程 #%d 收到退出信号\n", workerID)
					return
				}
			}
		}()
		fmt.Printf("图片处理工作协程 #%d 已启动\n", workerID)
	}
	// 启动结果收集协程
	// 注意这里一定要异步收集结果否则就会导致死锁
	go ip.collectResult()
}

// AddTask 添加图片处理任务
func (ip *ImageProcessorPool) AddTask(task ImageTask) {
	ip.taskChan <- task
}

// Stop 立即停止工作池
func (ip *ImageProcessorPool) Stop() {
	// 发出退出信号
	close(ip.done)
	// 告诉工作池没有更多任务了
	close(ip.taskChan)
	// 等待所有正在进行的协程完成
	ip.wg.Wait()
	// 关闭结果通道
	close(ip.resultChan)
	fmt.Println("所有协程已完成并退出...")
}

// 结果收集 & 处理协程
func (ip *ImageProcessorPool) collectResult() {
	// 默认阻塞
	for result := range ip.resultChan {
		// 将结果存储到数据库或者发送到其他的服务
		fmt.Printf("处理完成: 文件 %s (任务 #%d), 处理后大小: %d KB, 耗时: %v\n",
			result.FileName, result.TaskID, result.ResultSize, result.Duration)
	}
	fmt.Println("结果收集 & 处理完成")
}

// 处理图片的函数(模拟)
func processImage(task ImageTask, workerID int) ImageProcessorResult {
	fmt.Printf("工作协程 #%d 开始处理图片: %s (任务 #%d, 大小: %d KB)\n",
		workerID, task.FileName, task.ID, task.Size)
	// 模拟处理图片耗时与大小成正比
	processTime := time.Duration(task.Size) * time.Millisecond
	// 模拟处理
	time.Sleep(processTime)
	// 模拟图片处理/压缩后的大小
	newSize := task.Size / 2
	// 返回结果
	return ImageProcessorResult{
		TaskID:     task.ID,
		FileName:   task.FileName,
		ResultSize: newSize,
		Duration:   processTime,
	}
}
