package main

import (
	"fmt"
	"vdebu.workpool/pools"
)

func initCraw() {
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

func main() {
	fmt.Println("work start...")
	initCraw()
}
