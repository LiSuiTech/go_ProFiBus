package control

import (
	"context"
	"fmt"
	"time"

	controlDomain "go_ProFiBus/internal/domain/control"
	"go_ProFiBus/pkg/interfaces"
)

// ControlService 设备控制服务
type ControlService struct {
	repo            interfaces.ControlRepository
	actuatorFactory func(deviceID string) (interfaces.Actuator, error)
}

// NewControlService 创建设备控制服务
func NewControlService(repo interfaces.ControlRepository, actuatorFactory func(deviceID string) (interfaces.Actuator, error)) *ControlService {
	return &ControlService{
		repo:            repo,
		actuatorFactory: actuatorFactory,
	}
}

// ExecuteControlAction 执行控制动作
func (s *ControlService) ExecuteControlAction(ctx context.Context, action *controlDomain.ControlAction, userID, userName, ipAddress, userAgent string) error {
	// 检查权限
	if err := s.checkPermission(ctx, userID, action); err != nil {
		return fmt.Errorf("权限检查失败: %w", err)
	}

	// 如果需要确认但未确认
	if action.RequireConfirmation && action.ConfirmedAt == nil {
		return fmt.Errorf("控制动作需要确认")
	}

	// 创建审计日志
	auditLog := controlDomain.NewAuditLog(
		fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		action.ID,
		controlDomain.AuditEventCreated,
	)
	auditLog.UserID = userID
	auditLog.UserName = userName
	auditLog.IPAddress = ipAddress
	auditLog.UserAgent = userAgent
	auditLog.Details["action_type"] = string(action.ActionType)
	auditLog.Details["device_id"] = action.DeviceID
	_ = s.repo.CreateAuditLog(ctx, auditLog)

	// 保存控制动作
	if err := s.repo.CreateControlAction(ctx, action); err != nil {
		return fmt.Errorf("创建控制动作失败: %w", err)
	}

	// 需要确认的已在前面返回错误，此处直接执行
	return s.executeAction(ctx, action, userID, userName, ipAddress, userAgent)
}

// ConfirmControlAction 确认控制动作
func (s *ControlService) ConfirmControlAction(ctx context.Context, actionID, userID, userName, ipAddress, userAgent string) error {
	// 获取控制动作
	action, err := s.repo.GetControlActionByID(ctx, actionID)
	if err != nil {
		return fmt.Errorf("获取控制动作失败: %w", err)
	}

	if !action.IsPending() {
		return fmt.Errorf("控制动作已执行或已取消")
	}

	// 确认动作
	action.Confirm(userID)

	// 更新动作
	if err := s.repo.UpdateControlAction(ctx, action); err != nil {
		return fmt.Errorf("更新控制动作失败: %w", err)
	}

	// 创建审计日志
	auditLog := controlDomain.NewAuditLog(
		fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		action.ID,
		controlDomain.AuditEventConfirmed,
	)
	auditLog.UserID = userID
	auditLog.UserName = userName
	auditLog.IPAddress = ipAddress
	auditLog.UserAgent = userAgent
	_ = s.repo.CreateAuditLog(ctx, auditLog)

	// 执行控制动作
	return s.executeAction(ctx, action, userID, userName, ipAddress, userAgent)
}

// executeAction 执行控制动作
func (s *ControlService) executeAction(ctx context.Context, action *controlDomain.ControlAction, userID, userName, ipAddress, userAgent string) error {
	// 开始执行
	action.StartExecution(userID)
	if err := s.repo.UpdateControlAction(ctx, action); err != nil {
		return fmt.Errorf("更新控制动作失败: %w", err)
	}

	// 创建审计日志
	auditLog := controlDomain.NewAuditLog(
		fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		action.ID,
		controlDomain.AuditEventExecuted,
	)
	auditLog.UserID = userID
	auditLog.UserName = userName
	auditLog.IPAddress = ipAddress
	auditLog.UserAgent = userAgent
	_ = s.repo.CreateAuditLog(ctx, auditLog)

	// 获取执行器
	actuator, err := s.actuatorFactory(action.DeviceID)
	if err != nil {
		action.Fail(fmt.Sprintf("获取执行器失败: %v", err))
		_ = s.repo.UpdateControlAction(ctx, action)
		return err
	}

	// 转换为interfaces.ControlAction
	controlAction := &interfaces.ControlAction{
		ActionType:          interfaces.ActionType(action.ActionType),
		TargetDevice:        action.DeviceID,
		Parameters:          action.Parameters,
		Reason:              action.Reason,
		Severity:            action.Severity,
		RequireConfirmation: action.RequireConfirmation,
		Timeout:             30 * time.Second,
		Metadata:            action.Metadata,
	}

	// 执行控制
	startTime := time.Now()
	result, err := actuator.Execute(ctx, controlAction)
	duration := time.Since(startTime)

	// 更新动作状态
	if err != nil {
		action.Fail(err.Error())
	} else {
		if result.Success {
			resultMap := make(map[string]interface{})
			resultMap["success"] = result.Success
			resultMap["message"] = result.Message
			if result.Response != nil {
				resultMap["response"] = result.Response
			}
			action.Complete(resultMap, int(duration.Milliseconds()))
		} else {
			action.Fail(result.Message)
		}
	}

	if err := s.repo.UpdateControlAction(ctx, action); err != nil {
		return fmt.Errorf("更新控制动作失败: %w", err)
	}

	// 创建完成审计日志
	eventType := controlDomain.AuditEventCompleted
	if !action.IsCompleted() {
		eventType = controlDomain.AuditEventFailed
	}

	auditLog = controlDomain.NewAuditLog(
		fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		action.ID,
		eventType,
	)
	auditLog.UserID = userID
	auditLog.UserName = userName
	auditLog.IPAddress = ipAddress
	auditLog.UserAgent = userAgent
	auditLog.Details["duration_ms"] = action.DurationMs
	if action.ErrorMessage != "" {
		auditLog.Details["error"] = action.ErrorMessage
	}
	_ = s.repo.CreateAuditLog(ctx, auditLog)

	// 更新策略执行计数
	if action.PolicyID != "" {
		policy, err := s.repo.GetControlPolicyByID(ctx, action.PolicyID)
		if err == nil {
			policy.RecordExecution()
			_ = s.repo.UpdateControlPolicy(ctx, policy)
		}
	}

	return nil
}

