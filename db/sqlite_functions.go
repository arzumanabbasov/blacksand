package db

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strings"
	"time"

	"modernc.org/sqlite"
)

func sqliteText(value driver.Value) string {
	if value == nil {
		return ""
	}
	if data, ok := value.([]byte); ok {
		return string(data)
	}
	return fmt.Sprint(value)
}
func sqliteJSON(value driver.Value) any {
	var result any
	decoder := json.NewDecoder(strings.NewReader(sqliteText(value)))
	decoder.UseNumber()
	_ = decoder.Decode(&result)
	return result
}
func sqliteEncoded(value any) (driver.Value, error) {
	data, err := json.Marshal(value)
	return string(data), err
}
func sqliteArray(value driver.Value) []any {
	result, _ := sqliteJSON(value).([]any)
	if result == nil {
		return []any{}
	}
	return result
}
func sqliteNumber(value driver.Value) float64 {
	switch n := value.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	default:
		var number float64
		fmt.Sscan(sqliteText(value), &number)
		return number
	}
}

func sqliteTime(value driver.Value) (time.Time, error) {
	if value, ok := value.(time.Time); ok {
		return value, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05-07", "2006-01-02 15:04:05"} {
		if result, err := time.Parse(layout, sqliteText(value)); err == nil {
			return result, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid SQLite timestamp")
}
func sqliteStamp(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.999999999-07:00")
}
func sqliteContains(left, right any) bool {
	if a, ok := left.([]any); ok {
		b, ok := right.([]any)
		if !ok {
			b = []any{right}
		}
		for _, wanted := range b {
			found := false
			for _, item := range a {
				if sqliteContains(item, wanted) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	if a, ok := left.(map[string]any); ok {
		b, ok := right.(map[string]any)
		if !ok {
			return false
		}
		for key, wanted := range b {
			item, present := a[key]
			if !present || !sqliteContains(item, wanted) {
				return false
			}
		}
		return true
	}
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
func registerSQLiteFunctions() {
	functions := map[string]func([]driver.Value) (driver.Value, error){
		"sw_date_trunc": func(args []driver.Value) (driver.Value, error) {
			if sqliteText(args[0]) != "day" {
				return nil, fmt.Errorf("unsupported date_trunc unit")
			}
			value, err := sqliteTime(args[1])
			if err != nil {
				return nil, err
			}
			return sqliteStamp(time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)), nil
		},
		"sw_bytea": func(args []driver.Value) (driver.Value, error) {
			if args[0] == nil {
				return nil, nil
			}
			value := strings.TrimPrefix(sqliteText(args[0]), "\\x")
			return hex.DecodeString(value)
		},
		"sw_greatest": func(args []driver.Value) (driver.Value, error) {
			var result driver.Value
			for _, value := range args {
				if value != nil && (result == nil || sqliteNumber(value) > sqliteNumber(result)) {
					result = value
				}
			}
			return result, nil
		},
		"sw_least": func(args []driver.Value) (driver.Value, error) {
			var result driver.Value
			for _, value := range args {
				if value != nil && (result == nil || sqliteNumber(value) < sqliteNumber(result)) {
					result = value
				}
			}
			return result, nil
		},
		"sw_regex": func(args []driver.Value) (driver.Value, error) {
			pattern, err := regexp.Compile(sqliteText(args[1]))
			if err != nil {
				return nil, err
			}
			return pattern.MatchString(sqliteText(args[0])), nil
		},
		"sw_now": func([]driver.Value) (driver.Value, error) { return sqliteStamp(time.Now()), nil },
		"sw_time_add": func(args []driver.Value) (driver.Value, error) {
			value, err := sqliteTime(args[0])
			if err != nil {
				return nil, err
			}
			return sqliteStamp(value.Add(time.Duration(sqliteNumber(args[1]) * float64(time.Second)))), nil
		},
		"sw_epoch": func(args []driver.Value) (driver.Value, error) {
			if args[0] == nil {
				return nil, nil
			}
			value, err := sqliteTime(args[0])
			if err != nil {
				return nil, err
			}
			return float64(value.UnixNano()) / 1e9, nil
		},
		"sw_day": func(args []driver.Value) (driver.Value, error) {
			value, err := sqliteTime(args[0])
			if err != nil {
				return nil, err
			}
			return value.UTC().Format("2006-01-02"), nil
		},
		"sw_array_length": func(args []driver.Value) (driver.Value, error) { return int64(len(sqliteArray(args[0]))), nil },
		"sw_array_append": func(args []driver.Value) (driver.Value, error) {
			return sqliteEncoded(append(sqliteArray(args[0]), args[1]))
		},
		"sw_array_remove": func(args []driver.Value) (driver.Value, error) {
			result := []any{}
			for _, value := range sqliteArray(args[0]) {
				if !sqliteContains(value, args[1]) {
					result = append(result, value)
				}
			}
			return sqliteEncoded(result)
		},
		"sw_array_has": func(args []driver.Value) (driver.Value, error) {
			for _, value := range sqliteArray(args[0]) {
				if sqliteContains(value, args[1]) {
					return true, nil
				}
			}
			return false, nil
		},
		"sw_json_contains": func(args []driver.Value) (driver.Value, error) {
			return sqliteContains(sqliteJSON(args[0]), sqliteJSON(args[1])), nil
		},
		"sw_json_has": func(args []driver.Value) (driver.Value, error) {
			value := sqliteJSON(args[0])
			if object, ok := value.(map[string]any); ok {
				_, exists := object[sqliteText(args[1])]
				return exists, nil
			}
			for _, entry := range sqliteArray(args[0]) {
				if entry == sqliteText(args[1]) {
					return true, nil
				}
			}
			return false, nil
		},
		"sw_to_json": func(args []driver.Value) (driver.Value, error) { return sqliteEncoded(args[0]) },
		"sw_json_merge": func(args []driver.Value) (driver.Value, error) {
			left, right := sqliteJSON(args[0]), sqliteJSON(args[1])
			if object, ok := left.(map[string]any); ok {
				if newer, ok := right.(map[string]any); ok {
					for key, value := range newer {
						object[key] = value
					}
					return sqliteEncoded(object)
				}
			}
			a, ok := left.([]any)
			if !ok {
				a = []any{left}
			}
			b, ok := right.([]any)
			if !ok {
				b = []any{right}
			}
			return sqliteEncoded(append(a, b...))
		},
		"sw_json_set": func(args []driver.Value) (driver.Value, error) {
			object, ok := sqliteJSON(args[0]).(map[string]any)
			if !ok {
				object = map[string]any{}
			}
			key := strings.Trim(sqliteText(args[1]), "{}")
			object[key] = sqliteJSON(args[2])
			return sqliteEncoded(object)
		},
		"sw_try_inet": func(args []driver.Value) (driver.Value, error) {
			ip, err := netip.ParseAddr(sqliteText(args[0]))
			if err != nil {
				return nil, nil
			}
			return ip.String(), nil
		},
		"sw_masklen": func(args []driver.Value) (driver.Value, error) {
			prefix, err := netip.ParsePrefix(sqliteText(args[0]))
			if err != nil {
				return nil, nil
			}
			return int64(prefix.Bits()), nil
		},
		"sw_net_contains": func(args []driver.Value) (driver.Value, error) {
			prefix, err := netip.ParsePrefix(sqliteText(args[0]))
			if err != nil {
				return false, nil
			}
			ip, err := netip.ParseAddr(sqliteText(args[1]))
			if err != nil {
				return false, nil
			}
			return prefix.Contains(ip), nil
		},
		"sw_regexp_replace": func(args []driver.Value) (driver.Value, error) {
			pattern, err := regexp.Compile(sqliteText(args[1]))
			if err != nil {
				return nil, err
			}
			return pattern.ReplaceAllString(sqliteText(args[0]), sqliteText(args[2])), nil
		},
		"sw_array_union": sqliteUnion,
	}
	for name, fn := range functions {
		fn := fn
		if err := sqlite.RegisterFunction(name, &sqlite.FunctionImpl{NArgs: -1, Deterministic: name != "sw_now", Scalar: func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) { return fn(args) }}); err != nil {
			panic(err)
		}
	}
}

func sqliteUnion(args []driver.Value) (driver.Value, error) {
	mode := sqliteText(args[2])
	items := append(sqliteArray(args[0]), sqliteArray(args[1])...)
	unique := map[string]any{}
	for _, value := range items {
		encoded, _ := json.Marshal(value)
		key := string(encoded)
		object, _ := value.(map[string]any)
		if mode == "port" {
			key = fmt.Sprint(object["port"])
		}
		if mode == "params" {
			key = fmt.Sprint(object["location"]) + "\x00" + fmt.Sprint(object["name"])
		}
		if previous, exists := unique[key]; exists {
			if mode == "port" {
				old, _ := previous.(map[string]any)
				oldService, _ := old["service"].(string)
				newService, _ := object["service"].(string)
				if oldService != "" && newService == "" {
					continue
				}
			}
			if mode == "params" {
				old, _ := json.Marshal(previous)
				if string(old) > string(encoded) {
					continue
				}
			}
		}
		unique[key] = value
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]any, 0, len(keys))
	for _, key := range keys {
		result = append(result, unique[key])
	}
	return sqliteEncoded(result)
}
