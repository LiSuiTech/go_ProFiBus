package serial

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// Database 数据库数据源实现
// 支持通过SQL查询采集数据
type Database struct {
	config      *DatabaseConfig
	db          *sql.DB
	monitoring  bool
	dataChan    chan *interfaces.DataUpdate
	stopChan    chan struct{}
	mu          sync.RWMutex
	stats       *interfaces.ProtocolStatus
	queryCache  map[string]*sql.Stmt // 预编译查询缓存
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// 数据库类型
	Driver string // mysql, postgres, sqlserver, oracle, sqlite

	// 连接信息
	Host     string // 主机地址
	Port     int    // 端口
	Database string // 数据库名
	Username string // 用户名
	Password string // 密码

	// 连接字符串（优先使用）
	DSN string // Data Source Name

	// 连接池配置
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大生存时间

	// 查询配置
	DefaultTable  string        // 默认表名
	TimeColumn    string        // 时间列名
	PollingQuery  string        // 轮询查询SQL
	PollingInterval time.Duration // 轮询间隔

	// SSL/TLS配置
	SSLMode    string // disable/require/verify-ca/verify-full
	SSLCert    string // SSL证书路径
	SSLKey     string // SSL密钥路径
	SSLRootCert string // SSL根证书路径

	Timeout time.Duration
}

// Validate 验证配置
func (c *DatabaseConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}

	if c.DSN == "" {
		// 如果没有提供DSN，检查其他参数
		if c.Host == "" {
			return fmt.Errorf("host or DSN is required")
		}
		if c.Database == "" {
			return fmt.Errorf("database name is required")
		}
	}

	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 10
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 1 * time.Hour
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.PollingInterval == 0 {
		c.PollingInterval = 5 * time.Second
	}

	return nil
}

// GetType 获取配置类型
func (c *DatabaseConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolDatabase
}

// NewDatabase 创建数据库数据源
func NewDatabase(config *DatabaseConfig) (*Database, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Database{
		config:     config,
		dataChan:   make(chan *interfaces.DataUpdate, 1000),
		stopChan:   make(chan struct{}),
		queryCache: make(map[string]*sql.Stmt),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开数据库连接
func (d *Database) Open(devicePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 构建DSN
	dsn := d.config.DSN
	if dsn == "" {
		dsn = d.buildDSN()
	}

	// 打开数据库连接
	db, err := sql.Open(d.config.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(d.config.MaxOpenConns)
	db.SetMaxIdleConns(d.config.MaxIdleConns)
	db.SetConnMaxLifetime(d.config.ConnMaxLifetime)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	d.db = db
	d.stats.Connected = true

	return nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.monitoring {
		close(d.stopChan)
		d.monitoring = false
	}

	// 关闭预编译语句
	for _, stmt := range d.queryCache {
		stmt.Close()
	}
	d.queryCache = make(map[string]*sql.Stmt)

	if d.db != nil {
		err := d.db.Close()
		d.db = nil
		d.stats.Connected = false
		return err
	}

	return nil
}

// buildDSN 构建数据源名称
func (d *Database) buildDSN() string {
	cfg := d.config

	switch cfg.Driver {
	case "mysql":
		// MySQL: user:password@tcp(host:port)/database?params
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

		// 添加参数
		params := "?parseTime=true&charset=utf8mb4"
		if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
			params += "&tls=true"
		}
		return dsn + params

	case "postgres", "postgresql":
		// PostgreSQL: postgres://user:password@host:port/database?sslmode=disable
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

		sslMode := cfg.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("%s?sslmode=%s", dsn, sslMode)

	case "sqlserver", "mssql":
		// SQL Server: sqlserver://user:password@host:port?database=dbname
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	case "sqlite", "sqlite3":
		// SQLite: file:path/to/database.db
		return cfg.Database

	default:
		return ""
	}
}

// Query 执行SQL查询 - 实现数据读取功能
func (d *Database) Query(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("database not connected")
	}
	d.mu.RUnlock()

	// 执行查询
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		d.stats.ErrorCount++
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	d.stats.RequestCount++

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// 读取结果
	var results []map[string]interface{}

	for rows.Next() {
		// 创建值容器
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// 构建结果map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// 类型转换
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v
			default:
				row[col] = v
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// Execute 执行SQL语句（INSERT/UPDATE/DELETE）- 实现数据写入功能
func (d *Database) Execute(ctx context.Context, query string, args ...interface{}) (int64, error) {
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return 0, fmt.Errorf("database not connected")
	}
	d.mu.RUnlock()

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		d.stats.ErrorCount++
		return 0, fmt.Errorf("execute failed: %w", err)
	}

	d.stats.RequestCount++

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// StartPolling 启动数据轮询 - 实现数据监听功能
func (d *Database) StartPolling(ctx context.Context) error {
	d.mu.Lock()
	if d.monitoring {
		d.mu.Unlock()
		return fmt.Errorf("polling already started")
	}

	if d.config.PollingQuery == "" {
		d.mu.Unlock()
		return fmt.Errorf("polling query not configured")
	}

	d.monitoring = true
	d.stats.Running = true
	d.mu.Unlock()

	go d.pollingLoop(ctx)

	return nil
}

// StopPolling 停止数据轮询
func (d *Database) StopPolling() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.monitoring {
		return nil
	}

	d.stopChan <- struct{}{}
	d.monitoring = false
	d.stats.Running = false

	return nil
}

// pollingLoop 轮询循环
func (d *Database) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(d.config.PollingInterval)
	defer ticker.Stop()

	var lastResults []map[string]interface{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			results, err := d.Query(ctx, d.config.PollingQuery)
			if err != nil {
				continue
			}

			// 检测变化
			if d.detectChanges(lastResults, results) {
				// 发送更新
				for _, row := range results {
					update := &interfaces.DataUpdate{
						TargetID:  d.config.DefaultTable,
						Value:     row,
						Quality:   1.0,
						Timestamp: time.Now(),
						ChangeType: "value",
						Metadata: map[string]interface{}{
							"protocol": "database",
							"driver":   d.config.Driver,
							"table":    d.config.DefaultTable,
						},
					}

					select {
					case d.dataChan <- update:
					default:
						// 缓冲区满，丢弃
					}
				}
			}

			lastResults = results
		}
	}
}

