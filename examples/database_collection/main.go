package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/serial"
)

// 本示例展示：数据库数据采集功能
//
// 功能：
// 1. 数据读取 - 执行SQL查询获取数据
// 2. 数据监听 - 定时轮询数据库变化
// 3. 数据写入 - 执行INSERT/UPDATE/DELETE操作
// 4. 事务处理 - 批量操作的事务支持

func main() {
	fmt.Println("=== 数据库数据采集示例 ===\n")
	fmt.Println("支持的数据库类型: MySQL, PostgreSQL, SQL Server, Oracle, SQLite\n")

	// 数据库配置示例（使用SQLite进行演示）
	config := &serial.DatabaseConfig{
		Driver:   "sqlite3",
		Database: "./test_industrial.db", // SQLite数据库文件

		// 对于其他数据库类型的配置示例：
		// MySQL:
		// Driver:   "mysql",
		// Host:     "localhost",
		// Port:     3306,
		// Database: "industrial_db",
		// Username: "user",
		// Password: "password",

		// PostgreSQL:
		// Driver:   "postgres",
		// Host:     "localhost",
		// Port:     5432,
		// Database: "industrial_db",
		// Username: "user",
		// Password: "password",
		// SSLMode:  "disable",

		// SQL Server:
		// Driver:   "sqlserver",
		// Host:     "localhost",
		// Port:     1433,
		// Database: "industrial_db",
		// Username: "sa",
		// Password: "password",

		// 连接池配置
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 1 * time.Hour,

		// 轮询配置
		PollingQuery:    "SELECT * FROM sensor_data WHERE updated_at > datetime('now', '-1 minute')",
		PollingInterval: 5 * time.Second,
		DefaultTable:    "sensor_data",
	}

	fmt.Println("📋 数据库配置:")
	fmt.Printf("  类型: %s\n", config.Driver)
	if config.Driver == "sqlite3" {
		fmt.Printf("  文件: %s\n", config.Database)
	} else {
		fmt.Printf("  主机: %s:%d\n", config.Host, config.Port)
		fmt.Printf("  数据库: %s\n", config.Database)
	}
	fmt.Printf("  轮询间隔: %v\n", config.PollingInterval)
	fmt.Println()

	// 创建数据库数据源
	db, err := serial.NewDatabase(config)
	if err != nil {
		log.Fatalf("创建数据库连接失败: %v", err)
	}

	// 打开连接
	err = db.Open("")
	if err != nil {
		log.Printf("连接数据库失败: %v\n", err)
		log.Println("\n提示: 请确保:")
		log.Println("  1. 数据库服务已启动")
		log.Println("  2. 连接参数正确")
		log.Println("  3. 用户有访问权限")
		log.Println("  4. 对于SQLite，确保数据库文件存在或可创建")
		log.Println("\n将使用模拟模式演示...")
		useSimulationMode()
		return
	}

	defer db.Close()

	fmt.Println("✓ 数据库连接成功")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 创建示例表
	fmt.Println("=== 功能1: 创建示例表 ===")
	if err := createExampleTables(ctx, db); err != nil {
		log.Printf("创建表失败: %v", err)
	}

	// 2. 数据写入演示
	fmt.Println("\n=== 功能2: 数据写入 ===")
	demonstrateDataWriting(ctx, db)

	// 3. 数据读取演示
	fmt.Println("\n=== 功能3: 数据查询 ===")
	demonstrateDataReading(ctx, db)

	// 4. 事务处理演示
	fmt.Println("\n=== 功能4: 事务处理 ===")
	demonstrateTransaction(ctx, db)

	// 5. 数据监听演示（轮询）
	fmt.Println("\n=== 功能5: 数据监听（轮询）===")
	err = db.StartPolling(ctx)
	if err != nil {
		log.Printf("启动轮询失败: %v", err)
	} else {
		fmt.Println("数据轮询已启动")
		go monitorDataUpdates(db)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- 数据库采集运行中 ---")
	fmt.Println("轮询周期: 5秒")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)

	// 显示最终统计
	status := db.GetStatus()
	fmt.Println("\n=== 运行统计 ===")
	fmt.Printf("请求次数: %d\n", status.RequestCount)
	fmt.Printf("错误次数: %d\n", status.ErrorCount)

	fmt.Println("\n✓ 示例完成!")
}

