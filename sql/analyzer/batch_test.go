package analyzer

import (
	"fmt"
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/memory"
	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/geoffreyhinton/go_mysql_server/sql/plan"
	"github.com/stretchr/testify/require"
)

func TestRule(t *testing.T) {
	require := require.New(t)

	// Test basic Rule creation and execution
	called := false
	rule := Rule{
		Name: "test-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			called = true
			return n, nil
		},
	}

	require.Equal("test-rule", rule.Name)
	require.NotNil(rule.Apply)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// db := memory.NewDatabase("test")
	table := memory.NewTable("users", sql.Schema{{Name: "id", Type: sql.Int64}})
	node := plan.NewResolvedTable(table)

	result, err := rule.Apply(ctx, analyzer, node)
	require.NoError(err)
	require.Equal(node, result)
	require.True(called)
}

func TestBatch_ZeroIterations(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// db := memory.NewDatabase("test")
	table := memory.NewTable("users", sql.Schema{{Name: "id", Type: sql.Int64}})
	node := plan.NewResolvedTable(table)

	// Batch with 0 iterations should return original node
	batch := &Batch{
		Desc:       "test batch",
		Iterations: 0,
		Rules:      []Rule{},
	}

	result, err := batch.Eval(ctx, analyzer, node)
	require.NoError(err)
	require.Equal(node, result)
}

func TestBatch_SingleIteration(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// db := memory.NewDatabase("test")
	table := memory.NewTable("users", sql.Schema{{Name: "id", Type: sql.Int64}})
	node := plan.NewResolvedTable(table)

	callCount := 0
	rule := Rule{
		Name: "increment-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			callCount++
			return n, nil
		},
	}

	batch := &Batch{
		Desc:       "single iteration batch",
		Iterations: 1,
		Rules:      []Rule{rule},
	}

	result, err := batch.Eval(ctx, analyzer, node)
	require.NoError(err)
	require.Equal(node, result)
	require.Equal(1, callCount) // Should be called exactly once
}

func TestBatch_MultipleIterations_NoChange(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// db := memory.NewDatabase("test")
	table := memory.NewTable("users", sql.Schema{{Name: "id", Type: sql.Int64}})
	node := plan.NewResolvedTable(table)

	callCount := 0
	rule := Rule{
		Name: "no-change-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			callCount++
			return n, nil // Return same node (no change)
		},
	}

	batch := &Batch{
		Desc:       "multiple iteration batch",
		Iterations: 5,
		Rules:      []Rule{rule},
	}

	result, err := batch.Eval(ctx, analyzer, node)
	require.NoError(err)
	require.Equal(node, result)
	require.Equal(1, callCount) // Should stop after first iteration since no change
}

func TestBatch_MultipleIterations_WithChanges(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Use a mock node that can be transformed
	originalNode := &mockTransformableNode{value: 0}

	callCount := 0
	rule := Rule{
		Name: "increment-value-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			callCount++
			if mock, ok := n.(*mockTransformableNode); ok {
				if mock.value < 3 { // Transform up to value 3
					return &mockTransformableNode{value: mock.value + 1}, nil
				}
			}
			return n, nil
		},
	}

	batch := &Batch{
		Desc:       "transforming batch",
		Iterations: 10, // High limit
		Rules:      []Rule{rule},
	}

	result, err := batch.Eval(ctx, analyzer, originalNode)
	require.NoError(err)

	finalNode, ok := result.(*mockTransformableNode)
	require.True(ok)
	require.Equal(3, finalNode.value) // Should reach value 3
	require.Equal(4, callCount)       // Should be called 4 times (0->1->2->3->3)
}

func TestBatch_MaxIterationsReached(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	originalNode := &mockTransformableNode{value: 0}

	rule := Rule{
		Name: "always-transform-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			if mock, ok := n.(*mockTransformableNode); ok {
				return &mockTransformableNode{value: mock.value + 1}, nil
			}
			return n, nil
		},
	}

	batch := &Batch{
		Desc:       "max iterations batch",
		Iterations: 3, // Low limit
		Rules:      []Rule{rule},
	}

	result, err := batch.Eval(ctx, analyzer, originalNode)
	require.Error(err)
	require.True(ErrMaxAnalysisIters.Is(err))
	require.NotNil(result) // Should still return the result

	finalNode, ok := result.(*mockTransformableNode)
	require.True(ok)
	require.Equal(3, finalNode.value) // Should reach the max iterations
}

func TestBatch_RuleError(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// db := memory.NewDatabase("test")
	table := memory.NewTable("users", sql.Schema{{Name: "id", Type: sql.Int64}})
	node := plan.NewResolvedTable(table)

	rule := Rule{
		Name: "error-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			return nil, fmt.Errorf("test error")
		},
	}

	batch := &Batch{
		Desc:       "error batch",
		Iterations: 1,
		Rules:      []Rule{rule},
	}

	result, err := batch.Eval(ctx, analyzer, node)
	require.Error(err)
	require.Contains(err.Error(), "test error")
	require.Nil(result)
}

