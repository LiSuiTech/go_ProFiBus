package workflow

import "fmt"

// ErrDuplicateNodeID 重复的节点ID错误
type ErrDuplicateNodeID struct {
	NodeID string
}

func (e ErrDuplicateNodeID) Error() string {
	return fmt.Sprintf("duplicate node ID: %s", e.NodeID)
}

// ErrNodeNotFound 节点未找到错误
type ErrNodeNotFound struct {
	NodeID string
}

func (e ErrNodeNotFound) Error() string {
	return fmt.Sprintf("node not found: %s", e.NodeID)
}

// ErrCycleDetected 检测到循环错误
type ErrCycleDetected struct{}

func (e ErrCycleDetected) Error() string {
	return "cycle detected in workflow graph"
}

// ErrInvalidNodeType 无效的节点类型错误
type ErrInvalidNodeType struct {
	NodeType NodeType
}

func (e ErrInvalidNodeType) Error() string {
	return fmt.Sprintf("invalid node type: %s", e.NodeType)
}

// ErrExecutionNotFound 执行实例未找到错误
type ErrExecutionNotFound struct {
	ExecutionID string
}

func (e ErrExecutionNotFound) Error() string {
	return fmt.Sprintf("execution not found: %s", e.ExecutionID)
}

// ErrWorkflowNotFound 工作流未找到错误
type ErrWorkflowNotFound struct {
	WorkflowID string
}

func (e ErrWorkflowNotFound) Error() string {
	return fmt.Sprintf("workflow not found: %s", e.WorkflowID)
}

// ErrWorkflowRunning 工作流正在运行错误
type ErrWorkflowRunning struct {
	WorkflowID string
}

func (e ErrWorkflowRunning) Error() string {
	return fmt.Sprintf("workflow is currently running: %s", e.WorkflowID)
}
