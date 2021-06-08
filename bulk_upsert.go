package gormbulkups

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// Upsert (INSERT ... ON DUPLICATE KEY UPDATE) multiple records at once
// [objects]        Must be a slice of struct
// [chunkSize]      Number of records to upsert at once.
//                  Embedding a large number of variables at once will raise an error beyond the limit of prepared statement.
//                  Larger size will normally lead the better performance, but 2000 to 3000 is reasonable.
// [excludeColumns] Columns you want to exclude from upsert. You can omit if there is no column you want to exclude.
func BulkUpsert(db *gorm.DB, objects []interface{}, chunkSize int, excludeColumns ...string) error {
	// Split records with specified size not to exceed Database parameter limit
	for _, objSet := range splitObjects(objects, chunkSize) {
		if err := upsertObjSet(db, objSet, excludeColumns...); err != nil {
			return err
		}
	}
	return nil
}

func upsertObjSet(db *gorm.DB, objects []interface{}, excludeColumns ...string) error {
	if len(objects) == 0 {
		return nil
	}

	firstAttrs, err := extractMapValue(objects[0], excludeColumns)
	if err != nil {
		return err
	}

	attrSize := len(firstAttrs)

	// Scope to eventually run SQL
	mainScope := db.NewScope(objects[0])
	// Store placeholders for embedding variables
	placeholders := make([]string, 0, attrSize)

	// Replace with database column name
	dbColumns := make([]string, 0, attrSize)
	for _, key := range sortedKeys(firstAttrs) {
		dbColumns = append(dbColumns, gorm.ToColumnName(key))
	}

	duplicates := make([]string, 0)
	for _, field := range mainScope.Fields() {
		_, hasForeignKey := field.TagSettingsGet("FOREIGNKEY")
		_, isUnique := field.TagSettingsGet("UNIQUE")
		_, hasUniqueIndex := field.TagSettingsGet("UNIQUE_INDEX")
		if containString(excludeColumns, field.Struct.Name) ||
			field.StructField.Relationship != nil ||
			hasForeignKey ||
			field.IsIgnored ||
			field.IsPrimaryKey ||
			isUnique ||
			hasUniqueIndex {
			continue
		}

		duplicates = append(duplicates, fmt.Sprintf("`%s`=VALUES(`%s`)", field.DBName, field.DBName))
	}

	for _, obj := range objects {
		objAttrs, err := extractMapValue(obj, excludeColumns)
		if err != nil {
			return err
		}

		// If object sizes are different, SQL statement loses consistency
		if len(objAttrs) != attrSize {
			return errors.New("attribute sizes are inconsistent")
		}

		scope := db.NewScope(obj)

		// Append variables
		variables := make([]string, 0, attrSize)
		for _, key := range sortedKeys(objAttrs) {
			scope.AddToVars(objAttrs[key])
			variables = append(variables, "?")
		}

		valueQuery := "(" + strings.Join(variables, ", ") + ")"
		placeholders = append(placeholders, valueQuery)

		// Also append variables to mainScope
		mainScope.SQLVars = append(mainScope.SQLVars, scope.SQLVars...)
	}

	sql := "INSERT INTO %s (`%s`) VALUES %s"
	args := []interface{}{
		mainScope.QuotedTableName(),
		strings.Join(dbColumns, "`, `"),
		strings.Join(placeholders, ", "),
	}
	if len(duplicates) > 0 {
		sql += " ON DUPLICATE KEY UPDATE %s"
		args = append(args, strings.Join(duplicates, ", "))
	}
	mainScope.Raw(fmt.Sprintf(sql, args...))

	return db.Exec(mainScope.SQL, mainScope.SQLVars...).Error
}

// Obtain columns and values required for upsert from interface
func extractMapValue(value interface{}, excludeColumns []string) (map[string]interface{}, error) {
	if reflect.ValueOf(value).Kind() != reflect.Struct {
		return nil, errors.New("value must be kind of Struct")
	}

	var attrs = map[string]interface{}{}

	for _, field := range (&gorm.Scope{Value: value}).Fields() {
		// Exclude relational record because it's not directly contained in database columns
		_, hasForeignKey := field.TagSettingsGet("FOREIGNKEY")

		if !containString(excludeColumns, field.Struct.Name) &&
			field.StructField.Relationship == nil &&
			!hasForeignKey &&
			!field.IsIgnored {
			if field.Struct.Name == "CreatedAt" || field.Struct.Name == "UpdatedAt" {
				attrs[field.DBName] = time.Now()
			} else if field.StructField.HasDefaultValue && field.IsBlank {
				// If default value presents and field is empty, assign a default value
				if val, ok := field.TagSettingsGet("DEFAULT"); ok {
					attrs[field.DBName] = val
				} else {
					attrs[field.DBName] = field.Field.Interface()
				}
			} else {
				attrs[field.DBName] = field.Field.Interface()
			}
		}
	}
	return attrs, nil
}
