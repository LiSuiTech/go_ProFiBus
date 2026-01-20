package storage

import (
	"context"
	"fmt"
	"go_ProFiBus/logger"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore PostgreSQL数据库存储
type PostgresStore struct {
	pool   *pgxpool.Pool
	ctx    context.Context
	cancel context.CancelFunc
	log    *logger.Logger
	mu     sync.RWMutex
	stats  *StoreStats
}

// StoreStats 存储统计信息
type StoreStats struct {
	TotalConnections    int64
	ActiveConnections   int32
	IdleConnections     int32
	TotalQueries        uint64
	FailedQueries       uint64
	LastQueryTime       time.Time
	mu                  sync.RWMutex
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
}

// GetDefaultPostgresConfig 获取默认配置
func GetDefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		Database:        "profibus",
		User:            "postgres",
		Password:        "",
		MaxConnections:  10,
		MinConnections:  2,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// NewPostgresStore 创建PostgreSQL存储
func NewPostgresStore(config *PostgresConfig) (*PostgresStore, error) {
	if config == nil {
		config = GetDefaultPostgresConfig()
	}

	// 构建连接字符串
	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		config.Host, config.Port, config.Database, config.User, config.Password,
	)

	// 配置连接池
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("解析连接配置失败: %w", err)
	}

	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MinConns = config.MinConnections
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	// 创建连接池
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建连接池失败: %w", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		cancel()
		pool.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	ps := &PostgresStore{
		pool:   pool,
		ctx:    ctx,
		cancel: cancel,
		log:    logger.GetLogger(),
		stats:  &StoreStats{},
	}

	ps.log.Info("PostgreSQL连接池已创建 (最大连接: %d, 最小连接: %d)",
		config.MaxConnections, config.MinConnections)

	return ps, nil
}

// Close 关闭连接池
func (ps *PostgresStore) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.cancel()
	ps.pool.Close()

	ps.log.Info("PostgreSQL连接池已关闭")
	return nil
}

// Ping 测试连接
func (ps *PostgresStore) Ping() error {
	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Second)
	defer cancel()

	return ps.pool.Ping(ctx)
}

// WithTx 在事务中执行函数
func (ps *PostgresStore) WithTx(fn func(pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ps.ctx, 30*time.Second)
	defer cancel()

	tx, err := ps.pool.Begin(ctx)
	if err != nil {
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			ps.log.Error("回滚事务失败: %v", rbErr)
		}
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return fmt.Errorf("提交事务失败: %w", err)
	}

	ps.stats.mu.Lock()
	ps.stats.TotalQueries++
	ps.stats.LastQueryTime = time.Now()
	ps.stats.mu.Unlock()

	return nil
}

// Query 执行查询
func (ps *PostgresStore) Query(query string, args ...interface{}) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ps.ctx, 30*time.Second)
	defer cancel()

	rows, err := ps.pool.Query(ctx, query, args...)
	if err != nil {
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	ps.stats.mu.Lock()
	ps.stats.TotalQueries++
	ps.stats.LastQueryTime = time.Now()
	ps.stats.mu.Unlock()

	return rows, nil
}

// QueryRow 执行查询单行
func (ps *PostgresStore) QueryRow(query string, args ...interface{}) pgx.Row {
	ctx, cancel := context.WithTimeout(ps.ctx, 30*time.Second)
	defer cancel()

	ps.stats.mu.Lock()
	ps.stats.TotalQueries++
	ps.stats.LastQueryTime = time.Now()
	ps.stats.mu.Unlock()

	return ps.pool.QueryRow(ctx, query, args...)
}

// Exec 执行SQL语句
func (ps *PostgresStore) Exec(query string, args ...interface{}) (pgx.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ps.ctx, 30*time.Second)
	defer cancel()

	tag, err := ps.pool.Exec(ctx, query, args...)
	if err != nil {
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return tag, fmt.Errorf("执行失败: %w", err)
	}

	ps.stats.mu.Lock()
	ps.stats.TotalQueries++
	ps.stats.LastQueryTime = time.Now()
	ps.stats.mu.Unlock()

	return tag, nil
}

// CopyFrom 批量复制数据
func (ps *PostgresStore) CopyFrom(
	tableName pgx.Identifier,
	columnNames []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Minute)
	defer cancel()

	count, err := ps.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
	if err != nil {
		ps.stats.mu.Lock()
		ps.stats.FailedQueries++
		ps.stats.mu.Unlock()
		return 0, fmt.Errorf("批量复制失败: %w", err)
	}

	ps.stats.mu.Lock()
	ps.stats.TotalQueries++
	ps.stats.LastQueryTime = time.Now()
	ps.stats.mu.Unlock()

	ps.log.Debug("批量复制成功: %d 行到表 %v", count, tableName)
	return count, nil
}

// GetStats 获取统计信息
func (ps *PostgresStore) GetStats() *StoreStats {
	ps.stats.mu.RLock()
	defer ps.stats.mu.RUnlock()

	poolStats := ps.pool.Stat()

	return &StoreStats{
		TotalConnections:  poolStats.TotalConns(),
		ActiveConnections: poolStats.AcquiredConns(),
		IdleConnections:   poolStats.IdleConns(),
		TotalQueries:      ps.stats.TotalQueries,
		FailedQueries:     ps.stats.FailedQueries,
		LastQueryTime:     ps.stats.LastQueryTime,
	}
}

// GetPool 获取连接池（用于高级操作）
func (ps *PostgresStore) GetPool() *pgxpool.Pool {
	return ps.pool
}

// GetContext 获取context
func (ps *PostgresStore) GetContext() context.Context {
	return ps.ctx
}

// IsHealthy 检查健康状态
func (ps *PostgresStore) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(ps.ctx, 3*time.Second)
	defer cancel()

	err := ps.pool.Ping(ctx)
	return err == nil
}

// WaitForConnection 等待数据库连接可用
func (ps *PostgresStore) WaitForConnection(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ps.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ps.pool.Ping(ctx); err == nil {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("等待数据库连接超时")
		}
	}
}
