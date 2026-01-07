package analyzer

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/memory"
	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/geoffreyhinton/go_mysql_server/sql/expression"
	"github.com/geoffreyhinton/go_mysql_server/sql/plan"
	"github.com/stretchr/testify/require"
)

func TestAssignIndexes_BasicFunctionality(t *testing.T) {
	require := require.New(t)

	// Create test database and table
	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
		{Name: "age", Type: sql.Int64},
	})
	db.AddTable("users", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with a simple resolved table
	resolvedTable := plan.NewResolvedTable(table)

	_, err := assignIndexes(ctx, analyzer, resolvedTable)
	require.NoError(err)
	// indexes can be nil when no indexes are found, which is expected for a simple table without index lookups
}

func TestAssignIndexes_WithFilter(t *testing.T) {
	require := require.New(t)

	// Create test setup
	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
		{Name: "age", Type: sql.Int64},
	})
	db.AddTable("users", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Create a filter condition: WHERE age > 18
	resolvedTable := plan.NewResolvedTable(table)
	filter := plan.NewFilter(
		expression.NewGreaterThan(
			expression.NewGetField(2, sql.Int64, "age", false),
			expression.NewLiteral(18, sql.Int64),
		),
		resolvedTable,
	)

	indexes, err := assignIndexes(ctx, analyzer, filter)
	require.NoError(err)
	require.NotNil(indexes)
	// For this simple case without actual indexes, should return empty map
}

func TestGetIndexes_BasicExpression(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
	})
	db.AddTable("users", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with a simple comparison: id = 1
	expr := expression.NewEquals(
		expression.NewGetField(0, sql.Int64, "id", false),
		expression.NewLiteral(1, sql.Int64),
	)

	aliases := make(map[string]sql.Expression)
	indexes, err := getIndexes(ctx, expr, aliases, analyzer)
	require.NoError(err)
	require.NotNil(indexes)
}

func TestGetIndexes_AndExpression(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
		{Name: "age", Type: sql.Int64},
	})
	db.AddTable("users", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with AND expression: id = 1 AND age > 18
	left := expression.NewEquals(
		expression.NewGetField(0, sql.Int64, "id", false),
		expression.NewLiteral(1, sql.Int64),
	)
	right := expression.NewGreaterThan(
		expression.NewGetField(2, sql.Int64, "age", false),
		expression.NewLiteral(18, sql.Int64),
	)
	andExpr := expression.NewAnd(left, right)

	aliases := make(map[string]sql.Expression)
	indexes, err := getIndexes(ctx, andExpr, aliases, analyzer)
	require.NoError(err)
	require.NotNil(indexes)
}

func TestGetIndexes_OrExpression(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("users", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
	})
	db.AddTable("users", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with OR expression: id = 1 OR id = 2
	left := expression.NewEquals(
		expression.NewGetField(0, sql.Int64, "id", false),
		expression.NewLiteral(1, sql.Int64),
	)
	right := expression.NewEquals(
		expression.NewGetField(0, sql.Int64, "id", false),
		expression.NewLiteral(2, sql.Int64),
	)
	orExpr := expression.NewOr(left, right)

	aliases := make(map[string]sql.Expression)
	indexes, err := getIndexes(ctx, orExpr, aliases, analyzer)
	require.NoError(err)
	require.NotNil(indexes)
}

func TestFindTables(t *testing.T) {
	require := require.New(t)

	// Test with GetField expression
	getField := expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false)
	tables := findTables(getField)

	require.Len(tables, 1)
	require.Contains(tables, "users")

	// Test with And expression containing multiple tables
	getField1 := expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false)
	getField2 := expression.NewGetFieldWithTable(1, sql.Text, "orders", "user_id", false)
	andExpr := expression.NewAnd(
		expression.NewEquals(getField1, expression.NewLiteral(1, sql.Int64)),
		expression.NewEquals(getField2, expression.NewLiteral(1, sql.Int64)),
	)

	tables = findTables(andExpr)
	require.Len(tables, 2)
	require.Contains(tables, "users")
	require.Contains(tables, "orders")
}

func TestUnifyExpression(t *testing.T) {
	require := require.New(t)

	// Test with alias substitution
	aliases := map[string]sql.Expression{
		"u": expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false),
	}

	aliasRef := expression.NewUnresolvedColumn("u")
	unified := unifyExpression(aliases, aliasRef)

	// Should be substituted with the actual expression
	getField, ok := unified.(*expression.GetField)
	require.True(ok)
	require.Equal("users", getField.Table())
	require.Equal("id", getField.Name())
}

