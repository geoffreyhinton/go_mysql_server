package expression

import (
	"testing"
	"time"

	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/stretchr/testify/require"
)

func TestPlus(t *testing.T) {
	var testCases = []struct {
		name        string
		left, right float64
		expected    float64
	}{
		{"1 + 1", 1, 1, 2},
		{"-1 + 1", -1, 1, 0},
		{"0 + 0", 0, 0, 0},
		{"0.14159 + 3.0", 0.14159, 3.0, float64(0.14159) + float64(3)},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			result, err := NewPlus(
				NewLiteral(tt.left, sql.Float64),
				NewLiteral(tt.right, sql.Float64),
			).Eval(sql.NewEmptyContext(), sql.NewRow())
			require.NoError(err)
			require.Equal(tt.expected, result)
		})
	}

	require := require.New(t)
	result, err := NewPlus(NewLiteral("2", sql.LongText), NewLiteral(3, sql.Float64)).
		Eval(sql.NewEmptyContext(), sql.NewRow())
	require.NoError(err)
	require.Equal(float64(5), result)
}

func TestPlusInterval(t *testing.T) {
	require := require.New(t)
	expected := time.Date(2018, time.May, 2, 0, 0, 0, 0, time.UTC)

	op := NewPlus(
		NewLiteral("2018-05-01", sql.LongText),
		NewInterval(NewLiteral(int64(1), sql.Int64), "DAY"),
	)
	result, err := op.Eval(sql.NewEmptyContext(), sql.NewRow())
	require.NoError(err)
	require.Equal(expected, result)
	op = NewPlus(
		NewInterval(NewLiteral(int64(1), sql.Int64), "DAY"),
		NewLiteral("2018-05-01", sql.LongText),
	)

	result, err = op.Eval(sql.NewEmptyContext(), nil)
	require.NoError(err)
	require.Equal(expected, result)
}

func TestMinusInterval(t *testing.T) {
	require := require.New(t)
	expected := time.Date(2018, time.May, 1, 0, 0, 0, 0, time.UTC)
	op := NewMinus(NewLiteral("2018-05-02", sql.LongText),
		NewInterval(NewLiteral(int64(1), sql.Int64), "DAY"))
	result, err := op.Eval(sql.NewEmptyContext(), nil)
	require.NoError(err)
	require.Equal(expected, result)
}
