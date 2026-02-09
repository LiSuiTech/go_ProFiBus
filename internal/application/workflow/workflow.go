package workflow

import (
	"context"
	"encoding/json"
	"time"
)

// Workflow 工作流定义
// 类似 Dify 的工作流系统，支持 DAG（有向无环图）结构
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Nodes       []Node    `json:"nodes"`
	Edges       []Edge    `json:"edges"`
	Variables   []Variable `json:"variables"`
	Status      string    `json:"status"` // draft, running, stopped, error
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

// Node 工作流节点
type Node struct {
	ID          string                 `json:"id"`
	Type        NodeType               `json:"type"`
	Name        string                 `json:"name"`
	Config      map[string]interface{} `json:"config"`
	Position    Position               `json:"position"` // 节点在画布上的位置
	Inputs      []InputPort            `json:"inputs"`
	Outputs     []OutputPort           `json:"outputs"`
}

// NodeType 节点类型
type NodeType string

const (
	NodeTypeDataSource    NodeType = "data_source"     // 数据采集节点
	NodeTypeDeviceSource  NodeType = "device_source"   // 设备数据采集节点
	NodeTypeRuleDetection NodeType = "rule_detection"  // 规则检测节点
	NodeTypeMLAnalysis    NodeType = "ml_analysis"     // ML 分析节点
	NodeTypeCondition     NodeType = "condition"       // 条件分支节点
	NodeTypeLoop          NodeType = "loop"            // 循环节点
	NodeTypeVariableSet   NodeType = "variable_set"    // 变量设置节点
	NodeTypeOutput        NodeType = "output"          // 输出节点
	NodeTypeAlertOutput   NodeType = "alert_output"    // 告警输出节点
	NodeTypeDeviceControl NodeType = "device_control"   // 设备控制节点
	NodeTypeTransform     NodeType = "transform"       // 数据转换节点
	NodeTypeFilter        NodeType = "filter"          // 数据过滤节点
)

// Position 节点位置
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// InputPort 输入端口
type InputPort struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"`        // data, condition, etc.
	ParamName   string `json:"param_name"`  // 参数名称，用于节点内部引用
	DataType    string `json:"data_type"`   // 数据类型：string, number, boolean, object, array
	Required    bool   `json:"required"`    // 是否必需
	Description string `json:"description"` // 参数描述
	DefaultValue interface{} `json:"default_value,omitempty"` // 默认值
}

// OutputPort 输出端口
type OutputPort struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	ParamName   string `json:"param_name"`  // 参数名称，用于节点输出
	DataType    string `json:"data_type"`   // 数据类型：string, number, boolean, object, array
	Description string `json:"description"` // 参数描述
}

