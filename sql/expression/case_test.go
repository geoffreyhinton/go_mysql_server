package expression

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/stretchr/testify/require"
)

func TestCase(t *testing.T) {
	f1 := NewCase(
		NewGetField(0, sql.Int64, "foo", false),
		[]CaseBranch{
			{Cond: NewLiteral(int64(1), sql.Int64), Value: NewLiteral(int64(2), sql.Int64)},
			{Cond: NewLiteral(int64(3), sql.Int64), Value: NewLiteral(int64(4), sql.Int64)},
			{Cond: NewLiteral(int64(5), sql.Int64), Value: NewLiteral(int64(6), sql.Int64)},
		},
		NewLiteral(int64(7), sql.Int64),
	)

	f2 := NewCase(
		nil,
		[]CaseBranch{
			{
				Cond: NewEquals(
					NewGetField(0, sql.Int64, "foo", false),
					NewLiteral(int64(1), sql.Int64),
				),
				Value: NewLiteral(int64(2), sql.Int64),
			},
			{
				Cond: NewEquals(
					NewGetField(0, sql.Int64, "foo", false),
					NewLiteral(int64(3), sql.Int64),
				),
				Value: NewLiteral(int64(4), sql.Int64),
			},
			{
				Cond: NewEquals(
					NewGetField(0, sql.Int64, "foo", false),
					NewLiteral(int64(5), sql.Int64),
				),
				Value: NewLiteral(int64(6), sql.Int64),
			},
		},
		NewLiteral(int64(7), sql.Int64),
	)

	f3 := NewCase(
		NewGetField(0, sql.Int64, "foo", false),
		[]CaseBranch{
			{Cond: NewLiteral(int64(1), sql.Int64), Value: NewLiteral(int64(2), sql.Int64)},
			{Cond: NewLiteral(int64(3), sql.Int64), Value: NewLiteral(int64(4), sql.Int64)},
			{Cond: NewLiteral(int64(5), sql.Int64), Value: NewLiteral(int64(6), sql.Int64)},
		},
		nil,
	)

	testCases := []struct {
		name     string
		f        *Case
		row      sql.Row
		expected interface{}
	}{
		{
			"with expr and else branch 1",
			f1,
			sql.Row{int64(1)},
			int64(2),
		},
		{
			"with expr and else branch 2",
			f1,
			sql.Row{int64(3)},
			int64(4),
		},
		{
			"with expr and else branch 3",
			f1,
			sql.Row{int64(5)},
			int64(6),
		},
		{
			"with expr and else, else branch",
			f1,
			sql.Row{int64(9)},
			int64(7),
		},
		{
			"without expr and else branch 1",
			f2,
			sql.Row{int64(1)},
			int64(2),
		},
		{
			"without expr and else branch 2",
			f2,
			sql.Row{int64(3)},
			int64(4),
		},
		{
			"without expr and else branch 3",
			f2,
			sql.Row{int64(5)},
			int64(6),
		},
		{
			"without expr and else, else branch",
			f2,
			sql.Row{int64(9)},
			int64(7),
		},
		{
			"without else, else branch",
			f3,
			sql.Row{int64(9)},
			nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			result, err := tt.f.Eval(sql.NewEmptyContext(), tt.row)
			require.NoError(err)
			require.Equal(tt.expected, result)
		})
	}
}

func TestCaseNullBranch(t *testing.T) {
	require := require.New(t)
	f := NewCase(
		NewGetField(0, sql.Int64, "x", false),
		[]CaseBranch{
			{
				Cond:  NewLiteral(int64(1), sql.Int64),
				Value: NewLiteral(nil, sql.Null),
			},
		},
		nil,
	)
	result, err := f.Eval(sql.NewEmptyContext(), sql.Row{int64(1)})
	require.NoError(err)
	require.Nil(result)
}

