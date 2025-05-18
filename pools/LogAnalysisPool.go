package pools

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp   time.Time // 日志时间
	Level       string    // INFO/ERROR/FATAL
	Service     string    // 服务层
	Message     string    // 内容
	RequestID   string    // 请求ID
	ElapsedTime int64     // 历经时间
	StatusCode  int       // 状态码
}

// LogAnalysisTask 日志分析任务
type LogAnalysisTask struct {
	FilePath    string // 日志文件目录
	ServiceName string // 文件名称
}

// LogAnalysisResult 日志分析结果
type LogAnalysisResult struct {
	FilePath           string
	ServiceName        string  // 各服务统计
	TotalEntries       int     // 汇总统计
	ErrorCount         int     // 汇总统计
	WarningCount       int     // 汇总统计
	InfoCount          int     // 汇总统计
	AvgResponseTime    float64 // 平均响应时间（加权平均）
	MaxResponseTime    int64   // 最大响应时间
	ErrorRate          float64
	StatusCodeCounts   map[int]int // 状态码统计 WordCount
	ProcessingDuration time.Duration
}

// LogAnalyzerPool 日志分析工作池
type LogAnalyzerPool struct {
	workerCount int
	taskChan    chan LogAnalysisTask
	resultChan  chan LogAnalysisResult
	wg          sync.WaitGroup
	mutex       sync.Mutex          // 互斥锁
	results     []LogAnalysisResult // 存储通道中的result
}

// NewLogAnalyzerPool 创建日志分析工作池
func NewLogAnalyzerPool(workerCount int) *LogAnalyzerPool {
	return &LogAnalyzerPool{
		workerCount: workerCount,
		// 固定容量100
		taskChan:   make(chan LogAnalysisTask, 100),
		resultChan: make(chan LogAnalysisResult, 100),
		results:    make([]LogAnalysisResult, 0),
	}
}

// Start 启动工作池
func (la *LogAnalyzerPool) Start() {
	// 启动工作协程
	for i := 0; i < la.workerCount; i++ {
		la.wg.Add(1)
		workerID := i

		go func() {
			defer la.wg.Done()

			for task := range la.taskChan {
				startTime := time.Now()

				fmt.Printf("工作协程 #%d 开始分析日志文件: %s (服务: %s)\n",
					workerID, task.FilePath, task.ServiceName)

				// 分析日志文件
				result := la.analyzeLogFile(task)
				result.ProcessingDuration = time.Since(startTime)

				// 发送结果
				la.resultChan <- result
			}
		}()

		fmt.Printf("日志分析工作协程 #%d 已启动\n", workerID)
	}

	// 启动结果收集协程
	go la.collectResults()
}

// AddTask 添加分析任务
func (la *LogAnalyzerPool) AddTask(task LogAnalysisTask) {
	la.taskChan <- task
}

// Stop 停止工作池
func (la *LogAnalyzerPool) Stop() {
	close(la.taskChan)
	la.wg.Wait()
	close(la.resultChan)

	// 等待结果收集完成
	time.Sleep(100 * time.Millisecond)
}

// GetResults 获取汇总结果
func (la *LogAnalyzerPool) GetResults() []LogAnalysisResult {
	// 上锁防止冲突
	la.mutex.Lock()
	defer la.mutex.Unlock()

	return la.results
}

// 结果收集协程
func (la *LogAnalyzerPool) collectResults() {
	// 遍历通道中存储的结果输出并将其转移至切片
	for result := range la.resultChan {
		fmt.Printf("完成日志分析: %s, 条目数: %d, 错误数: %d, 平均响应时间: %.2f ms\n",
			result.FilePath, result.TotalEntries, result.ErrorCount, result.AvgResponseTime)

		// 上锁保存结果
		la.mutex.Lock()
		la.results = append(la.results, result)
		la.mutex.Unlock()
	}

	fmt.Println("所有日志分析结果已收集")
}

