package db

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

//go:embed schema_sqlite_columns.json
var sqliteColumnsJSON string

type sqliteColumn struct {
	Name    string `json:"name"`
	JSON    bool   `json:"json"`
	Boolean bool   `json:"boolean"`
	Binary  bool   `json:"binary"`
}

var sqliteColumnsOnce sync.Once
var sqliteTableColumns map[string][]sqliteColumn

func sqliteColumns() map[string][]sqliteColumn {
	sqliteColumnsOnce.Do(func() {
		if err := json.Unmarshal([]byte(sqliteColumnsJSON), &sqliteTableColumns); err != nil {
			panic(err)
		}
	})
	return sqliteTableColumns
}

func sqliteRecordProjection(table, value string) (string, error) {
	columns, exists := sqliteColumns()[table]
	if !exists {
		return "", fmt.Errorf("unknown SQLite archive table %s", table)
	}
	parts := make([]string, 0, len(columns))
	for _, col := range columns {
		expression := "json_extract(" + value + ",'$.' || '" + col.Name + "')"
		if col.Binary {
			expression = "sw_bytea(" + expression + ")"
		}
		parts = append(parts, expression+" AS \""+col.Name+"\"")
	}
	return strings.Join(parts, ","), nil
}

func sqliteRewriteRecords(query string) (string, error) {
	pattern := regexp.MustCompile(`(?i)json_populate_record(set)?\(NULL::(\w+),\s*(\$\d+)::json\)`)
	var failure error
	query = pattern.ReplaceAllStringFunc(query, func(value string) string {
		match := pattern.FindStringSubmatch(value)
		projection, err := sqliteRecordProjection(match[2], "sw_record.value")
		if err != nil {
			failure = err
			return value
		}
		if match[1] != "" {
			return "(SELECT " + projection + " FROM json_each(" + match[3] + ") AS sw_record)"
		}
		return "(SELECT " + projection + " FROM (SELECT " + match[3] + " AS value) AS sw_record)"
	})
	if failure != nil {
		return "", failure
	}
	if strings.Contains(strings.ToUpper(query), "INSERT INTO") && strings.Contains(query, "SELECT * FROM (SELECT") {
		query = sqliteReplace(query, `\)\s+ON CONFLICT`, ") WHERE 1 ON CONFLICT")
	}
	return query, nil
}

func sqliteSplitExpressions(body string) []string {
	var result []string
	start, depth := 0, 0
	for i, char := range body {
		if char == '(' || char == '[' {
			depth++
		}
		if char == ')' || char == ']' {
			depth--
		}
		if char == ',' && depth == 0 {
			result = append(result, strings.TrimSpace(body[start:i]))
			start = i + 1
		}
	}
	return append(result, strings.TrimSpace(body[start:]))
}

func sqliteRowColumns(query, alias string) ([]sqliteColumn, error) {
	for _, match := range regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(\w+)(?:\s+(?:AS )?(\w+))?`).FindAllStringSubmatch(query, -1) {
		if (match[1] == alias || match[2] == alias) && sqliteColumns()[match[1]] != nil {
			return sqliteColumns()[match[1]], nil
		}
	}
	closeMatch := regexp.MustCompile(`(?i)\)\s+(?:AS )?` + regexp.QuoteMeta(alias) + `\b`).FindStringIndex(query)
	if closeMatch == nil {
		return nil, fmt.Errorf("cannot resolve JSON row alias %s", alias)
	}
	close := closeMatch[0]
	depth, open := 1, close-1
	for open >= 0 {
		if query[open] == ')' {
			depth++
		}
		if query[open] == '(' {
			depth--
			if depth == 0 {
				break
			}
		}
		open--
	}
	if open < 0 {
		return nil, fmt.Errorf("unbalanced JSON row subquery")
	}
	subquery := strings.TrimSpace(query[open+1 : close])
	start := strings.Index(strings.ToUpper(subquery), "SELECT ")
	if start < 0 {
		return nil, fmt.Errorf("JSON row has no SELECT")
	}
	from := -1
	depth = 0
	for i := start + 7; i+5 <= len(subquery); i++ {
		if subquery[i] == '(' {
			depth++
		}
		if subquery[i] == ')' {
			depth--
		}
		if depth == 0 && strings.EqualFold(subquery[i:i+4], "FROM") && strings.ContainsRune(" \n\t\r", rune(subquery[i-1])) && strings.ContainsRune(" \n\t\r", rune(subquery[i+4])) {
			from = i
			break
		}
	}
	if from < 0 {
		return nil, fmt.Errorf("JSON row has no FROM")
	}
	projection := subquery[start+7 : from]
	if strings.TrimSpace(projection) == "*" || regexp.MustCompile(`^\w+\.\*$`).MatchString(strings.TrimSpace(projection)) {
		table := regexp.MustCompile(`(?i)^\s*(\w+)`).FindString(subquery[from+4:])
		table = strings.TrimSpace(table)
		if columns := sqliteColumns()[table]; columns != nil {
			return columns, nil
		}
		return nil, fmt.Errorf("unknown JSON row table %s", table)
	}
	var columns []sqliteColumn
	for _, field := range sqliteSplitExpressions(projection) {
		name := regexp.MustCompile(`(?i)(?:\s+AS\s+|\s+)(\w+)\s*$`).FindStringSubmatch(field)
		column := sqliteColumn{}
		if len(name) > 0 {
			column.Name = name[1]
		} else {
			parts := strings.Split(field, ".")
			column.Name = strings.TrimSpace(parts[len(parts)-1])
		}
		if !regexp.MustCompile(`^\w+$`).MatchString(column.Name) {
			return nil, fmt.Errorf("unknown JSON row field %s", field)
		}
		column.JSON = strings.Contains(strings.ToLower(field), "array_agg(") || strings.Contains(strings.ToLower(field), "jsonb_")
		columns = append(columns, column)
	}
	return columns, nil
}

func sqliteRewriteRowJSON(query string) (string, error) {
	pattern := regexp.MustCompile(`(?i)(?:to_jsonb|row_to_json)\((\w+)\)`)
	var failure error
	query = pattern.ReplaceAllStringFunc(query, func(value string) string {
		alias := pattern.FindStringSubmatch(value)[1]
		columns, err := sqliteRowColumns(query, alias)
		if err != nil {
			failure = err
			return value
		}
		parts := make([]string, 0, len(columns)*2)
		for _, column := range columns {
			field := alias + ".\"" + column.Name + "\""
			if column.JSON {
				field = "json(COALESCE(" + field + ",'null'))"
			}
			if column.Boolean {
				field = "json(CASE WHEN " + field + " IS NULL THEN 'null' WHEN " + field + " THEN 'true' ELSE 'false' END)"
			}
			if column.Binary {
				field = "CASE WHEN " + field + " IS NULL THEN NULL ELSE '\\x' || lower(hex(" + field + ")) END"
			}
			parts = append(parts, "'"+column.Name+"'", field)
		}
		return "json_object(" + strings.Join(parts, ",") + ")"
	})
	return query, failure
}