// Edge 工作流边（连接）
type Edge struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`       // 源节点ID
	Target      string            `json:"target"`       // 目标节点ID
	SourcePort  string            `json:"source_port"`  // 源端口ID
	TargetPort  string            `json:"target_port"`  // 目标端口ID
	Condition   string            `json:"condition,omitempty"` // 条件（用于条件分支）
	ParamMapping map[string]string `json:"param_mapping,omitempty"` // 参数映射：目标参数名 -> 源参数名
}

// Variable 工作流变量
type Variable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, object, array
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// WorkflowExecution 工作流执行实例
type WorkflowExecution struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflow_id"`
	Status       ExecutionStatus        `json:"status"`
	NodeStatuses map[string]NodeStatus `json:"node_statuses"`
	Variables    map[string]interface{} `json:"variables"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// NodeStatus 节点执行状态
type NodeStatus struct {
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"` // pending, running, completed, failed, skipped
	StartedAt *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Output     interface{} `json:"output,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// NodeData 节点执行时的数据
type NodeData struct {
	NodeID string                 `json:"node_id"`
	Data   map[string]interface{} `json:"data"`
}

// Validate 验证工作流定义
func (w *Workflow) Validate() error {
	// 检查节点ID唯一性
	nodeIDs := make(map[string]bool)
	for _, node := range w.Nodes {
		if nodeIDs[node.ID] {
			return ErrDuplicateNodeID{NodeID: node.ID}
		}
		nodeIDs[node.ID] = true
	}

	// 检查边引用的节点是否存在
	nodeMap := make(map[string]bool)
	for _, node := range w.Nodes {
		nodeMap[node.ID] = true
	}

	for _, edge := range w.Edges {
		if !nodeMap[edge.Source] {
			return ErrNodeNotFound{NodeID: edge.Source}
		}
		if !nodeMap[edge.Target] {
			return ErrNodeNotFound{NodeID: edge.Target}
		}
	}

	// 检查是否有循环（简单的拓扑排序检查）
	if hasCycle(w) {
		return ErrCycleDetected{}
	}

	return nil
}

// hasCycle 检查工作流是否有循环
func hasCycle(w *Workflow) bool {
	// 构建邻接表
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, node := range w.Nodes {
		inDegree[node.ID] = 0
		adj[node.ID] = []string{}
	}

	for _, edge := range w.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// 拓扑排序
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	count := 0
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		count++

		for _, neighbor := range adj[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 如果处理的节点数少于总节点数，说明有循环
	return count < len(w.Nodes)
}

// GetStartNodes 获取起始节点（没有输入的节点）
func (w *Workflow) GetStartNodes() []string {
	hasInput := make(map[string]bool)
	for _, edge := range w.Edges {
		hasInput[edge.Target] = true
	}

	startNodes := []string{}
	for _, node := range w.Nodes {
		if !hasInput[node.ID] {
			startNodes = append(startNodes, node.ID)
		}
	}

	return startNodes
}

// GetNodeByID 根据ID获取节点
func (w *Workflow) GetNodeByID(nodeID string) (*Node, bool) {
	for i := range w.Nodes {
		if w.Nodes[i].ID == nodeID {
			return &w.Nodes[i], true
		}
	}
	return nil, false
}

// GetNextNodes 获取节点的下一个节点
func (w *Workflow) GetNextNodes(nodeID string) []string {
	nextNodes := []string{}
	for _, edge := range w.Edges {
		if edge.Source == nodeID {
			nextNodes = append(nextNodes, edge.Target)
		}
	}
	return nextNodes
}

// GetPrevNodes 获取节点的前一个节点
func (w *Workflow) GetPrevNodes(nodeID string) []string {
	prevNodes := []string{}
	for _, edge := range w.Edges {
		if edge.Target == nodeID {
			prevNodes = append(prevNodes, edge.Source)
		}
	}
	return prevNodes
}

// MarshalJSON 自定义 JSON 序列化
func (w *Workflow) MarshalJSON() ([]byte, error) {
	type Alias Workflow
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias:     (*Alias)(w),
		CreatedAt: w.CreatedAt.Format(time.RFC3339),
		UpdatedAt: w.UpdatedAt.Format(time.RFC3339),
	})
}

// UnmarshalJSON 自定义 JSON 反序列化
func (w *Workflow) UnmarshalJSON(data []byte) error {
	type Alias Workflow
	aux := &struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339, aux.CreatedAt)
		if err == nil {
			w.CreatedAt = t
		}
	}

	if aux.UpdatedAt != "" {
		t, err := time.Parse(time.RFC3339, aux.UpdatedAt)
		if err == nil {
			w.UpdatedAt = t
		}
	}

	return nil
}

// NodeExecutor 节点执行器接口
type NodeExecutor interface {
	// Execute 执行节点
	Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error)

	// GetNodeType 获取节点类型
	GetNodeType() NodeType

	// ValidateConfig 验证节点配置
	ValidateConfig(config map[string]interface{}) error
}

// WorkflowRepository 工作流仓储接口
type WorkflowRepository interface {
	// Save 保存工作流
	Save(ctx context.Context, workflow *Workflow) error

	// GetByID 根据ID获取工作流
	GetByID(ctx context.Context, id string) (*Workflow, error)

	// List 列出工作流
	List(ctx context.Context, filters map[string]interface{}) ([]*Workflow, error)

	// Delete 删除工作流
	Delete(ctx context.Context, id string) error

	// SaveExecution 保存执行实例
	SaveExecution(ctx context.Context, execution *WorkflowExecution) error

	// GetExecution 获取执行实例
	GetExecution(ctx context.Context, id string) (*WorkflowExecution, error)

	// ListExecutions 列出执行实例
	ListExecutions(ctx context.Context, workflowID string) ([]*WorkflowExecution, error)
}
