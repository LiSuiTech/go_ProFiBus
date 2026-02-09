package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/logger"
)

// Engine 工作流执行引擎
type Engine struct {
	repository   WorkflowRepository
	executors    map[NodeType]NodeExecutor
	executions   map[string]*WorkflowExecution
	mu           sync.RWMutex
	log          *logger.Logger
}

// NewEngine 创建工作流引擎
func NewEngine(repository WorkflowRepository) *Engine {
	return &Engine{
		repository: repository,
		executors:  make(map[NodeType]NodeExecutor),
		executions: make(map[string]*WorkflowExecution),
		log:        logger.GetLogger(),
	}
}

// RegisterExecutor 注册节点执行器
func (e *Engine) RegisterExecutor(executor NodeExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executors[executor.GetNodeType()] = executor
}

// Execute 执行工作流
func (e *Engine) Execute(ctx context.Context, workflowID string, inputs map[string]interface{}) (string, error) {
	// 获取工作流定义
	workflow, err := e.repository.GetByID(ctx, workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow: %w", err)
	}

	// 验证工作流
	if err := workflow.Validate(); err != nil {
		return "", fmt.Errorf("workflow validation failed: %w", err)
	}

	// 创建执行实例
	executionID := fmt.Sprintf("exec_%d", time.Now().UnixNano())
	execution := &WorkflowExecution{
		ID:           executionID,
		WorkflowID:   workflowID,
		Status:       ExecutionStatusPending,
		NodeStatuses: make(map[string]NodeStatus),
		Variables:    make(map[string]interface{}),
	}

	// 初始化变量
	for _, variable := range workflow.Variables {
		execution.Variables[variable.Name] = variable.Value
	}

	// 合并输入变量
	for k, v := range inputs {
		execution.Variables[k] = v
	}

	// 初始化节点状态
	for _, node := range workflow.Nodes {
		execution.NodeStatuses[node.ID] = NodeStatus{
			NodeID: node.ID,
			Status: "pending",
		}
	}

	// 保存执行实例
	if err := e.repository.SaveExecution(ctx, execution); err != nil {
		return "", fmt.Errorf("failed to save execution: %w", err)
	}

	// 存储执行实例
	e.mu.Lock()
	e.executions[executionID] = execution
	e.mu.Unlock()

	// 异步执行
	go e.runExecution(ctx, workflow, execution)

	return executionID, nil
}