// checkPermission 检查权限
func (s *ControlService) checkPermission(ctx context.Context, userID string, action *controlDomain.ControlAction) error {
	// 获取用户权限（查不到或出错时拒绝，fail-closed）
	permission, err := s.repo.GetControlPermission(ctx, userID, action.ActionType)
	if err != nil {
		return fmt.Errorf("无法获取控制权限: %w", err)
	}

	if !permission.Enabled {
		return fmt.Errorf("用户没有执行此动作的权限")
	}

	// 检查设备权限
	if !permission.CanControlDevice(action.DeviceID) {
		return fmt.Errorf("用户没有控制设备 %s 的权限", action.DeviceID)
	}

	// 检查严重程度权限
	if action.Severity > permission.MaxSeverity {
		return fmt.Errorf("动作严重程度 %d 超过允许的最大值 %d", action.Severity, permission.MaxSeverity)
	}

	// 检查时间权限
	if !permission.CanControlAtTime(time.Now()) {
		return fmt.Errorf("当前时间不在允许的执行时间范围内")
	}

	// 检查是否需要确认
	if permission.RequireConfirmation {
		action.RequireConfirmation = true
	}

	return nil
}

// EvaluateAndExecute 评估分析结果并执行控制
func (s *ControlService) EvaluateAndExecute(ctx context.Context, deviceID string, analysisResults []interfaces.AnalysisResult, userID, userName string) ([]*controlDomain.ControlAction, error) {
	// 获取启用的控制策略
	filters := interfaces.ControlPolicyFilters{
		Enabled: func() *bool { b := true; return &b }(),
		Limit:   100,
	}
	policies, err := s.repo.ListControlPolicies(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("获取控制策略失败: %w", err)
	}

	// 按优先级排序
	sortedPolicies := s.sortPoliciesByPriority(policies)

	executedActions := make([]*controlDomain.ControlAction, 0)

	// 评估每个策略
	for _, policy := range sortedPolicies {
		if !policy.CanExecute() {
			continue
		}

		// 评估条件
		if s.evaluateCondition(policy.ConditionConfig, analysisResults) {
			// 创建控制动作
			actionConfig := policy.ActionConfig
			actionType, _ := actionConfig["action_type"].(string)
			if actionType == "" {
				continue
			}

			action := controlDomain.NewControlAction(
				fmt.Sprintf("action_%d", time.Now().UnixNano()),
				deviceID,
				controlDomain.ActionType(actionType),
			)
			action.PolicyID = policy.ID
			action.Reason = fmt.Sprintf("策略 %s 触发", policy.Name)

			// 设置参数
			if params, ok := actionConfig["parameters"].(map[string]interface{}); ok {
				action.Parameters = params
			}

			// 设置严重程度
			if severity, ok := actionConfig["severity"].(float64); ok {
				action.Severity = int(severity)
			}

			// 执行动作
			if err := s.ExecuteControlAction(ctx, action, userID, userName, "", ""); err != nil {
				// 记录错误但继续处理其他策略
				continue
			}

			executedActions = append(executedActions, action)
		}
	}

	return executedActions, nil
}

// evaluateCondition 评估条件
func (s *ControlService) evaluateCondition(conditionConfig map[string]interface{}, results []interfaces.AnalysisResult) bool {
	// TODO: 实现条件评估逻辑
	// 这里简化处理，实际应该根据conditionConfig中的规则进行评估
	// 例如：检查分析结果中是否有异常、分数是否超过阈值等

	// 简化实现：如果有任何结果，返回true
	return len(results) > 0
}

// sortPoliciesByPriority 按优先级排序策略
func (s *ControlService) sortPoliciesByPriority(policies []*controlDomain.ControlPolicy) []*controlDomain.ControlPolicy {
	sorted := make([]*controlDomain.ControlPolicy, len(policies))
	copy(sorted, policies)

	// 冒泡排序（按优先级降序）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Priority < sorted[j+1].Priority {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}