// detectChanges 检测数据变化
func (d *Database) detectChanges(old, new []map[string]interface{}) bool {
	if len(old) != len(new) {
		return true
	}

	// 简化：认为有新数据就是变化
	// 实际应该比较具体字段
	return len(new) > 0
}

// GetDataChannel 获取数据通道
func (d *Database) GetDataChannel() <-chan *interfaces.DataUpdate {
	return d.dataChan
}

// GetStatus 获取协议状态
func (d *Database) GetStatus() *interfaces.ProtocolStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := *d.stats
	return &status
}

// Transaction 执行事务
func (d *Database) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return fmt.Errorf("database not connected")
	}
	db := d.db
	d.mu.RUnlock()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// PrepareQuery 预编译查询
func (d *Database) PrepareQuery(name string, query string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database not connected")
	}

	stmt, err := d.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}

	// 如果已存在，关闭旧的
	if oldStmt, exists := d.queryCache[name]; exists {
		oldStmt.Close()
	}

	d.queryCache[name] = stmt
	return nil
}

// ExecutePrepared 执行预编译查询
func (d *Database) ExecutePrepared(ctx context.Context, name string, args ...interface{}) ([]map[string]interface{}, error) {
	d.mu.RLock()
	stmt, exists := d.queryCache[name]
	d.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("prepared query '%s' not found", name)
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// 解析结果（与Query方法类似）
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v
			default:
				row[col] = v
			}
		}

		results = append(results, row)
	}

	return results, rows.Err()
}

// GetTableSchema 获取表结构
func (d *Database) GetTableSchema(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	var query string

	switch d.config.Driver {
	case "mysql":
		query = fmt.Sprintf("DESCRIBE %s", tableName)
	case "postgres", "postgresql":
		query = fmt.Sprintf(`
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_name = '%s'`, tableName)
	case "sqlserver", "mssql":
		query = fmt.Sprintf(`
			SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_NAME = '%s'`, tableName)
	default:
		return nil, fmt.Errorf("unsupported driver for schema query: %s", d.config.Driver)
	}

	return d.Query(ctx, query)
}

// GetTables 获取所有表名
func (d *Database) GetTables(ctx context.Context) ([]string, error) {
	var query string

	switch d.config.Driver {
	case "mysql":
		query = fmt.Sprintf("SHOW TABLES FROM %s", d.config.Database)
	case "postgres", "postgresql":
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	case "sqlserver", "mssql":
		query = "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE'"
	case "sqlite", "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table'"
	default:
		return nil, fmt.Errorf("unsupported driver: %s", d.config.Driver)
	}

	results, err := d.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var tables []string
	for _, row := range results {
		for _, v := range row {
			if tableName, ok := v.(string); ok {
				tables = append(tables, tableName)
			}
		}
	}

	return tables, nil
}

// Read 实现io.Reader接口
func (d *Database) Read(buffer []byte) (int, error) {
	return 0, fmt.Errorf("Read not supported for database")
}

// Write 实现io.Writer接口
func (d *Database) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("Write not supported for database")
}
