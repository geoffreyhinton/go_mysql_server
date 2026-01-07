package expression

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/stretchr/testify/require"
)

func TestNewGetField(t *testing.T) {
	require := require.New(t)

	field := NewGetField(2, sql.Text, "name", true)

	require.Equal(2, field.Index())
	require.Equal(sql.Text, field.Type())
	require.Equal("name", field.Name())
	require.Equal("", field.Table())
	require.True(field.IsNullable())
	require.True(field.Resolved())
}

func TestNewGetFieldWithTable(t *testing.T) {
	require := require.New(t)

	field := NewGetFieldWithTable(1, sql.Int64, "users", "id", false)

	require.Equal(1, field.Index())
	require.Equal(sql.Int64, field.Type())
	require.Equal("id", field.Name())
	require.Equal("users", field.Table())
	require.False(field.IsNullable())
	require.True(field.Resolved())
}

func TestGetFieldEval(t *testing.T) {
	require := require.New(t)
	ctx := sql.NewEmptyContext()

	field := NewGetField(1, sql.Text, "name", true)
	row := sql.Row{"id1", "Alice", 25}

	result, err := field.Eval(ctx, row)
	require.NoError(err)
	require.Equal("Alice", result)
}

func TestGetFieldEvalIndexOutOfBounds(t *testing.T) {
	testCases := []struct {
		name       string
		fieldIndex int
		row        sql.Row
	}{
		{
			name:       "negative index",
			fieldIndex: -1,
			row:        sql.Row{"value1", "value2"},
		},
		{
			name:       "index too large",
			fieldIndex: 3,
			row:        sql.Row{"value1", "value2"},
		},
		{
			name:       "empty row",
			fieldIndex: 0,
			row:        sql.Row{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			ctx := sql.NewEmptyContext()

			field := NewGetField(tc.fieldIndex, sql.Text, "test", true)

			result, err := field.Eval(ctx, tc.row)
			require.Error(err)
			require.True(ErrIndexOutOfBounds.Is(err))
			require.Nil(result)
		})
	}
}

func TestGetFieldChildren(t *testing.T) {
	require := require.New(t)

	field := NewGetField(0, sql.Text, "name", true)
	children := field.Children()

	require.Nil(children)
	require.Len(children, 0)
}

func TestGetFieldWithChildren(t *testing.T) {
	require := require.New(t)

	field := NewGetField(0, sql.Text, "name", true)

	// Should return same instance with no children
	result, err := field.WithChildren()
	require.NoError(err)
	require.Equal(field, result)

	// Should return error if children provided
	_, err = field.WithChildren(NewLiteral("test", sql.Text))
	require.Error(err)
	require.True(sql.ErrInvalidChildrenNumber.Is(err))
}

func TestGetFieldString(t *testing.T) {
	testCases := []struct {
		name     string
		field    *GetField
		expected string
	}{
		{
			name:     "field without table",
			field:    NewGetField(0, sql.Text, "name", true),
			expected: "name",
		},
		{
			name:     "field with table",
			field:    NewGetFieldWithTable(0, sql.Text, "users", "name", true),
			expected: "users.name",
		},
		{
			name:     "field with empty name",
			field:    NewGetField(0, sql.Text, "", true),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			require.Equal(tc.expected, tc.field.String())
		})
	}
}

func TestGetFieldWithIndex(t *testing.T) {
	require := require.New(t)

	originalField := NewGetFieldWithTable(0, sql.Text, "users", "name", true)
	newField := originalField.WithIndex(5)

	require.Equal(5, newField.(*GetField).Index())
	require.Equal(sql.Text, newField.Type())
	require.Equal("name", newField.(*GetField).Name())
	require.Equal("users", newField.(*GetField).Table())
	require.True(newField.IsNullable())

	// Original field should remain unchanged
	require.Equal(0, originalField.Index())
}

func TestGetSessionField(t *testing.T) {
	require := require.New(t)
	ctx := sql.NewEmptyContext()

	sessionField := NewGetSessionField("auto_increment", sql.Int64, int64(1))

	require.Equal("auto_increment", sessionField.name)
	require.Equal(sql.Int64, sessionField.Type())
	require.Equal("@@auto_increment", sessionField.String())
	require.False(sessionField.IsNullable())
	require.True(sessionField.Resolved())

	result, err := sessionField.Eval(ctx, nil)
	require.NoError(err)
	require.Equal(int64(1), result)
}

func TestGetSessionFieldNullable(t *testing.T) {
	require := require.New(t)

	sessionField := NewGetSessionField("test_var", sql.Text, nil)
	require.True(sessionField.IsNullable())
}

func TestGetSessionFieldChildren(t *testing.T) {
	require := require.New(t)

	sessionField := NewGetSessionField("test_var", sql.Text, "value")
	children := sessionField.Children()

	require.Nil(children)
	require.Len(children, 0)
}

func TestGetSessionFieldWithChildren(t *testing.T) {
	require := require.New(t)

	sessionField := NewGetSessionField("test_var", sql.Text, "value")

	// Should return same instance with no children
	result, err := sessionField.WithChildren()
	require.NoError(err)
	require.Equal(sessionField, result)

	// Should return error if children provided
	_, err = sessionField.WithChildren(NewLiteral("test", sql.Text))
	require.Error(err)
	require.True(sql.ErrInvalidChildrenNumber.Is(err))
}
