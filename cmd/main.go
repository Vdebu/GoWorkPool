package main

import (
	"fmt"
	"math/rand"
	"time"
	"vdebu.workpool/pools"
)

// 爬
func initCrawProcessPool() {
	// 初始化工作池
	pool := pools.NewWorkerPool(5, 100)
	// 启动工作池进入阻塞
	pool.Start()

	// 准备要处理的数据
	// 准备要爬取的URL列表
	urls := []string{
		"https://golang.org",
		"https://github.com",
		"https://stackoverflow.com",
		"https://www.google.com",
		"https://www.baidu.com",
		"https://news.ycombinator.com",
		"https://reddit.com",
		"https://medium.com",
		"https://dev.to",
		"https://www.zhihu.com",
		"https://www.bilibili.com",
	}
	// 添加任务
	for i, url := range urls {
		pool.AddTask(pools.CrawTask{
			URL: url,
			ID:  i + 1,
		})
	}
	// 等待所有任务完成
	pool.Stop()
}

// 图片处理
func initImageProcessPool() {
	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())
	// 创建有12个协程的图片处理工作池
	pool := pools.NewImageProcessorPool(12, 100)
	// 启动工作池 -> 监听数据
	pool.Start()
	// 生成任务
	imgCount := 39
	for i := 1; i <= imgCount; i++ {
		// 随机生成图片大小
		size := rand.Intn(900) + 100
		// 创建任务
		task := pools.ImageTask{
			FileName: fmt.Sprintf("image_%d.jpg", i),
			ID:       i,
			Size:     size,
		}
		// 将任务加入队列
		pool.AddTask(task)
		// 模拟不规律的任务到达
		time.Sleep(time.Duration(rand.Intn(800)) * time.Millisecond)
	}
	fmt.Println("等待所有任务完成...")
	time.Sleep(time.Second * 2)
	// 停止工作池
	pool.Stop()
	// 等待最后的结果打印
	time.Sleep(500 * time.Millisecond)
}

// 日志分析
func initLogAnalyzerPool() {
	// 创建日志分析工作池，使用4个工作协程
	analyzer := pools.NewLogAnalyzerPool(4)
	analyzer.Start()

	// 模拟添加日志文件分析任务
	logFiles := []pools.LogAnalysisTask{
		{
			FilePath:    "/var/log/app/api-service.log",
			ServiceName: "api-service",
		},
		{
			FilePath:    "/var/log/app/user-service.log",
			ServiceName: "user-service",
		},
		{
			FilePath:    "/var/log/app/payment-service.log",
			ServiceName: "payment-service",
		},
		{
			FilePath:    "/var/log/app/notification-service.log",
			ServiceName: "notification-service",
		},
		{
			FilePath:    "/var/log/app/auth-service.log",
			ServiceName: "auth-service",
		},
	}
	// 添加分析任务
	for _, task := range logFiles {
		analyzer.AddTask(task)
	}

	// 任务添加完毕等待分析完成
	fmt.Println("等待日志分析完成...")
	time.Sleep(2 * time.Second)

	// 停止工作池
	analyzer.Stop()

	// 生成并输出报告
	report := analyzer.GenerateReport()
	fmt.Println(report)
}
func main() {
	fmt.Println("work start...")
	//initCrawProcessPool()
	initImageProcessPool()
}
