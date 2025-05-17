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
		// 模拟不规则的任务到达
		time.Sleep(time.Duration(rand.Intn(800)) * time.Millisecond)
	}
	fmt.Println("等待所有任务完成...")
	time.Sleep(time.Second * 2)
	// 停止工作池
	pool.Stop()
	// 等待最后的结果打印
	time.Sleep(500 * time.Millisecond)
}
func main() {
	fmt.Println("work start...")
	//initCrawProcessPool()
	initImageProcessPool()
}
