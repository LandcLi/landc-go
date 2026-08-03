package engine

import (
	"fmt"
	"sort"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// DAGGraph — DAG 有向无环图（支持端口级条件分支）
// ============================================================

type EdgeInfo struct {
	TargetID      string
	SourcePort    string // 源端口（条件分支用，如 "true"/"false"）
	TargetPort    string // 目标端口
	ConditionExpr string
	Label         string
}

type DAGGraph struct {
	workflow  *model.Workflow
	adj       map[string][]*EdgeInfo
	inDegree  map[string]int
	nodeMap   map[string]*model.Node
	rootNodes []*model.Node // 缓存
}

func NewDAGGraph(wf *model.Workflow) (*DAGGraph, error) {
	g := &DAGGraph{
		workflow: wf,
		adj:      make(map[string][]*EdgeInfo),
		inDegree: make(map[string]int),
		nodeMap:  make(map[string]*model.Node),
	}

	for _, node := range wf.Nodes {
		g.nodeMap[node.ID] = node
		g.inDegree[node.ID] = 0
		g.adj[node.ID] = nil
	}

	for _, edge := range wf.Edges {
		if edge.Internal {
			continue // 内部边（如Loop回边），不参与DAG调度
		}
		g.adj[edge.SourceID] = append(g.adj[edge.SourceID], &EdgeInfo{
			TargetID:      edge.TargetID,
			SourcePort:    edge.SourcePort,
			TargetPort:    edge.TargetPort,
			ConditionExpr: edge.ConditionExpr,
			Label:         edge.Label,
		})
		g.inDegree[edge.TargetID]++
	}

	if hasCycle, cyclePath := g.detectCycle(); hasCycle {
		return nil, fmt.Errorf("workflow DAG has cycle: %v", cyclePath)
	}

	g.buildRootNodes()
	return g, nil
}

func (g *DAGGraph) buildRootNodes() {
	for _, node := range g.workflow.Nodes {
		if g.inDegree[node.ID] == 0 {
			g.rootNodes = append(g.rootNodes, node)
		}
	}
	sort.Slice(g.rootNodes, func(i, j int) bool {
		return g.rootNodes[i].OrderNo < g.rootNodes[j].OrderNo
	})
}

func (g *DAGGraph) GetNode(nodeID string) *model.Node {
	return g.nodeMap[nodeID]
}

func (g *DAGGraph) GetRootNodes() []*model.Node {
	return g.rootNodes
}

// GetDownstream 获取节点的所有下游边（默认全部，不过滤条件分支）
func (g *DAGGraph) GetDownstream(nodeID string) []*EdgeInfo {
	return g.adj[nodeID]
}

// GetActivatedDownstream — 核心改进：根据节点输出解析条件分支。
// 对于 condition/switch 节点，只激活 SourcePort 匹配的分支；
// 对于普通节点，激活所有下游边（默认行为）。
func (g *DAGGraph) GetActivatedDownstream(nodeID, nodeOutput string) []*EdgeInfo {
	allEdges := g.adj[nodeID]
	if len(allEdges) == 0 {
		return nil
	}

	node := g.nodeMap[nodeID]
	if node == nil {
		return allEdges
	}

	// 只有条件节点才做分支过滤
	if node.Type != model.NodeTypeCondition && node.Type != model.NodeTypeSwitch {
		return allEdges
	}

	// 按 SourcePort 匹配
	var activated []*EdgeInfo
	var defaultEdges []*EdgeInfo

	for _, e := range allEdges {
		switch e.SourcePort {
		case nodeOutput, "":
			activated = append(activated, e)
		case "default":
			defaultEdges = append(defaultEdges, e)
		}
	}

	// 没有匹配分支时，回退到 default 端口
	if len(activated) == 0 && len(defaultEdges) > 0 {
		return defaultEdges
	}

	return activated
}

// GetDepNodes 获取节点的直接上游（排除 Internal 回边，不参与 DAG 入度计算）
func (g *DAGGraph) GetDepNodes(nodeID string) []string {
	var deps []string
	for _, edge := range g.workflow.Edges {
		if edge.TargetID == nodeID && !edge.Internal {
			deps = append(deps, edge.SourceID)
		}
	}
	return deps
}

// GetReadyNodes 获取所有依赖已完成的就绪节点
func (g *DAGGraph) GetReadyNodes(completedNodes map[string]bool) []*model.Node {
	var ready []*model.Node
	for _, node := range g.workflow.Nodes {
		if completedNodes[node.ID] {
			continue
		}
		deps := g.GetDepNodes(node.ID)
		allDone := true
		for _, depID := range deps {
			if !completedNodes[depID] {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, node)
		}
	}
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].OrderNo < ready[j].OrderNo
	})
	return ready
}

// TopologicalSort 拓扑排序
func (g *DAGGraph) TopologicalSort() ([]*model.Node, error) {
	inDegree := make(map[string]int)
	for k, v := range g.inDegree {
		inDegree[k] = v
	}

	var queue []string
	for _, n := range g.rootNodes {
		queue = append(queue, n.ID)
	}

	var result []*model.Node
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, g.nodeMap[nodeID])

		for _, edge := range g.adj[nodeID] {
			inDegree[edge.TargetID]--
			if inDegree[edge.TargetID] == 0 {
				queue = append(queue, edge.TargetID)
			}
		}
	}

	if len(result) != len(g.workflow.Nodes) {
		return nil, fmt.Errorf("DAG has cycle (topological sort incomplete)")
	}
	return result, nil
}

// detectCycle DFS 三色标记法环检测
func (g *DAGGraph) detectCycle() (bool, []string) { //nolint:gocritic // 命名返回值与 DFS 局部栈变量重名
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	for _, n := range g.workflow.Nodes {
		color[n.ID] = white
	}

	var path []string
	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		color[nodeID] = gray
		path = append(path, nodeID)

		for _, edge := range g.adj[nodeID] {
			if color[edge.TargetID] == gray {
				path = append(path, edge.TargetID)
				return true
			}
			if color[edge.TargetID] == white {
				if dfs(edge.TargetID) {
					return true
				}
			}
		}
		color[nodeID] = black
		path = path[:len(path)-1]
		return false
	}

	for _, n := range g.workflow.Nodes {
		if color[n.ID] == white {
			path = nil
			if dfs(n.ID) {
				return true, path
			}
		}
	}
	return false, nil
}