func TestBatch_MultipleRules(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	originalNode := &mockTransformableNode{value: 0}

	rule1CallCount := 0
	rule1 := Rule{
		Name: "add-1-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			rule1CallCount++
			if mock, ok := n.(*mockTransformableNode); ok {
				return &mockTransformableNode{value: mock.value + 1}, nil
			}
			return n, nil
		},
	}

	rule2CallCount := 0
	rule2 := Rule{
		Name: "add-2-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			rule2CallCount++
			if mock, ok := n.(*mockTransformableNode); ok {
				return &mockTransformableNode{value: mock.value + 2}, nil
			}
			return n, nil
		},
	}

	batch := &Batch{
		Desc:       "multiple rules batch",
		Iterations: 1,
		Rules:      []Rule{rule1, rule2},
	}

	result, err := batch.Eval(ctx, analyzer, originalNode)
	require.NoError(err)

	finalNode, ok := result.(*mockTransformableNode)
	require.True(ok)
	require.Equal(3, finalNode.value) // 0 + 1 + 2 = 3
	require.Equal(1, rule1CallCount)
	require.Equal(1, rule2CallCount)
}

func TestBatch_EvalOnce(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	originalNode := &mockTransformableNode{value: 5}

	rule := Rule{
		Name: "double-rule",
		Apply: func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
			if mock, ok := n.(*mockTransformableNode); ok {
				return &mockTransformableNode{value: mock.value * 2}, nil
			}
			return n, nil
		},
	}

	batch := &Batch{
		Desc:       "eval once batch",
		Iterations: 10, // Won't matter for evalOnce
		Rules:      []Rule{rule},
	}

	result, err := batch.evalOnce(ctx, analyzer, originalNode)
	require.NoError(err)

	finalNode, ok := result.(*mockTransformableNode)
	require.True(ok)
	require.Equal(10, finalNode.value) // 5 * 2 = 10
}

func TestNodesEqual(t *testing.T) {
	tests := []struct {
		name     string
		nodeA    sql.Node
		nodeB    sql.Node
		expected bool
	}{
		{
			name:     "identical mock nodes",
			nodeA:    &mockTransformableNode{value: 42},
			nodeB:    &mockTransformableNode{value: 42},
			expected: true,
		},
		{
			name:     "different mock nodes",
			nodeA:    &mockTransformableNode{value: 42},
			nodeB:    &mockTransformableNode{value: 24},
			expected: false,
		},
		{
			name:     "same node reference",
			nodeA:    &mockTransformableNode{value: 42},
			nodeB:    nil, // Will be set to same as nodeA in test
			expected: true,
		},
		{
			name:     "mock equaler node vs non-equaler",
			nodeA:    &mockEqualerNode{id: "test"},
			nodeB:    &mockTransformableNode{value: 1},
			expected: false,
		},
		{
			name:     "two equaler nodes - equal",
			nodeA:    &mockEqualerNode{id: "test"},
			nodeB:    &mockEqualerNode{id: "test"},
			expected: true,
		},
		{
			name:     "two equaler nodes - not equal",
			nodeA:    &mockEqualerNode{id: "test1"},
			nodeB:    &mockEqualerNode{id: "test2"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			nodeA := tt.nodeA
			nodeB := tt.nodeB

			// Handle the "same node reference" case
			if tt.name == "same node reference" {
				nodeB = nodeA
			}

			result := nodesEqual(nodeA, nodeB)
			require.Equal(tt.expected, result)
		})
	}
}

// Mock node for testing transformations
type mockTransformableNode struct {
	value int
}

func (m *mockTransformableNode) Resolved() bool                                      { return true }
func (m *mockTransformableNode) Schema() sql.Schema                                  { return nil }
func (m *mockTransformableNode) Children() []sql.Node                                { return nil }
func (m *mockTransformableNode) RowIter(ctx *sql.Context) (sql.RowIter, error)       { return nil, nil }
func (m *mockTransformableNode) WithChildren(children ...sql.Node) (sql.Node, error) { return m, nil }
func (m *mockTransformableNode) String() string {
	return fmt.Sprintf("mock_transformable(%d)", m.value)
}

// Mock node that implements equaler interface
type mockEqualerNode struct {
	id string
}

func (m *mockEqualerNode) Resolved() bool                                      { return true }
func (m *mockEqualerNode) Schema() sql.Schema                                  { return nil }
func (m *mockEqualerNode) Children() []sql.Node                                { return nil }
func (m *mockEqualerNode) RowIter(ctx *sql.Context) (sql.RowIter, error)       { return nil, nil }
func (m *mockEqualerNode) WithChildren(children ...sql.Node) (sql.Node, error) { return m, nil }
func (m *mockEqualerNode) String() string                                      { return fmt.Sprintf("mock_equaler(%s)", m.id) }
func (m *mockEqualerNode) Equal(other sql.Node) bool {
	if otherMock, ok := other.(*mockEqualerNode); ok {
		return m.id == otherMock.id
	}
	return false
}
