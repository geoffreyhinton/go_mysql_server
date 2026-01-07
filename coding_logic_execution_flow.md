Executor Plan Runner Logic Location:
The execution logic is distributed across individual plan node implementations in the plan directory. Each plan node implements the Iterator Pattern through two key methods:

1. RowIter() Method - Execution Entry Point:
Each plan node implements RowIter(ctx *sql.Context) (sql.RowIter, error) which:

Sets up execution for that operation
Returns an iterator that produces rows
2. Iterator Next() Method - Row-by-Row Execution:
The returned iterator implements Next() (sql.Row, error) which:

Produces one row at a time
Contains the actual execution logic
Examples:
Table Scan Execution (resolved_table.go:28-37):

```go
func (t *ResolvedTable) RowIter(ctx *sql.Context) (sql.RowIter, error) {
    partitions, err := t.Table.Partitions(ctx)  // Get table partitions
    return sql.NewTableRowIter(ctx, t.Table, partitions), nil  // Return table iterator
}
```

Filter Execution (filter.go:26-35):

```go
func (p *Filter) RowIter(ctx *sql.Context) (sql.RowIter, error) {
    i, err := p.Child.RowIter(ctx)  // Get child iterator
    return NewFilterIter(ctx, p.Expression, i), nil  // Wrap with filter logic
}

func (i *FilterIter) Next() (sql.Row, error) {
    for {
        row, err := i.childIter.Next()  // Get row from child
        ok, err := sql.EvaluateCondition(i.ctx, i.cond, row)  // Test condition
        if ok { return row, nil }  // Return if passes filter
    }
}
```

Project Execution (project.go:55-63):

```go
func (p *Project) RowIter(ctx *sql.Context) (sql.RowIter, error) {
    i, err := p.Child.RowIter(ctx)  // Get child iterator
    return &iter{p, i, ctx}, nil   // Return projection iterator
}

func (i *iter) Next() (sql.Row, error) {
    childRow, err := i.childIter.Next()  // Get row from child
    return projectRow(i.ctx, i.p.Projections, childRow)  // Apply projections
}
```

Data Flow Example:
For query: SELECT username FROM users WHERE age > 10


Plan Tree Structure:
Project (p.Child = Filter node)
  ↓
Filter (p.Child = ResolvedTable node)  
  ↓
ResolvedTable (p.Table = users table)


Execution Flow:
```go
// 1. Project calls child (Filter) for iterator
filterIter := filterNode.RowIter(ctx)

// 2. Filter calls its child (ResolvedTable) for iterator  
tableIter := resolvedTableNode.RowIter(ctx)

// 3. When Project needs data:
childRow, err := filterIter.Next()  // ← Returns: sql.Row{username: "a", age: 15}

// 4. Project transforms it:
projectedRow := projectRow(ctx, projections, childRow)  // ← Returns: sql.Row{username: "a"}
```
Key Points:
p.Child = Plan node (logic/structure)
childIter.Next() = Actual data rows like {username: "a", age: 15}
Data is streamed row-by-row through the iterator chain
No node stores all the data - it's produced on-demand
The actual data {username: "a", age: 15} comes from the storage layer (like memory tables) and flows up through the plan tree via the iterator pattern.



Example for When you want SELECT age FROM users:
1. Projections Setup:

```go
// i.p.Projections would actually contain an expression, not just "age" string:
i.p.Projections = []sql.Expression{
    expression.NewGetField(1, sql.Int64, "age", false), // Points to age column
}
```
2. Input Data (childRow):
```go
// From users table - full row with all columns:
childRow = sql.Row{
    "john",  // username (index 0)
    25,      // age (index 1) 
    "admin", // role (index 2)
}
```

3. projectRow Execution:
```go
func projectRow(s *sql.Context, expressions []sql.Expression, row sql.Row) (sql.Row, error) {
    var fields []interface{}
    
    // Loop through projections - in this case just one: "age" expression
    for _, expr := range expressions {  // expr = GetField for "age"
        f, err := expr.Eval(s, row)  // Evaluate "age" against full row
        // GetField.Eval() extracts row[1] = 25
        if err != nil {
            return nil, err
        }
        fields = append(fields, f)  // fields = [25]
    }
    
    return sql.NewRow(fields...), nil  // Returns: sql.Row{25}
}
```

4. Result:
```go
// Input:  childRow = sql.Row{"john", 25, "admin"}
// Output: projectedRow = sql.Row{25}  // Only age column
```
Complete Flow:
```go
// 1. Full row from table
childRow = sql.Row{"john", 25, "admin"}

// 2. Project just age
projectedRow = projectRow(ctx, [ageExpression], childRow)

// 3. Result: only age value
projectedRow = sql.Row{25}
```