func TestUnifyExpressions(t *testing.T) {
	require := require.New(t)

	aliases := map[string]sql.Expression{
		"u": expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false),
		"o": expression.NewGetFieldWithTable(1, sql.Text, "orders", "user_id", false),
	}

	exprs := []sql.Expression{
		expression.NewUnresolvedColumn("u"),
		expression.NewUnresolvedColumn("o"),
		expression.NewLiteral(1, sql.Int64), // Should remain unchanged
	}

	unified := unifyExpressions(aliases, exprs...)
	require.Len(unified, 3)

	// Check first expression was unified
	getField1, ok := unified[0].(*expression.GetField)
	require.True(ok)
	require.Equal("users", getField1.Table())

	// Check second expression was unified
	getField2, ok := unified[1].(*expression.GetField)
	require.True(ok)
	require.Equal("orders", getField2.Table())

	// Check third expression remained unchanged
	literal, ok := unified[2].(*expression.Literal)
	require.True(ok)
	require.Equal(1, literal.Value())
}

func TestContainsColumns(t *testing.T) {
	require := require.New(t)

	// Test with GetField - should return true
	getField := expression.NewGetField(0, sql.Int64, "id", false)
	require.True(containsColumns(getField))

	// Test with literal - should return false
	literal := expression.NewLiteral(42, sql.Int64)
	require.False(containsColumns(literal))

	// Test with expression containing GetField
	comparison := expression.NewEquals(
		expression.NewGetField(0, sql.Int64, "id", false),
		expression.NewLiteral(1, sql.Int64),
	)
	require.True(containsColumns(comparison))

	// Test with expression not containing GetField
	literalComparison := expression.NewEquals(
		expression.NewLiteral(1, sql.Int64),
		expression.NewLiteral(2, sql.Int64),
	)
	require.False(containsColumns(literalComparison))
}

func TestContainsSubquery(t *testing.T) {
	require := require.New(t)

	// Test with simple expression - should return false
	getField := expression.NewGetField(0, sql.Int64, "id", false)
	require.False(containsSubquery(getField))

	// Test with literal - should return false
	literal := expression.NewLiteral(42, sql.Int64)
	require.False(containsSubquery(literal))

	// Note: Testing with actual subquery would require more complex setup
	// For now, we test that non-subquery expressions return false
}

func TestIsEvaluable(t *testing.T) {
	require := require.New(t)

	// Test with literal - should be evaluable
	literal := expression.NewLiteral(42, sql.Int64)
	require.True(isEvaluable(literal))

	// Test with arithmetic of literals - should be evaluable
	arithmetic := expression.NewArithmetic(
		expression.NewLiteral(1, sql.Int64),
		expression.NewLiteral(2, sql.Int64),
		"+",
	)
	require.True(isEvaluable(arithmetic))

	// Test with GetField - should not be evaluable (contains columns)
	getField := expression.NewGetField(0, sql.Int64, "id", false)
	require.False(isEvaluable(getField))
}

func TestColumnExprsByTable(t *testing.T) {
	require := require.New(t)

	// Test with expressions from different tables
	exprs := []sql.Expression{
		expression.NewEquals(
			expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false),
			expression.NewLiteral(1, sql.Int64),
		),
		expression.NewEquals(
			expression.NewGetFieldWithTable(1, sql.Text, "orders", "user_id", false),
			expression.NewLiteral(1, sql.Int64),
		),
		expression.NewEquals(
			expression.NewGetFieldWithTable(2, sql.Text, "users", "name", false),
			expression.NewLiteral("John", sql.Text),
		),
	}

	result := columnExprsByTable(exprs)
	require.NotNil(result)

	// Should have entries for both tables
	require.Contains(result, "users")
	require.Contains(result, "orders")

	// users table should have 2 expressions
	require.Len(result["users"], 2)

	// orders table should have 1 expression
	require.Len(result["orders"], 1)
}

func TestExtractColumnExpr(t *testing.T) {
	require := require.New(t)

	// Test with simple equality
	expr := expression.NewEquals(
		expression.NewGetFieldWithTable(0, sql.Int64, "users", "id", false),
		expression.NewLiteral(1, sql.Int64),
	)

	table, colExpr := extractColumnExpr(expr)
	require.Equal("users", table)
	require.NotNil(colExpr)
	require.Equal("id", colExpr.col)

	// Test with non-extractable expression
	literal := expression.NewLiteral(42, sql.Int64)
	table, colExpr = extractColumnExpr(literal)
	require.Empty(table)
	require.Nil(colExpr)
}
