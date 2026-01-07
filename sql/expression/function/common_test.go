package function

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/stretchr/testify/require"
)

func eval(t *testing.T, e sql.Expression, row sql.Row) interface{} {
	ctx := sql.NewEmptyContext()

	t.Helper()
	v, err := e.Eval(ctx, row)
	require.NoError(t, err)
	return v
}