func TestCaseWithChildren(t *testing.T) {
	// Test case 1: Simple CASE with Expr and Else
	originalCase := NewCase(
		NewGetField(0, sql.Int64, "foo", false),
		[]CaseBranch{
			{Cond: NewLiteral(int64(1), sql.Int64), Value: NewLiteral(int64(2), sql.Int64)},
			{Cond: NewLiteral(int64(3), sql.Int64), Value: NewLiteral(int64(4), sql.Int64)},
		},
		NewLiteral(int64(5), sql.Int64),
	)

	// New children: [expr, cond1, value1, cond2, value2, else]
	newChildren := []sql.Expression{
		NewGetField(1, sql.Int64, "bar", false), // new expr
		NewLiteral(int64(10), sql.Int64),        // new cond1
		NewLiteral(int64(20), sql.Int64),        // new value1
		NewLiteral(int64(30), sql.Int64),        // new cond2
		NewLiteral(int64(40), sql.Int64),        // new value2
		NewLiteral(int64(50), sql.Int64),        // new else
	}

	newCase, err := originalCase.WithChildren(newChildren...)
	require.NoError(t, err)

	c, ok := newCase.(*Case)
	require.True(t, ok)

	// Verify the new structure
	require.Equal(t, newChildren[0], c.Expr)
	require.Len(t, c.Branches, 2)
	require.Equal(t, newChildren[1], c.Branches[0].Cond)
	require.Equal(t, newChildren[2], c.Branches[0].Value)
	require.Equal(t, newChildren[3], c.Branches[1].Cond)
	require.Equal(t, newChildren[4], c.Branches[1].Value)
	require.Equal(t, newChildren[5], c.Else)

	// Test case 2: Searched CASE without Expr but with Else
	searchedCase := NewCase(
		nil,
		[]CaseBranch{
			{
				Cond:  NewEquals(NewGetField(0, sql.Int64, "x", false), NewLiteral(int64(1), sql.Int64)),
				Value: NewLiteral(int64(100), sql.Int64),
			},
		},
		NewLiteral(int64(200), sql.Int64),
	)

	// New children: [cond1, value1, else]
	searchedChildren := []sql.Expression{
		NewEquals(NewGetField(0, sql.Int64, "y", false), NewLiteral(int64(2), sql.Int64)),
		NewLiteral(int64(300), sql.Int64),
		NewLiteral(int64(400), sql.Int64),
	}

	newSearchedCase, err := searchedCase.WithChildren(searchedChildren...)
	require.NoError(t, err)

	sc, ok := newSearchedCase.(*Case)
	require.True(t, ok)

	require.Nil(t, sc.Expr)
	require.Len(t, sc.Branches, 1)
	require.Equal(t, searchedChildren[0], sc.Branches[0].Cond)
	require.Equal(t, searchedChildren[1], sc.Branches[0].Value)
	require.Equal(t, searchedChildren[2], sc.Else)

	// Test case 3: Case without Expr and without Else
	noElseCase := NewCase(
		nil,
		[]CaseBranch{
			{Cond: NewLiteral(true, sql.Boolean), Value: NewLiteral(int64(1), sql.Int64)},
			{Cond: NewLiteral(false, sql.Boolean), Value: NewLiteral(int64(2), sql.Int64)},
		},
		nil,
	)

	// New children: [cond1, value1, cond2, value2]
	noElseChildren := []sql.Expression{
		NewLiteral(false, sql.Boolean),
		NewLiteral(int64(10), sql.Int64),
		NewLiteral(true, sql.Boolean),
		NewLiteral(int64(20), sql.Int64),
	}

	newNoElseCase, err := noElseCase.WithChildren(noElseChildren...)
	require.NoError(t, err)

	nec, ok := newNoElseCase.(*Case)
	require.True(t, ok)

	require.Nil(t, nec.Expr)
	require.Len(t, nec.Branches, 2)
	require.Equal(t, noElseChildren[0], nec.Branches[0].Cond)
	require.Equal(t, noElseChildren[1], nec.Branches[0].Value)
	require.Equal(t, noElseChildren[2], nec.Branches[1].Cond)
	require.Equal(t, noElseChildren[3], nec.Branches[1].Value)
	require.Nil(t, nec.Else)

	// Test case 4: Error case - wrong number of children
	wrongChildren := []sql.Expression{
		NewLiteral(int64(1), sql.Int64), // Too few children
	}

	_, err = originalCase.WithChildren(wrongChildren...)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid children number")

	// Test case 5: Case with Expr but no Else
	exprNoElseCase := NewCase(
		NewGetField(0, sql.Int64, "test", false),
		[]CaseBranch{
			{Cond: NewLiteral(int64(1), sql.Int64), Value: NewLiteral(int64(10), sql.Int64)},
		},
		nil,
	)

	// New children: [expr, cond1, value1]
	exprNoElseChildren := []sql.Expression{
		NewGetField(1, sql.Int64, "new_test", false),
		NewLiteral(int64(5), sql.Int64),
		NewLiteral(int64(50), sql.Int64),
	}

	newExprNoElseCase, err := exprNoElseCase.WithChildren(exprNoElseChildren...)
	require.NoError(t, err)

	enec, ok := newExprNoElseCase.(*Case)
	require.True(t, ok)

	require.Equal(t, exprNoElseChildren[0], enec.Expr)
	require.Len(t, enec.Branches, 1)
	require.Equal(t, exprNoElseChildren[1], enec.Branches[0].Cond)
	require.Equal(t, exprNoElseChildren[2], enec.Branches[0].Value)
	require.Nil(t, enec.Else)
}