// runExecution 运行执行实例
func (e *Engine) runExecution(ctx context.Context, workflow *Workflow, execution *WorkflowExecution) {
	defer func() {
		now := time.Now()
		execution.CompletedAt = &now
		if execution.Status == ExecutionStatusRunning {
			execution.Status = ExecutionStatusCompleted
		}
		e.repository.SaveExecution(ctx, execution)
	}()

	execution.Status = ExecutionStatusRunning
	now := time.Now()
	execution.StartedAt = &now
	e.repository.SaveExecution(ctx, execution)

	// 获取起始节点
	startNodes := workflow.GetStartNodes()
	if len(startNodes) == 0 {
		execution.Status = ExecutionStatusFailed
		execution.Error = "no start nodes found"
		return
	}

	// 执行拓扑排序
	topologicalOrder := e.topologicalSort(workflow)
	if len(topologicalOrder) == 0 {
		execution.Status = ExecutionStatusFailed
		execution.Error = "failed to perform topological sort"
		return
	}

	// 按拓扑顺序执行节点
	nodeOutputs := make(map[string]map[string]interface{})
	for _, nodeID := range topologicalOrder {
		node, exists := workflow.GetNodeByID(nodeID)
		if !exists {
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("node not found: %s", nodeID)
			return
		}

		// 更新节点状态为运行中
		nodeStatus := execution.NodeStatuses[nodeID]
		nodeStatus.Status = "running"
		now := time.Now()
		nodeStatus.StartedAt = &now
		execution.NodeStatuses[nodeID] = nodeStatus
		e.repository.SaveExecution(ctx, execution)

		// 收集节点输入
		inputs := make(map[string]interface{})
		for _, edge := range workflow.Edges {
			if edge.Target == nodeID {
				if output, ok := nodeOutputs[edge.Source]; ok {
					// 获取源节点和目标节点
					sourceNode, sourceExists := workflow.GetNodeByID(edge.Source)
					targetNode, targetExists := workflow.GetNodeByID(edge.Target)
					
					if !sourceExists || !targetExists {
						continue
					}
					
					// 获取源端口和目标端口
					var sourcePort *OutputPort
					var targetPort *InputPort
					
					if edge.SourcePort != "" {
						for i := range sourceNode.Outputs {
							if sourceNode.Outputs[i].ID == edge.SourcePort {
								sourcePort = &sourceNode.Outputs[i]
								break
							}
						}
					}
					
					if edge.TargetPort != "" {
						for i := range targetNode.Inputs {
							if targetNode.Inputs[i].ID == edge.TargetPort {
								targetPort = &targetNode.Inputs[i]
								break
							}
						}
					}
					
					// 根据参数映射传递数据
					if sourcePort != nil && targetPort != nil {
						// 优先使用参数映射
						if len(edge.ParamMapping) > 0 {
							for targetParam, sourceParam := range edge.ParamMapping {
								// 从源节点的输出中查找参数
								if sourcePort.ParamName != "" {
									// 如果源端口有参数名，使用参数名查找
									if data, ok := output[sourcePort.ParamName]; ok {
										inputs[targetParam] = data
									} else if data, ok := output[sourcePort.ID]; ok {
										inputs[targetParam] = data
									}
								} else if data, ok := output[sourceParam]; ok {
									inputs[targetParam] = data
								}
							}
						} else {
							// 如果没有参数映射，使用端口参数名
							if sourcePort.ParamName != "" && targetPort.ParamName != "" {
								if data, ok := output[sourcePort.ParamName]; ok {
									inputs[targetPort.ParamName] = data
								} else if data, ok := output[sourcePort.ID]; ok {
									inputs[targetPort.ParamName] = data
								}
							} else {
								// 回退到端口ID映射
								if data, ok := output[edge.SourcePort]; ok {
									inputs[edge.TargetPort] = data
								}
							}
						}
					} else {
						// 如果没有指定端口，传递所有输出
						for k, v := range output {
							inputs[k] = v
						}
					}
				}
			}
		}
		
		// 为未连接的输入端口设置默认值
		if targetNode, exists := workflow.GetNodeByID(nodeID); exists {
			for _, inputPort := range targetNode.Inputs {
				if _, ok := inputs[inputPort.ParamName]; !ok && inputPort.DefaultValue != nil {
					inputs[inputPort.ParamName] = inputPort.DefaultValue
				}
			}
		}

		// 获取执行器
		executor, exists := e.executors[node.Type]
		if !exists {
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("executor not found for node type: %s", node.Type)
			return
		}

		// 执行节点
		output, err := executor.Execute(ctx, node, inputs, execution.Variables)
		if err != nil {
			nodeStatus.Status = "failed"
			nodeStatus.Error = err.Error()
			execution.NodeStatuses[nodeID] = nodeStatus
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("node %s failed: %v", nodeID, err)
			return
		}

		// 更新节点状态
		nodeStatus.Status = "completed"
		now = time.Now()
		nodeStatus.CompletedAt = &now
		nodeStatus.Output = output
		execution.NodeStatuses[nodeID] = nodeStatus
		nodeOutputs[nodeID] = output

		// 更新变量（如果节点是变量设置节点）
		if node.Type == NodeTypeVariableSet {
			if varName, ok := node.Config["variable_name"].(string); ok {
				if varValue, ok := output["value"]; ok {
					execution.Variables[varName] = varValue
				}
			}
		}

		// 保存执行状态
		e.repository.SaveExecution(ctx, execution)
	}

	execution.Status = ExecutionStatusCompleted
}

// topologicalSort 拓扑排序
func (e *Engine) topologicalSort(workflow *Workflow) []string {
	// 构建邻接表和入度
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, node := range workflow.Nodes {
		inDegree[node.ID] = 0
		adj[node.ID] = []string{}
	}

	for _, edge := range workflow.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// 找到所有入度为0的节点
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		// 减少相邻节点的入度
		for _, neighbor := range adj[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return result
}

// GetExecution 获取执行实例
func (e *Engine) GetExecution(ctx context.Context, executionID string) (*WorkflowExecution, error) {
	e.mu.RLock()
	execution, exists := e.executions[executionID]
	e.mu.RUnlock()

	if exists {
		return execution, nil
	}

	// 从仓库加载
	return e.repository.GetExecution(ctx, executionID)
}

// CancelExecution 取消执行
func (e *Engine) CancelExecution(ctx context.Context, executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	execution, exists := e.executions[executionID]
	if !exists {
		exec, err := e.repository.GetExecution(ctx, executionID)
		if err != nil {
			return err
		}
		execution = exec
	}

	if execution.Status != ExecutionStatusRunning {
		return fmt.Errorf("execution is not running")
	}

	execution.Status = ExecutionStatusCancelled
	now := time.Now()
	execution.CompletedAt = &now
	return e.repository.SaveExecution(ctx, execution)
}

// ListExecutions 列出执行实例
func (e *Engine) ListExecutions(ctx context.Context, workflowID string) ([]*WorkflowExecution, error) {
	return e.repository.ListExecutions(ctx, workflowID)
}
