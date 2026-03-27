package repo

import (
	"reflect"
	"strings"
)

// skippedColumns are db tag values excluded from INSERT/UPDATE operations.
var skippedColumns = map[string]bool{
	"tenant":     true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

// structToColumnsAndValues reads db tags from a struct pointer and returns
// the column names and corresponding values, skipping tenant and timestamp columns.
func structToColumnsAndValues(row any) (cols []string, vals []any) {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		col := strings.Split(tag, ",")[0]
		if skippedColumns[col] {
			continue
		}
		cols = append(cols, col)
		vals = append(vals, v.Field(i).Interface())
	}
	return cols, vals
}

// fieldToColumn maps a Go struct field name to its db tag value.
// Returns the column name and true if found, or ("", false) if not.
func fieldToColumn(row any, goFieldName string) (string, bool) {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	field, ok := t.FieldByName(goFieldName)
	if !ok {
		return "", false
	}
	tag := field.Tag.Get("db")
	if tag == "" || tag == "-" {
		return "", false
	}
	col := strings.Split(tag, ",")[0]
	return col, true
}

// fieldsToColumns converts a slice of Go struct field names to their db column names.
// Fields without a db tag or with skipped column names are silently dropped.
func fieldsToColumns(row any, goFieldNames []string) []string {
	cols := make([]string, 0, len(goFieldNames))
	for _, name := range goFieldNames {
		col, ok := fieldToColumn(row, name)
		if !ok || skippedColumns[col] {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}