func TestCaseChildren(t *testing.T) {
	require := require.New(t)

	// Test case with all components: expr, branches, and else
	expr := NewGetField(0, sql.Int64, "foo", false)
	branch1Cond := NewLiteral(int64(1), sql.Int64)
	branch1Value := NewLiteral(int64(2), sql.Int64)
	branch2Cond := NewLiteral(int64(3), sql.Int64)
	branch2Value := NewLiteral(int64(4), sql.Int64)
	elseExpr := NewLiteral(int64(7), sql.Int64)

	caseWithAll := NewCase(
		expr,
		[]CaseBranch{
			{Cond: branch1Cond, Value: branch1Value},
			{Cond: branch2Cond, Value: branch2Value},
		},
		elseExpr,
	)

	children := caseWithAll.Children()
	expected := []sql.Expression{
		expr,            // expr comes first
		branch1Cond,     // then branch conditions and values in pairs
		branch1Value,
		branch2Cond,
		branch2Value,
		elseExpr,        // else comes last
	}
	require.Equal(expected, children)

	// Test case without expr (searched case)
	caseNoExpr := NewCase(
		nil,
		[]CaseBranch{
			{Cond: branch1Cond, Value: branch1Value},
			{Cond: branch2Cond, Value: branch2Value},
		},
		elseExpr,
	)

	children = caseNoExpr.Children()
	expected = []sql.Expression{
		// no expr at beginning
		branch1Cond,
		branch1Value,
		branch2Cond,
		branch2Value,
		elseExpr,
	}
	require.Equal(expected, children)

	// Test case without else
	caseNoElse := NewCase(
		expr,
		[]CaseBranch{
			{Cond: branch1Cond, Value: branch1Value},
		},
		nil,
	)

	children = caseNoElse.Children()
	expected = []sql.Expression{
		expr,
		branch1Cond,
		branch1Value,
		// no else at end
	}
	require.Equal(expected, children)

	// Test case with only branches (no expr, no else)
	caseOnlyBranches := NewCase(
		nil,
		[]CaseBranch{
			{Cond: branch1Cond, Value: branch1Value},
		},
		nil,
	)

	children = caseOnlyBranches.Children()
	expected = []sql.Expression{
		branch1Cond,
		branch1Value,
	}
	require.Equal(expected, children)
}
