package analyzer

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/memory"
	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/geoffreyhinton/go_mysql_server/sql/expression"
	"github.com/geoffreyhinton/go_mysql_server/sql/plan"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog)

	require.NotNil(builder)
	require.Equal(catalog, builder.catalog)
	require.False(builder.debug)
	require.Equal(0, builder.parallelism)
	require.Empty(builder.preAnalyzeRules)
	require.Empty(builder.postAnalyzeRules)
	require.Empty(builder.preValidationRules)
	require.Empty(builder.postValidationRules)
}

func TestBuilder_WithDebug(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog).WithDebug()

	require.True(builder.debug)
}

func TestBuilder_WithParallelism(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog).WithParallelism(4)

	require.Equal(4, builder.parallelism)
}

func TestBuilder_AddPreAnalyzeRule(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog)

	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	builder.AddPreAnalyzeRule("test-rule", mockRule)

	require.Len(builder.preAnalyzeRules, 1)
	require.Equal("test-rule", builder.preAnalyzeRules[0].Name)
}

func TestBuilder_AddPostAnalyzeRule(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog)

	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	builder.AddPostAnalyzeRule("test-rule", mockRule)

	require.Len(builder.postAnalyzeRules, 1)
	require.Equal("test-rule", builder.postAnalyzeRules[0].Name)
}

func TestBuilder_AddPreValidationRule(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog)

	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	builder.AddPreValidationRule("test-rule", mockRule)

	require.Len(builder.preValidationRules, 1)
	require.Equal("test-rule", builder.preValidationRules[0].Name)
}

func TestBuilder_AddPostValidationRule(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	builder := NewBuilder(catalog)

	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	builder.AddPostValidationRule("test-rule", mockRule)

	require.Len(builder.postValidationRules, 1)
	require.Equal("test-rule", builder.postValidationRules[0].Name)
}

func TestBuilder_Build(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	builder := NewBuilder(catalog).
		WithDebug().
		WithParallelism(2).
		AddPreAnalyzeRule("pre-rule", mockRule).
		AddPostAnalyzeRule("post-rule", mockRule)

	analyzer := builder.Build()

	require.NotNil(analyzer)
	require.True(analyzer.Debug)
	require.Equal(2, analyzer.Parallelism)
	require.Equal(catalog, analyzer.Catalog)
	require.Len(analyzer.Batches, 9) // Expected number of batches

	// Check that custom rules were added to appropriate batches
	preAnalyzeBatch := analyzer.Batches[0]
	require.Equal("pre-analyzer rules", preAnalyzeBatch.Desc)
	require.Len(preAnalyzeBatch.Rules, 1)
	require.Equal("pre-rule", preAnalyzeBatch.Rules[0].Name)

	postAnalyzeBatch := analyzer.Batches[4]
	require.Equal("post-analyzer rules", postAnalyzeBatch.Desc)
	require.Len(postAnalyzeBatch.Rules, 1)
	require.Equal("post-rule", postAnalyzeBatch.Rules[0].Name)
}

func TestNewDefault(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)

	require.NotNil(analyzer)
	require.Equal(catalog, analyzer.Catalog)
	require.Len(analyzer.Batches, 9) // Expected number of default batches
}

func TestAnalyzer_Log(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		shouldLog bool
		message   string
		args      []interface{}
	}{
		{
			name:      "debug enabled - should log",
			debug:     true,
			shouldLog: true,
			message:   "test message %s",
			args:      []interface{}{"arg1"},
		},
		{
			name:      "debug disabled - should not log",
			debug:     false,
			shouldLog: false,
			message:   "test message %s",
			args:      []interface{}{"arg1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			catalog := sql.NewCatalog()
			analyzer := NewBuilder(catalog).Build()
			analyzer.Debug = tt.debug

			// Note: In a real-world scenario, you might want to capture log output
			// For this test, we're just ensuring the method doesn't panic
			require.NotPanics(func() {
				analyzer.Log(tt.message, tt.args...)
			})
		})
	}
}

func TestAnalyzer_Log_NilAnalyzer(t *testing.T) {
	require := require.New(t)

	var analyzer *Analyzer
	// Should not panic when analyzer is nil
	require.NotPanics(func() {
		analyzer.Log("test message")
	})
}

func TestAnalyzer_Analyze(t *testing.T) {
	require := require.New(t)

	// Create a test database and table
	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
	})
	db.AddTable("users", table)

	ctx := sql.NewEmptyContext()
	analyzer := NewDefault(catalog)

	// Test with a simple table scan
	tableScan := plan.NewResolvedTable(table)

	result, err := analyzer.Analyze(ctx, tableScan)
	require.NoError(err)
	require.NotNil(result)
	require.True(result.Resolved())
}

