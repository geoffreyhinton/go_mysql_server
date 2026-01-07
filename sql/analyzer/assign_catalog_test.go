package analyzer

import (
	"testing"

	"github.com/geoffreyhinton/go_mysql_server/memory"
	"github.com/geoffreyhinton/go_mysql_server/sql"
	"github.com/geoffreyhinton/go_mysql_server/sql/plan"
	"github.com/stretchr/testify/require"
)

func TestAssignCatalog(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("test_table", sql.Schema{
		{Name: "id", Type: sql.Int64, PrimaryKey: true},
		{Name: "name", Type: sql.Text},
	})
	db.AddTable("test_table", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()
	ctx.SetCurrentDatabase("test_db")

	tests := []struct {
		name     string
		node     sql.Node
		testFunc func(t *testing.T, result sql.Node)
	}{
		{
			name: "CreateIndex gets catalog assigned",
			node: plan.NewCreateIndex("idx_name", plan.NewResolvedTable(table), []sql.Expression{}, "", make(map[string]string)),
			testFunc: func(t *testing.T, result sql.Node) {
				createIndex, ok := result.(*plan.CreateIndex)
				require.True(ok)
				require.Equal(catalog, createIndex.Catalog)
				require.Equal("test_db", createIndex.CurrentDatabase)
			},
		},
		{
			name: "DropIndex gets catalog assigned",
			node: plan.NewDropIndex("idx_name", plan.NewResolvedTable(table)),
			testFunc: func(t *testing.T, result sql.Node) {
				dropIndex, ok := result.(*plan.DropIndex)
				require.True(ok)
				require.Equal(catalog, dropIndex.Catalog)
				require.Equal("test_db", dropIndex.CurrentDatabase)
			},
		},
		{
			name: "ShowIndexes gets registry assigned",
			node: plan.NewShowIndexes(db, "test_table", nil),
			testFunc: func(t *testing.T, result sql.Node) {
				showIndexes, ok := result.(*plan.ShowIndexes)
				require.True(ok)
				require.Equal(ctx.IndexRegistry, showIndexes.Registry)
			},
		},
		{
			name: "ShowDatabases gets catalog assigned",
			node: plan.NewShowDatabases(),
			testFunc: func(t *testing.T, result sql.Node) {
				showDatabases, ok := result.(*plan.ShowDatabases)
				require.True(ok)
				require.Equal(catalog, showDatabases.Catalog)
			},
		},
		{
			name: "ShowCreateTable gets catalog and database assigned",
			node: plan.NewShowCreateTable("test_db", nil, plan.NewResolvedTable(table), false),
			testFunc: func(t *testing.T, result sql.Node) {
				showCreateTable, ok := result.(*plan.ShowCreateTable)
				require.True(ok)
				require.Equal(catalog, showCreateTable.Catalog)
				require.Equal("test_db", showCreateTable.Database)
			},
		},
		{
			name: "ShowProcessList gets database and processlist assigned",
			node: plan.NewShowProcessList(),
			testFunc: func(t *testing.T, result sql.Node) {
				showProcessList, ok := result.(*plan.ShowProcessList)
				require.True(ok)
				require.Equal("test_db", showProcessList.Database)
				require.Equal(catalog.ProcessList, showProcessList.ProcessList)
			},
		},
		{
			name: "ShowTableStatus gets catalog assigned",
			node: plan.NewShowTableStatus("test_db"),
			testFunc: func(t *testing.T, result sql.Node) {
				showTableStatus, ok := result.(*plan.ShowTableStatus)
				require.True(ok)
				require.Equal(catalog, showTableStatus.Catalog)
			},
		},
		{
			name: "Use gets catalog assigned",
			node: plan.NewUse(db),
			testFunc: func(t *testing.T, result sql.Node) {
				use, ok := result.(*plan.Use)
				require.True(ok)
				require.Equal(catalog, use.Catalog)
			},
		},
		{
			name: "LockTables gets catalog assigned",
			node: plan.NewLockTables([]*plan.TableLock{{Table: plan.NewResolvedTable(table)}}),
			testFunc: func(t *testing.T, result sql.Node) {
				lockTables, ok := result.(*plan.LockTables)
				require.True(ok)
				require.Equal(catalog, lockTables.Catalog)
			},
		},
		{
			name: "UnlockTables gets catalog assigned",
			node: plan.NewUnlockTables(),
			testFunc: func(t *testing.T, result sql.Node) {
				unlockTables, ok := result.(*plan.UnlockTables)
				require.True(ok)
				require.Equal(catalog, unlockTables.Catalog)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The node needs to be resolved for assignCatalog to work
			resolved := tt.node
			if !resolved.Resolved() {
				var err error
				resolved, err = analyzer.Analyze(ctx, tt.node)
				require.NoError(err)
			}

			result, err := assignCatalog(ctx, analyzer, resolved)
			require.NoError(err)
			require.NotNil(result)

			if tt.testFunc != nil {
				tt.testFunc(t, result)
			}
		})
	}
}

func TestAssignCatalog_UnresolvedNode(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with unresolved node - should return unchanged
	unresolvedTable := plan.NewUnresolvedTable("nonexistent", "")

	result, err := assignCatalog(ctx, analyzer, unresolvedTable)
	require.NoError(err)
	require.Equal(unresolvedTable, result) // Should be unchanged
}

func TestAssignCatalog_UnknownNodeType(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("test_table", sql.Schema{
		{Name: "id", Type: sql.Int64},
	})
	db.AddTable("test_table", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()

	// Test with a node type that doesn't get catalog assigned (like ResolvedTable)
	resolvedTable := plan.NewResolvedTable(table)

	result, err := assignCatalog(ctx, analyzer, resolvedTable)
	require.NoError(err)
	require.Equal(resolvedTable, result) // Should be unchanged
}

func TestAssignCatalog_NestedNodes(t *testing.T) {
	require := require.New(t)

	catalog := sql.NewCatalog()
	db := memory.NewDatabase("test_db")
	catalog.AddDatabase(db)

	table := memory.NewTable("test_table", sql.Schema{
		{Name: "id", Type: sql.Int64},
	})
	db.AddTable("test_table", table)

	analyzer := NewDefault(catalog)
	ctx := sql.NewEmptyContext()
	ctx.SetCurrentDatabase("test_db")

	// Create a nested structure with ShowDatabases inside a Project
	showDatabases := plan.NewShowDatabases()
	project := plan.NewProject([]sql.Expression{}, showDatabases)

	result, err := assignCatalog(ctx, analyzer, project)
	require.NoError(err)
	require.NotNil(result)

	// Check that the nested ShowDatabases node got the catalog assigned
	projectResult, ok := result.(*plan.Project)
	require.True(ok)

	showDbResult, ok := projectResult.Child.(*plan.ShowDatabases)
	require.True(ok)
	require.Equal(catalog, showDbResult.Catalog)
}
