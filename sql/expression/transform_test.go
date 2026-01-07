package expression

import (
	"fmt"
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/stretchr/testify/require"
)

func TestTransformUp(t *testing.T) {
	require := require.New(t)

	// Test 1: Transform leaf expression (no children)
	t.Run("leaf expression", func(t *testing.T) {
		original := NewLiteral(42, sql.Int64)

		// Transform that doubles the value if it's a literal
		doubleTransform := func(e sql.Expression) (sql.Expression, error) {
			if lit, ok := e.(*Literal); ok {
				if val, ok := lit.Value().(int); ok {
					return NewLiteral(val*2, sql.Int64), nil
				}
			}
			return e, nil
		}

		result, err := TransformUp(original, doubleTransform)
		require.NoError(err)

		transformed, ok := result.(*Literal)
		require.True(ok)
		require.Equal(84, transformed.Value())
	})

	// Test 2: Transform complex expression with children
	t.Run("complex expression with children", func(t *testing.T) {
		// Create: 1 + 2
		original := NewArithmetic(
			NewLiteral(1, sql.Int64),
			NewLiteral(2, sql.Int64),
			"+",
		)

		// Transform that increments all integer literals by 10
		incrementTransform := func(e sql.Expression) (sql.Expression, error) {
			if lit, ok := e.(*Literal); ok {
				if val, ok := lit.Value().(int); ok {
					return NewLiteral(val+10, sql.Int64), nil
				}
			}
			return e, nil
		}

		result, err := TransformUp(original, incrementTransform)
		require.NoError(err)

		// Should be: (1+10) + (2+10) = 11 + 12
		arithmetic, ok := result.(*Arithmetic)
		require.True(ok)

		left, ok := arithmetic.Left.(*Literal)
		require.True(ok)
		require.Equal(11, left.Value())

		right, ok := arithmetic.Right.(*Literal)
		require.True(ok)
		require.Equal(12, right.Value())
	})

	// Test 3: Transform nested expressions (bottom-up)
	t.Run("nested expressions bottom-up", func(t *testing.T) {
		// Create: NOT(1 = 2)
		original := NewNot(NewEquals(
			NewLiteral(1, sql.Int64),
			NewLiteral(2, sql.Int64),
		))

		// Track transformation order
		order := []string{}
		orderTransform := func(e sql.Expression) (sql.Expression, error) {
			switch e.(type) {
			case *Literal:
				order = append(order, "literal")
			case *Equals:
				order = append(order, "equals")
			case *Not:
				order = append(order, "not")
			}
			return e, nil
		}

		_, err := TransformUp(original, orderTransform)
		require.NoError(err)

		// Should process literals first, then equals, then not (bottom-up)
		expected := []string{"literal", "literal", "equals", "not"}
		require.Equal(expected, order)
	})

	// Test 4: Transform with error handling
	t.Run("transform with error", func(t *testing.T) {
		original := NewLiteral("test", sql.Text)

		errorTransform := func(e sql.Expression) (sql.Expression, error) {
			return nil, fmt.Errorf("transformation error")
		}

		result, err := TransformUp(original, errorTransform)
		require.Error(err)
		require.Nil(result)
		require.Contains(err.Error(), "transformation error")
	})

	// Test 5: Transform with children error
	t.Run("transform with children error", func(t *testing.T) {
		// Create expression with children
		original := NewEquals(
			NewLiteral(1, sql.Int64),
			NewLiteral(2, sql.Int64),
		)

		// Transform that errors on literals
		errorOnLiteralTransform := func(e sql.Expression) (sql.Expression, error) {
			if _, ok := e.(*Literal); ok {
				return nil, fmt.Errorf("literal error")
			}
			return e, nil
		}

		result, err := TransformUp(original, errorOnLiteralTransform)
		require.Error(err)
		require.Nil(result)
		require.Contains(err.Error(), "literal error")
	})

	// Test 6: Identity transform (no changes)
	t.Run("identity transform", func(t *testing.T) {
		original := NewEquals(
			NewGetField(0, sql.Int64, "col1", false),
			NewLiteral(42, sql.Int64),
		)

		identityTransform := func(e sql.Expression) (sql.Expression, error) {
			return e, nil
		}

		result, err := TransformUp(original, identityTransform)
		require.NoError(err)
		require.Equal(original.String(), result.String())
	})

	// Test 7: Transform that changes expression type
	t.Run("transform changing expression type", func(t *testing.T) {
		original := NewLiteral(0, sql.Int64)

		// Transform literal 0 to IsNull expression
		zeroToNullTransform := func(e sql.Expression) (sql.Expression, error) {
			if lit, ok := e.(*Literal); ok {
				if val, ok := lit.Value().(int); ok && val == 0 {
					return NewIsNull(NewLiteral(nil, sql.Null)), nil
				}
			}
			return e, nil
		}

		result, err := TransformUp(original, zeroToNullTransform)
		require.NoError(err)

		isNull, ok := result.(*IsNull)
		require.True(ok)
		require.NotNil(isNull)
	})

	// Test 8: Deep nesting
	t.Run("deep nesting", func(t *testing.T) {
		// Create: ((1 + 2) + 3)
		inner := NewArithmetic(
			NewLiteral(1, sql.Int64),
			NewLiteral(2, sql.Int64),
			"+",
		)
		original := NewArithmetic(
			inner,
			NewLiteral(3, sql.Int64),
			"+",
		)

		// Count how many expressions we visit
		count := 0
		countTransform := func(e sql.Expression) (sql.Expression, error) {
			count++
			return e, nil
		}

		result, err := TransformUp(original, countTransform)
		require.NoError(err)
		require.NotNil(result)
		// Should visit: 3 literals + 2 arithmetic = 5 expressions
		require.Equal(5, count)
	})
}

func TestTransformUpWithWithChildrenError(t *testing.T) {
	require := require.New(t)

	// Create a mock expression that fails on WithChildren
	mockExpr := &mockExpressionWithChildrenError{
		children: []sql.Expression{NewLiteral(1, sql.Int64)},
	}

	identityTransform := func(e sql.Expression) (sql.Expression, error) {
		return e, nil
	}

	result, err := TransformUp(mockExpr, identityTransform)
	require.Error(err)
	require.Nil(result)
	require.Contains(err.Error(), "WithChildren error")
}

// Mock expression that always fails on WithChildren
type mockExpressionWithChildrenError struct {
	children []sql.Expression
}

func (m *mockExpressionWithChildrenError) Resolved() bool   { return true }
func (m *mockExpressionWithChildrenError) IsNullable() bool { return false }
func (m *mockExpressionWithChildrenError) Type() sql.Type   { return sql.Int64 }
func (m *mockExpressionWithChildrenError) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	return nil, nil
}
func (m *mockExpressionWithChildrenError) Children() []sql.Expression {
	return m.children
}
func (m *mockExpressionWithChildrenError) WithChildren(children ...sql.Expression) (sql.Expression, error) {
	return nil, fmt.Errorf("WithChildren error")
}
func (m *mockExpressionWithChildrenError) String() string {
	return "mock_error_expr"
}