// createExampleTables 创建示例表
func createExampleTables(ctx context.Context, db *serial.Database) error {
	// 创建传感器数据表
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS sensor_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			sensor_type TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT,
			quality REAL DEFAULT 1.0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := db.Execute(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	fmt.Println("✓ 创建表: sensor_data")
	return nil
}

// demonstrateDataWriting 演示数据写入
func demonstrateDataWriting(ctx context.Context, db *serial.Database) {
	// 插入多条传感器数据
	sensors := []struct {
		DeviceID   string
		SensorType string
		Value      float64
		Unit       string
	}{
		{"PLC-001", "temperature", 25.5, "°C"},
		{"PLC-001", "pressure", 1.2, "bar"},
		{"PLC-002", "temperature", 30.2, "°C"},
		{"PLC-002", "vibration", 0.05, "mm/s"},
		{"PLC-003", "flow", 150.0, "L/min"},
	}

	for _, s := range sensors {
		insertSQL := `
			INSERT INTO sensor_data (device_id, sensor_type, value, unit)
			VALUES (?, ?, ?, ?)
		`

		rowsAffected, err := db.Execute(ctx, insertSQL, s.DeviceID, s.SensorType, s.Value, s.Unit)
		if err != nil {
			fmt.Printf("  ✗ 插入失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ 插入数据: %s - %s = %.2f %s (影响 %d 行)\n",
			s.DeviceID, s.SensorType, s.Value, s.Unit, rowsAffected)
	}
}

// demonstrateDataReading 演示数据读取
func demonstrateDataReading(ctx context.Context, db *serial.Database) {
	// 查询所有数据
	query := "SELECT device_id, sensor_type, value, unit, quality, created_at FROM sensor_data ORDER BY created_at DESC LIMIT 10"

	results, err := db.Query(ctx, query)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	fmt.Printf("查询结果: %d 条记录\n", len(results))
	for i, row := range results {
		fmt.Printf("  [%d] 设备: %v, 类型: %v, 值: %v %v, 质量: %v\n",
			i+1, row["device_id"], row["sensor_type"], row["value"], row["unit"], row["quality"])
	}

	// 条件查询
	fmt.Println("\n条件查询 (温度传感器):")
	query = "SELECT device_id, value, unit FROM sensor_data WHERE sensor_type = ? ORDER BY value DESC"
	results, err = db.Query(ctx, query, "temperature")
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	for i, row := range results {
		fmt.Printf("  [%d] %v: %.2v %v\n",
			i+1, row["device_id"], row["value"], row["unit"])
	}

	// 聚合查询
	fmt.Println("\n聚合查询 (平均温度):")
	query = "SELECT AVG(value) as avg_temp, COUNT(*) as count FROM sensor_data WHERE sensor_type = 'temperature'"
	results, err = db.Query(ctx, query)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	if len(results) > 0 {
		fmt.Printf("  平均温度: %.2v°C, 样本数: %v\n", results[0]["avg_temp"], results[0]["count"])
	}
}

// demonstrateTransaction 演示事务处理
func demonstrateTransaction(ctx context.Context, db *serial.Database) {
	err := db.Transaction(ctx, func(tx *sql.Tx) error {
		// 在事务中插入多条数据
		insertSQL := "INSERT INTO sensor_data (device_id, sensor_type, value, unit) VALUES (?, ?, ?, ?)"

		_, err := tx.ExecContext(ctx, insertSQL, "PLC-004", "temperature", 28.0, "°C")
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, insertSQL, "PLC-004", "humidity", 65.0, "%")
		if err != nil {
			return err
		}

		fmt.Println("  ✓ 事务提交成功: 插入2条记录")
		return nil
	})

	if err != nil {
		fmt.Printf("  ✗ 事务失败: %v\n", err)
	}
}

// monitorDataUpdates 监听数据更新
func monitorDataUpdates(db *serial.Database) {
	dataChan := db.GetDataChannel()

	updateCount := 0

	for update := range dataChan {
		updateCount++

		fmt.Printf("[监听] 数据变化 #%d\n", updateCount)
		fmt.Printf("       时间: %s\n", update.Timestamp.Format("15:04:05"))

		if data, ok := update.Value.(map[string]interface{}); ok {
			fmt.Printf("       设备: %v\n", data["device_id"])
			fmt.Printf("       类型: %v\n", data["sensor_type"])
			fmt.Printf("       值: %v %v\n", data["value"], data["unit"])
		}

		fmt.Println()
	}
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode() {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	fmt.Println("1. 创建数据库表:")
	fmt.Println("   CREATE TABLE sensor_data (...) ✓")

	fmt.Println("\n2. 插入数据:")
	fmt.Println("   INSERT: PLC-001, temperature, 25.5°C ✓")
	fmt.Println("   INSERT: PLC-001, pressure, 1.2 bar ✓")
	fmt.Println("   INSERT: PLC-002, temperature, 30.2°C ✓")

	fmt.Println("\n3. 查询数据:")
	fmt.Println("   SELECT * FROM sensor_data")
	fmt.Println("   结果: 3 条记录")
	fmt.Println("     [1] PLC-001: temperature = 25.5°C")
	fmt.Println("     [2] PLC-001: pressure = 1.2 bar")
	fmt.Println("     [3] PLC-002: temperature = 30.2°C")

	fmt.Println("\n4. 聚合查询:")
	fmt.Println("   SELECT AVG(value) FROM sensor_data WHERE sensor_type='temperature'")
	fmt.Println("   结果: 平均温度 = 27.85°C")

	fmt.Println("\n5. 事务处理:")
	fmt.Println("   BEGIN TRANSACTION")
	fmt.Println("   INSERT: PLC-004, temperature, 28.0°C")
	fmt.Println("   INSERT: PLC-004, humidity, 65.0%")
	fmt.Println("   COMMIT ✓")

	fmt.Println("\n6. 数据轮询:")
	fmt.Println("   每5秒查询: SELECT * FROM sensor_data WHERE updated_at > ...")
	fmt.Println("   检测到新数据 → 发送更新通知")

	fmt.Println("\n数据库采集特性:")
	fmt.Println("  • 支持多种数据库: MySQL, PostgreSQL, SQL Server, Oracle, SQLite")
	fmt.Println("  • 连接池管理")
	fmt.Println("  • 事务支持")
	fmt.Println("  • 预编译查询")
	fmt.Println("  • 自动重连")
	fmt.Println("  • 定时轮询")

	fmt.Println("\n应用场景:")
	fmt.Println("  • MES/ERP系统数据采集")
	fmt.Println("  • 历史数据库查询")
	fmt.Println("  • 生产计划数据同步")
	fmt.Println("  • 质量数据追溯")
	fmt.Println("  • 设备维护记录查询")
}
