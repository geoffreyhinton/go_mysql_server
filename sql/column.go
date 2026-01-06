package sql

import "reflect"

type Column struct {
	Name       string
	Type       Type
	Default    interface{}
	Nullable   bool
	Source     string
	PrimaryKey bool
	Comment    string
}

// Check ensures the value is correct for this column.
func (c *Column) Check(v interface{}) bool {
	if v == nil {
		return c.Nullable
	}

	_, err := c.Type.Convert(v)
	return err == nil
}

// Equals checks whether two columns are equal.
func (c *Column) Equals(c2 *Column) bool {
	return c.Name == c2.Name &&
		c.Source == c2.Source &&
		c.Nullable == c2.Nullable &&
		reflect.DeepEqual(c.Default, c2.Default) &&
		reflect.DeepEqual(c.Type, c2.Type)
}