// 分析单个日志文件
func (la *LogAnalyzerPool) analyzeLogFile(task LogAnalysisTask) LogAnalysisResult {
	result := LogAnalysisResult{
		FilePath:         task.FilePath,
		ServiceName:      task.ServiceName,
		StatusCodeCounts: make(map[int]int),
	}

	// 打开日志文件
	file, err := os.Open(task.FilePath)
	if err != nil {
		fmt.Printf("无法打开日志文件 %s: %v\n", task.FilePath, err)
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	totalResponseTime := int64(0)

	// 日志行解析正则表达式 - 这里使用一个简化的示例
	// 实际场景中可能需要根据日志格式调整
	logRegexp := regexp.MustCompile(`\[(.*?)\]\s+(\w+)\s+\[(\w+)\]\s+(.*?)\s+requestID=(\S+)\s+elapsed=(\d+)ms\s+status=(\d+)`)

	// 逐行处理日志
	for scanner.Scan() {
		line := scanner.Text()
		matches := logRegexp.FindStringSubmatch(line)

		if len(matches) >= 8 {
			// 解析日志条目
			entry := LogEntry{}

			// 解析时间戳（根据实际日志格式调整）
			entry.Timestamp, _ = time.Parse("2006-01-02 15:04:05", matches[1])
			entry.Level = matches[2]
			entry.Service = matches[3]
			entry.Message = matches[4]
			entry.RequestID = matches[5]
			entry.ElapsedTime = parseInt64(matches[6])
			entry.StatusCode = parseInt(matches[7])

			// 更新统计信息
			result.TotalEntries++

			// 计算日志级别统计
			switch strings.ToUpper(entry.Level) {
			case "ERROR":
				result.ErrorCount++
			case "WARN", "WARNING":
				result.WarningCount++
			case "INFO":
				result.InfoCount++
			}

			// 记录响应时间
			totalResponseTime += entry.ElapsedTime
			if entry.ElapsedTime > result.MaxResponseTime {
				result.MaxResponseTime = entry.ElapsedTime
			}

			// 记录状态码
			if entry.StatusCode > 0 {
				result.StatusCodeCounts[entry.StatusCode]++
			}
		}
	}

	// 计算平均响应时间
	if result.TotalEntries > 0 {
		result.AvgResponseTime = float64(totalResponseTime) / float64(result.TotalEntries)
		result.ErrorRate = float64(result.ErrorCount) / float64(result.TotalEntries)
	}

	return result
}

// 辅助函数: 解析整数
func parseInt(s string) int {
	var value int
	fmt.Sscanf(s, "%d", &value)
	return value
}

// 辅助函数: 解析int64
func parseInt64(s string) int64 {
	var value int64
	fmt.Sscanf(s, "%d", &value)
	return value
}

// GenerateReport 生成分析报告
func (la *LogAnalyzerPool) GenerateReport() string {
	la.mutex.Lock()
	results := la.results
	la.mutex.Unlock()

	if len(results) == 0 {
		return "没有可用的分析结果"
	}

	var sb strings.Builder

	sb.WriteString("# 日志分析报告\n\n")
	sb.WriteString(fmt.Sprintf("分析时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("分析的日志文件数: %d\n\n", len(results)))

	// 汇总统计
	var totalEntries, totalErrors, totalWarnings int
	var totalAvgTime float64
	statusCodeTotals := make(map[int]int)

	for _, r := range results {
		totalEntries += r.TotalEntries
		totalErrors += r.ErrorCount
		totalWarnings += r.WarningCount
		totalAvgTime += r.AvgResponseTime * float64(r.TotalEntries)

		for code, count := range r.StatusCodeCounts {
			statusCodeTotals[code] += count
		}
	}

	overallAvgTime := 0.0
	if totalEntries > 0 {
		overallAvgTime = totalAvgTime / float64(totalEntries)
	}

	sb.WriteString("## 总体统计\n\n")
	sb.WriteString(fmt.Sprintf("- 总日志条目: %d\n", totalEntries))
	sb.WriteString(fmt.Sprintf("- 错误总数: %d (%.2f%%)\n", totalErrors, float64(totalErrors)*100/float64(totalEntries)))
	sb.WriteString(fmt.Sprintf("- 警告总数: %d (%.2f%%)\n", totalWarnings, float64(totalWarnings)*100/float64(totalEntries)))
	sb.WriteString(fmt.Sprintf("- 平均响应时间: %.2f ms\n\n", overallAvgTime))

	// HTTP状态码统计
	sb.WriteString("## HTTP状态码分布\n\n")
	for code, count := range statusCodeTotals {
		sb.WriteString(fmt.Sprintf("- %d: %d (%.2f%%)\n", code, count, float64(count)*100/float64(totalEntries)))
	}
	sb.WriteString("\n")

	// 各服务统计
	sb.WriteString("## 服务统计\n\n")
	serviceStats := make(map[string]*LogAnalysisResult)

	for _, r := range results {
		service := r.ServiceName
		if service == "" {
			service = "未知服务"
		}

		if stats, exists := serviceStats[service]; exists {
			stats.TotalEntries += r.TotalEntries
			stats.ErrorCount += r.ErrorCount
			stats.WarningCount += r.WarningCount
			stats.InfoCount += r.InfoCount

			// 更新平均响应时间（加权平均）
			totalTime := stats.AvgResponseTime * float64(stats.TotalEntries-r.TotalEntries)
			totalTime += r.AvgResponseTime * float64(r.TotalEntries)
			stats.AvgResponseTime = totalTime / float64(stats.TotalEntries)

			// 更新最大响应时间
			if r.MaxResponseTime > stats.MaxResponseTime {
				stats.MaxResponseTime = r.MaxResponseTime
			}

			// 合并状态码统计
			for code, count := range r.StatusCodeCounts {
				stats.StatusCodeCounts[code] += count
			}
		} else {
			// 复制结果创建新的服务统计
			stats := &LogAnalysisResult{
				ServiceName:      service,
				TotalEntries:     r.TotalEntries,
				ErrorCount:       r.ErrorCount,
				WarningCount:     r.WarningCount,
				InfoCount:        r.InfoCount,
				AvgResponseTime:  r.AvgResponseTime,
				MaxResponseTime:  r.MaxResponseTime,
				StatusCodeCounts: make(map[int]int),
			}

			for code, count := range r.StatusCodeCounts {
				stats.StatusCodeCounts[code] = count
			}

			serviceStats[service] = stats
		}
	}

	// 输出各服务统计
	for service, stats := range serviceStats {
		sb.WriteString(fmt.Sprintf("### %s\n\n", service))
		sb.WriteString(fmt.Sprintf("- 总日志条目: %d\n", stats.TotalEntries))
		sb.WriteString(fmt.Sprintf("- 错误数: %d (%.2f%%)\n", stats.ErrorCount, float64(stats.ErrorCount)*100/float64(stats.TotalEntries)))
		sb.WriteString(fmt.Sprintf("- 警告数: %d (%.2f%%)\n", stats.WarningCount, float64(stats.WarningCount)*100/float64(stats.TotalEntries)))
		sb.WriteString(fmt.Sprintf("- 平均响应时间: %.2f ms\n", stats.AvgResponseTime))
		sb.WriteString(fmt.Sprintf("- 最大响应时间: %d ms\n\n", stats.MaxResponseTime))
	}

	return sb.String()
}
