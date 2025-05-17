package pools

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

// DBTask 数据库操作任务
type DBTask struct {
	ID      int
	Type    string // 操作类型
	UserID  int
	Payload map[string]interface{}
}

// DBResult 数据库操作结果
type DBResult struct {
	TaskID       int
	Succeeded    bool
	Error        error
	RowsAffected int64
}

// DBTaskProcessPool 数据库工作池
type DBTaskProcessPool struct {
	db          *sql.DB
	workerCount int
	taskChan    chan DBTask
	resultChan  chan DBResult
	wg          sync.WaitGroup
	ctx         context.Context // 超过预定时间取消操作
	cancel      context.CancelFunc
}

func NewDBTaskProcessPool(dsn string, workerCount int, queueSize int) (*DBTaskProcessPool, error) {
	// 链接到数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// 最大连接数 -> 根据工作者的数量进行设置
	db.SetMaxOpenConns(workerCount * 2)
	db.SetMaxIdleConns(workerCount)
	// 链接!最大生命周期
	db.SetConnMaxLifetime(time.Hour)
	// 检查链接是否成功
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	// 链接成功
	ctx, cancel := context.WithCancel(context.Background())
	// 初始化工作池实例
	return &DBTaskProcessPool{
		db:          db,
		workerCount: workerCount,
		taskChan:    make(chan DBTask, queueSize),
		resultChan:  make(chan DBResult, queueSize),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start 启动工作池
func (dp *DBTaskProcessPool) Start() {
	// 创建协程
	for i := 1; i <= dp.workerCount; i++ {
		// ADD
		dp.wg.Add(1)
		workerID := i
		// 创建协程
		go func() {
			// DONE
			dp.wg.Done()
			// 监听传入的任务
			for {
				select {
				case task, ok := <-dp.taskChan:
					if !ok {
						// 任务通道已关闭
						fmt.Printf("DB工作协程 #%d 退出\n", workerID)
						return
					}
					// 处理任务
					result := dp.processDBTask(task, workerID)
					// 发送处理好的结果
					dp.resultChan <- result
				case <-dp.ctx.Done():
					// 如果当前ctx的任务已完成(上下文取消) -> cancel() 则结束当前协程
					fmt.Printf("DB工作协程 #%d 收到取消信号\n", workerID)
					return
				}
			}
		}()
		fmt.Printf("DB工作协程 #%d 已启动\n", workerID)
	}
	// 启动结果处理协程
}

// 关闭工作池
func (dp *DBTaskProcessPool) Stop() {
	fmt.Println("正在关闭工作池")
	// 关闭任务队列
	close(dp.taskChan)
	// 等待所有任务完成
	dp.wg.Wait()
	// 关闭结果通道
	close(dp.resultChan)
	// 关闭数据库链接
	dp.db.Close()
	fmt.Println("数据库连接池已关闭...")
}

// AddTask 添加数据库任务
func (dp *DBTaskProcessPool) AddTask(task DBTask) {
	// 如果工作池没有关闭就添加任务
	select {
	case dp.taskChan <- task:
	case <-dp.ctx.Done():
		fmt.Println("数据库连接池已关闭任务添加失败: ", task.ID, "...")
	}
}

// 执行数据库任务
func (dp *DBTaskProcessPool) processDBTask(task DBTask, workerID int) DBResult {
	fmt.Printf("DB工作协程 #%d 执行任务 #%d, 类型: %s, 用户ID: %d\n",
		workerID, task.ID, task.Type, task.UserID)
	// 记录任务ID
	result := DBResult{TaskID: task.ID}
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 根据Type执行不同的操作
	switch task.Type {
	case "insert":
		// 插入操作
		res, err := dp.insertUser(ctx, task)
		if err != nil {
			result.Error = err
		} else {
			result.Succeeded = true
			result.RowsAffected, _ = res.RowsAffected()
		}
	case "delete":
		// 删除操作
		// 更新操作
		res, err := dp.deleteUser(ctx, task)
		if err != nil {
			result.Error = err
		} else {
			result.Succeeded = true
			result.RowsAffected, _ = res.RowsAffected()
		}

	case "update":
		// 更新操作
		res, err := dp.updateUser(ctx, task)
		if err != nil {
			result.Error = err
		} else {
			result.Succeeded = true
			result.RowsAffected, _ = res.RowsAffected()
		}
	default: // 未知操作

	}
	// 模拟操作耗时
	time.Sleep(2 * time.Second)
	return result
}

// 插入用户数据（模拟）
func (dp *DBTaskProcessPool) insertUser(ctx context.Context, task DBTask) (sql.Result, error) {
	// 在实际应用中，这里会是真正的SQL语句
	// 这里仅作为示例，模拟操作
	fmt.Printf("模拟执行: INSERT INTO users (id, name, email) VALUES (%d, '%s', '%s')\n",
		task.UserID, task.Payload["name"], task.Payload["email"])

	// 模拟查询延迟
	time.Sleep(50 * time.Millisecond)

	// 返回模拟的结果
	return MockResult{1, 1}, nil
}

// 更新用户数据（模拟）
func (dp *DBTaskProcessPool) updateUser(ctx context.Context, task DBTask) (sql.Result, error) {
	// 模拟执行更新语句
	fmt.Printf("模拟执行: UPDATE users SET name='%s', email='%s' WHERE id=%d\n",
		task.Payload["name"], task.Payload["email"], task.UserID)

	// 模拟查询延迟
	time.Sleep(30 * time.Millisecond)

	// 返回模拟的结果
	return MockResult{0, 1}, nil
}

// 删除用户数据（模拟）
func (dp *DBTaskProcessPool) deleteUser(ctx context.Context, task DBTask) (sql.Result, error) {
	// 模拟执行删除语句
	fmt.Printf("模拟执行: DELETE FROM users WHERE id=%d\n", task.UserID)

	// 模拟查询延迟
	time.Sleep(20 * time.Millisecond)

	// 返回模拟的结果
	return MockResult{0, 1}, nil
}

// 处理数据库操作结果
func (dp *DBTaskProcessPool) handleResults() {
	for result := range dp.resultChan {
		if result.Succeeded {
			fmt.Printf("任务 #%d 执行成功，影响的行数: %d\n", result.TaskID, result.RowsAffected)
		} else {
			fmt.Printf("任务 #%d 执行失败: %v\n", result.TaskID, result.Error)
		}

		// 在真实应用中，这里可能会更新任务状态、发送通知等
	}
}