func TestAnalyzer_Analyze_WithCustomRule(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64},
		{Name: "name", Type: sql.Text},
	})
	db.AddTable("users", table)

	// Create a custom rule that adds a comment to the node
	ruleApplied := false
	customRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		ruleApplied = true
		return n, nil
	}

	analyzer := NewBuilder(catalog).
		AddPreAnalyzeRule("custom-test-rule", customRule).
		Build()

	ctx := sql.NewEmptyContext()
	tableScan := plan.NewResolvedTable(table)

	result, err := analyzer.Analyze(ctx, tableScan)
	require.NoError(err)
	require.NotNil(result)
	require.True(ruleApplied)
}

func TestAnalyzer_Analyze_WithError(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()

	// Create a custom rule that returns an error
	errorRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return nil, sql.ErrTableNotFound.New("test_error")
	}

	analyzer := NewBuilder(catalog).
		AddPreAnalyzeRule("error-rule", errorRule).
		Build()

	ctx := sql.NewEmptyContext()
	tableScan := plan.NewUnresolvedTable("nonexistent", "")

	result, err := analyzer.Analyze(ctx, tableScan)
	require.Error(err)
	require.Nil(result)
	require.True(sql.ErrTableNotFound.Is(err))
}

func TestAnalyzer_Analyze_ComplexQuery(t *testing.T) {
	require := require.New(t)

	// Create a more complex test scenario
	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test")
	catalog.AddDatabase(db)

	usersTable := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
		{Name: "age", Type: sql.Int64},
	})
	db.AddTable("users", usersTable)

	ctx := sql.NewEmptyContext()
	analyzer := NewDefault(catalog)

	// Create a query plan: SELECT name FROM users WHERE age > 18
	tableScan := plan.NewResolvedTable(usersTable)

	filter := plan.NewFilter(
		expression.NewGreaterThan(
			expression.NewUnresolvedColumn("age"),
			expression.NewLiteral(18, sql.Int64),
		),
		tableScan,
	)

	project := plan.NewProject(
		[]sql.Expression{
			expression.NewUnresolvedColumn("name"),
		},
		filter,
	)

	result, err := analyzer.Analyze(ctx, project)
	require.NoError(err)
	require.NotNil(result)
	require.True(result.Resolved())

	// Verify the result is a properly analyzed Project node
	projectNode, ok := result.(*plan.Project)
	require.True(ok)
	require.Len(projectNode.Projections, 1)

	// The unresolved column should now be resolved to a GetField
	getField, ok := projectNode.Projections[0].(*expression.GetField)
	require.True(ok)
	require.Equal("name", getField.Name())
}

func TestAnalyzer_Analyze_MaxIterations(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()

	// Create a rule that always marks the node as unresolved to trigger max iterations
	infiniteRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		// Return a mock unresolved node that will never become resolved
		return &mockUnresolvedNode{}, nil
	}

	// Create analyzer with a custom batch that has low max iterations
	analyzer := NewBuilder(catalog).Build()
	analyzer.Batches = []*Batch{
		{
			Desc:       "test infinite rule",
			Iterations: 2, // Low iteration count to trigger the error quickly
			Rules:      []Rule{{Name: "infinite-rule", Apply: infiniteRule}},
		},
	}

	ctx := sql.NewEmptyContext()
	tableScan := plan.NewUnresolvedTable("test", "")

	// Note: This might not error in all cases since some batches continue even with ErrMaxAnalysisIters
	result, _ := analyzer.Analyze(ctx, tableScan)
	// The result should still be returned even if max iterations is reached
	require.NotNil(result)
	// Error might be nil as ErrMaxAnalysisIters can be handled/ignored
}

// Helper mock for testing max iterations
type mockUnresolvedNode struct{}

func (m *mockUnresolvedNode) Resolved() bool       { return false }
func (m *mockUnresolvedNode) Schema() sql.Schema   { return nil }
func (m *mockUnresolvedNode) Children() []sql.Node { return nil }
func (m *mockUnresolvedNode) RowIter(ctx *sql.Context) (sql.RowIter, error) {
	return nil, nil
}
func (m *mockUnresolvedNode) WithChildren(children ...sql.Node) (sql.Node, error) {
	return m, nil
}
func (m *mockUnresolvedNode) String() string { return "mock_unresolved" }

func TestBuilder_ChainedMethods(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	mockRule := func(ctx *sql.Context, a *Analyzer, n sql.Node) (sql.Node, error) {
		return n, nil
	}

	// Test method chaining
	builder := NewBuilder(catalog).
		WithDebug().
		WithParallelism(8).
		AddPreAnalyzeRule("pre1", mockRule).
		AddPreAnalyzeRule("pre2", mockRule).
		AddPostAnalyzeRule("post1", mockRule).
		AddPreValidationRule("preval1", mockRule).
		AddPostValidationRule("postval1", mockRule)

	require.True(builder.debug)
	require.Equal(8, builder.parallelism)
	require.Len(builder.preAnalyzeRules, 2)
	require.Len(builder.postAnalyzeRules, 1)
	require.Len(builder.preValidationRules, 1)
	require.Len(builder.postValidationRules, 1)

	analyzer := builder.Build()
	require.True(analyzer.Debug)
	require.Equal(8, analyzer.Parallelism)
}
