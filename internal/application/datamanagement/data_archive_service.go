package datamanagement

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dataManagementDomain "go_ProFiBus/internal/domain/datamanagement"
	"go_ProFiBus/pkg/interfaces"
)

// DataArchiveService 数据归档服务
type DataArchiveService struct {
	repo           interfaces.DataManagementRepository
	timeseriesRepo interfaces.Repository // 使用通用Repository接口
	archiveBaseDir string
}

// NewDataArchiveService 创建数据归档服务
func NewDataArchiveService(repo interfaces.DataManagementRepository, timeseriesRepo interfaces.Repository, archiveBaseDir string) *DataArchiveService {
	// 确保归档目录存在
	if archiveBaseDir == "" {
		archiveBaseDir = "archive"
	}
	_ = os.MkdirAll(archiveBaseDir, 0755)

	return &DataArchiveService{
		repo:           repo,
		timeseriesRepo: timeseriesRepo,
		archiveBaseDir: archiveBaseDir,
	}
}

// ExecuteArchive 执行归档
func (s *DataArchiveService) ExecuteArchive(ctx context.Context, policyID string) error {
	// 获取归档策略
	policy, err := s.repo.GetArchivePolicyByID(ctx, policyID)
	if err != nil {
		return fmt.Errorf("获取归档策略失败: %w", err)
	}

	if !policy.ShouldRun() {
		return nil // 未到执行时间
	}

	// 创建归档记录
	record := dataManagementDomain.NewArchiveRecord(
		fmt.Sprintf("archive_%d", time.Now().UnixNano()),
		policyID,
		policy.SourceType,
	)
	record.SourceID = policy.SourceID
	record.Status = dataManagementDomain.ArchiveStatusRunning

	if err := s.repo.CreateArchiveRecord(ctx, record); err != nil {
		return fmt.Errorf("创建归档记录失败: %w", err)
	}

	// 计算归档时间范围
	cutoffTime := time.Now().AddDate(0, 0, -policy.ArchiveAfterDays)
	startTime := cutoffTime.AddDate(0, 0, -policy.RetentionDays)

	// 执行归档
	archivePath, recordCount, archiveSize, err := s.archiveData(ctx, policy, startTime, cutoffTime)
	if err != nil {
		record.Fail(err.Error())
		_ = s.repo.UpdateArchiveRecord(ctx, record)
		return fmt.Errorf("归档数据失败: %w", err)
	}

	// 更新归档记录
	record.Complete(recordCount, archiveSize, archivePath)
	if err := s.repo.UpdateArchiveRecord(ctx, record); err != nil {
		return fmt.Errorf("更新归档记录失败: %w", err)
	}

	// 更新策略的下次执行时间
	policy.UpdateNextRun()
	if err := s.repo.UpdateArchivePolicy(ctx, policy); err != nil {
		return fmt.Errorf("更新归档策略失败: %w", err)
	}

	return nil
}

// archiveData 归档数据
func (s *DataArchiveService) archiveData(ctx context.Context, policy *dataManagementDomain.ArchivePolicy, startTime, endTime time.Time) (string, int64, int64, error) {
	// TODO: 实际归档逻辑
	// 1. 从timeseries表查询需要归档的数据
	// 2. 导出为文件（CSV、JSON等格式）
	// 3. 如果启用压缩，压缩文件
	// 4. 移动到归档位置
	// 5. 从原始表中删除已归档的数据

	// 生成归档文件路径
	archiveDir := filepath.Join(s.archiveBaseDir, policy.SourceType, policy.SourceID)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", 0, 0, fmt.Errorf("创建归档目录失败: %w", err)
	}

	filename := fmt.Sprintf("archive_%s_%d_%d.json",
		policy.SourceID,
		startTime.Unix(),
		endTime.Unix(),
	)
	archivePath := filepath.Join(archiveDir, filename)

	// 模拟归档（实际应该查询和导出数据）
	// 这里创建一个空的归档文件作为占位符
	file, err := os.Create(archivePath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("创建归档文件失败: %w", err)
	}
	file.Close()

	// 获取文件大小
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 如果启用压缩
	if policy.CompressionEnabled {
		// TODO: 实现压缩逻辑
		// compressedPath := archivePath + ".gz"
		// 压缩文件...
	}

	return archivePath, 0, fileInfo.Size(), nil
}

// RunScheduledArchives 运行定时归档任务
func (s *DataArchiveService) RunScheduledArchives(ctx context.Context) error {
	// 获取需要执行的归档策略
	policies, err := s.repo.GetPoliciesToRun(ctx)
	if err != nil {
		return fmt.Errorf("获取归档策略失败: %w", err)
	}

	// 执行每个策略
	for _, policy := range policies {
		if err := s.ExecuteArchive(ctx, policy.ID); err != nil {
			// 记录错误但继续处理其他策略
			fmt.Printf("归档策略 %s 执行失败: %v\n", policy.ID, err)
		}
	}

	return nil
}

// GetArchiveStats 获取归档统计
func (s *DataArchiveService) GetArchiveStats(ctx context.Context, policyID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	filters := interfaces.ArchiveRecordFilters{
		PolicyID:  &policyID,
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     1000,
	}

	records, err := s.repo.ListArchiveRecords(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("查询归档记录失败: %w", err)
	}

	totalRecords := int64(0)
	totalSize := int64(0)
	completedCount := 0
	failedCount := 0

	for _, record := range records {
		if record.Status == dataManagementDomain.ArchiveStatusCompleted {
			totalRecords += record.RecordCount
			totalSize += record.ArchiveSize
			completedCount++
		} else if record.Status == dataManagementDomain.ArchiveStatusFailed {
			failedCount++
		}
	}

	return map[string]interface{}{
		"total_records":   totalRecords,
		"total_size":      totalSize,
		"completed_count": completedCount,
		"failed_count":    failedCount,
		"record_count":    len(records),
	}, nil
}
