package db

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func sqliteArrayCast(query string, ordinal int) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`(?i)\$%d::(bigint|text|jsonb|integer|int)\[\]`, ordinal))
	match := pattern.FindStringSubmatch(query)
	if len(match) == 0 {
		return ""
	}
	return strings.ToLower(match[1])
}
func sqliteReplace(query, pattern, replacement string) string {
	return regexp.MustCompile("(?i)"+pattern).ReplaceAllString(query, replacement)
}

// SQL literals are shielded before rewriting dialect tokens. Neither bound user
// values nor SQL string literals are interpreted as SQL by the compatibility layer.
func sqliteSQL(query string) (string, error) {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "LOCK TABLE ") {
		return "SELECT 1", nil
	}
	var literals []string
	query = regexp.MustCompile(`'(?:''|[^'])*'`).ReplaceAllStringFunc(query, func(literal string) string {
		literals = append(literals, literal)
		return fmt.Sprintf("__SW_STRING_%d__", len(literals)-1)
	})
	literal := func(token string) string {
		index, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(token, "__SW_STRING_"), "__"))
		if index < len(literals) {
			return literals[index]
		}
		return token
	}
	var err error
	query, err = sqliteRewriteRecords(query)
	if err != nil {
		return "", err
	}
	query, err = sqliteRewriteRowJSON(query)
	if err != nil {
		return "", err
	}
	query = sqliteReplace(query, `EXTRACT\(EPOCH FROM (MAX\([^()]+\)|[\w.]+)\)::bigint`, "CAST(sw_epoch($1) AS INTEGER)")
	query = sqliteReplace(query, `\bFOR (?:UPDATE|SHARE|KEY SHARE|NO KEY UPDATE)(?: OF \w+(?:,\s*\w+)*)?(?: SKIP LOCKED)?`, "")
	query = sqliteReplace(query, `\bILIKE\b`, "LIKE")
	query = sqliteReplace(query, `\bON CONFLICT ON CONSTRAINT \w+`, "ON CONFLICT")
	query = sqliteReplace(query, `\b(?:BIGSERIAL|SERIAL) PRIMARY KEY`, "INTEGER PRIMARY KEY AUTOINCREMENT")
	query = sqliteReplace(query, `\bTIMESTAMPTZ\b`, "TIMESTAMP")
	query = sqliteReplace(query, `\bBYTEA\b`, "BLOB")
	query = sqliteReplace(query, `\bADD COLUMN IF NOT EXISTS\b`, "ADD COLUMN")
	query = sqliteReplace(query, `\bDEFAULT now\(\)`, "DEFAULT (sw_now())")
	query = sqliteReplace(query, `\bnow\(\)`, "sw_now()")
	query = sqliteReplace(query, `ARRAY\[(\$\d+)::jsonb\]`, "json_array(json($1))")
	query = sqliteReplace(query, `::(?:bigint|text|jsonb|json|integer|int|boolean|cidr|inet|double precision|numeric)(?:\[\])?`, "")
	query = sqliteReplace(query, `\b(?:JSONB|CIDR|INET)\b`, "TEXT")
	query = sqliteReplace(query, `\bpg_advisory(?:_xact)?_(?:lock|unlock)\([^)]*\)`, "1")
	query = sqliteReplace(query, `\bGREATEST\(`, "sw_greatest(")
	query = sqliteReplace(query, `\bLEAST\(`, "sw_least(")
	query = sqliteReplace(query, `\btry_inet\(`, "sw_try_inet(")
	query = sqliteReplace(query, `\bmasklen\(`, "sw_masklen(")
	query = sqliteReplace(query, `\bregexp_replace\(`, "sw_regexp_replace(")
	query = sqliteReplace(query, `\barray_to_json\(([^()]*)\)`, "($1)")
	query = sqliteReplace(query, `\b(?:cardinality|array_length)\(([^(),]+)(?:,\s*1)?\)`, "sw_array_length($1)")
	query = sqliteReplace(query, `\barray_append\(`, "sw_array_append(")
	query = sqliteReplace(query, `\barray_remove\(`, "sw_array_remove(")
	query = sqliteReplace(query, `\bjsonb_array_length\(`, "json_array_length(")
	query = sqliteReplace(query, `\bjsonb_typeof\(`, "json_type(")
	query = sqliteReplace(query, `\bjsonb_set\(`, "sw_json_set(")
	query = sqliteReplace(query, `\bjsonb_build_array\(`, "json_array(")
	query = sqliteReplace(query, `\bjsonb_build_object\(`, "json_object(")
	query = sqliteReplace(query, `\bjsonb_object_agg\(`, "json_group_object(")
	query = sqliteReplace(query, `\bjsonb_agg\(`, "json_group_array(")
	query = sqliteReplace(query, `\barray_agg\(`, "json_group_array(")
	query = sqliteReplace(query, `\bARRAY\[([^\]]*)\]`, "json_array($1)")
	query = sqliteReplace(query, `\bto_jsonb\((json_array\([^()]*\))\)`, "$1")
	// Array union subqueries in asset upserts have fixed documented keys.
	for {
		start := strings.Index(strings.ToUpper(query), "SELECT ARRAY(")
		if start < 0 {
			break
		}
		open := start + len("SELECT ARRAY")
		end := sqliteClosingParen(query, open)
		if end < 0 {
			return "", fmt.Errorf("unbalanced array expression")
		}
		body := query[open+1 : end]
		unnest := strings.Index(strings.ToLower(body), "unnest(")
		if unnest < 0 {
			return "", fmt.Errorf("unsupported array subquery")
		}
		argOpen := unnest + len("unnest")
		argEnd := sqliteClosingParen(body, argOpen)
		if argEnd < 0 {
			return "", fmt.Errorf("unbalanced unnest")
		}
		values := strings.SplitN(body[argOpen+1:argEnd], "||", 2)
		if len(values) != 2 {
			return "", fmt.Errorf("unsupported array union")
		}
		mode := "all"
		if strings.Contains(strings.ToLower(body), "open_ports") {
			mode = "port"
		}
		if strings.Contains(strings.ToLower(body), "params") {
			mode = "params"
		}
		query = query[:start] + fmt.Sprintf("sw_array_union(%s,%s,'%s')", values[0], values[1], mode) + query[end+1:]
	}
	query = sqliteReplace(query, `([\w.$]+)\s*=\s*ANY\(([^()]+)\)`, "sw_array_has($2,$1)")
	query = sqliteReplace(query, `([\w.]+)\s*>>=\s*(sw_try_inet\([^()]+\)|[\w.$]+)`, "sw_net_contains($1,$2)")
	query = sqliteReplace(query, `([\w.]+)\s*\?\s*([\w.$]+)`, "sw_json_has($1,$2)")
	query = sqliteReplace(query, `\(([\w.]+\s*->\s*__SW_STRING_\d+__)\)\s*\?\s*([\w.$]+)`, "sw_json_has($1,$2)")
	query = sqliteReplace(query, `([\w.]+)\s*@>\s*ANY\((\$\d+)\)`, "EXISTS(SELECT 1 FROM json_each($2) sw_contains WHERE sw_json_contains($1,sw_contains.value))")
	query = sqliteReplace(query, `([\w.]+)\s*@>\s*(json_array\([^()]*\)|[\w.$]+)`, "sw_json_contains($1,$2)")
	query = sqliteReplace(query, `\bUPDATE (\w+) (\w+)\s+SET`, "UPDATE $1 AS $2 SET")
	query = regexp.MustCompile(`(?i)sw_now\(\)\s*([+-])\s*make_interval\(secs\s*=>\s*([^()]+)\)`).ReplaceAllStringFunc(query, func(value string) string {
		match := regexp.MustCompile(`(?i)sw_now\(\)\s*([+-])\s*make_interval\(secs\s*=>\s*([^()]+)\)`).FindStringSubmatch(value)
		return "sw_time_add(sw_now()," + match[1] + "(" + match[2] + "))"
	})
	query = regexp.MustCompile(`(?i)sw_now\(\)\s*-\s*\((\$\d+)\s*\*\s*INTERVAL\s*(__SW_STRING_\d+__)\)`).ReplaceAllStringFunc(query, func(value string) string {
		match := regexp.MustCompile(`(?i)sw_now\(\)\s*-\s*\((\$\d+)\s*\*\s*INTERVAL\s*(__SW_STRING_\d+__)\)`).FindStringSubmatch(value)
		seconds := 86400
		if strings.Contains(literal(match[2]), "second") {
			seconds = 1
		}
		return fmt.Sprintf("sw_time_add(sw_now(),-(%s*%d))", match[1], seconds)
	})
	query = sqliteReplace(query, `\bAT TIME ZONE __SW_STRING_\d+__`, "")
	query = sqliteReplace(query, `TO_CHAR\(DATE_TRUNC\(__SW_STRING_\d+__,\s*([^()]+)\),\s*__SW_STRING_\d+__\)`, "sw_day($1)")
	query = sqliteReplace(query, `to_char\(([^()]+),\s*__SW_STRING_\d+__\)`, "sw_day($1)")
	query = sqliteReplace(query, `EXTRACT\(EPOCH FROM MAX\(([^()]+)\)\)`, "sw_epoch(MAX($1))")
	query = sqliteReplace(query, `EXTRACT\(EPOCH FROM ([\w.]+)\)`, "sw_epoch($1)")
	query = sqliteReplace(query, `EXTRACT\(EPOCH FROM \(sw_now\(\) - min\(([^()]+)\)\)\)`, "(sw_epoch(sw_now()) - sw_epoch(min($1)))")
	query = sqliteReplace(query, `date_trunc\((__SW_STRING_\d+__),\s*sw_now\(\)\)`, "sw_date_trunc($1,sw_now())")
	// Table-valued array operations preserve ordinality via json_each.key.
	query = sqliteReplace(query, `FROM unnest\(([^()]+)\) WITH ORDINALITY AS requested\(company_id, position\)`, "FROM (SELECT value AS company_id, key+1 AS position FROM json_each($1)) AS requested")
	query = sqliteReplace(query, `FROM (?:unnest|jsonb_array_elements_text)\(([^()]+)\)\s+(?:AS )?(\w+)\((\w+)\)`, "FROM (SELECT value AS $3 FROM json_each($1)) AS $2")
	query = sqliteReplace(query, `jsonb_array_elements_text\(([^()]+)\) AS elem`, "json_each($1) AS sw_elem")
	if strings.Contains(query, "AS sw_elem") {
		query = sqliteReplace(query, `\belem\b`, "sw_elem.value")
	}
	query = sqliteLateralJSON(query)
	query = sqliteReplace(query, `([\w.]+)\s*~\s*(__SW_STRING_\d+__)`, "sw_regex($1,$2)")
	// JSON concatenation is restricted to known JSON/array columns, never text.
	query = sqliteReplace(query, `\bto_jsonb\((\$\d+)\)`, "sw_to_json($1)")
	query = sqliteReplace(query, `\bto_jsonb\((json_array\([^()]*\))\)`, "$1")
	query = sqliteJSONConcat(query)
	query = sqliteReplace(query, `STRING_AGG\(DISTINCT (\w+),\s*__SW_STRING_\d+__\)`, "group_concat(DISTINCT $1)")
	for i, value := range literals {
		query = strings.ReplaceAll(query, fmt.Sprintf("__SW_STRING_%d__", i), value)
	}
	return query, nil
}

func sqliteClosingParen(value string, open int) int {
	depth := 0
	for i := open; i < len(value); i++ {
		if value[i] == '(' {
			depth++
		}
		if value[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func sqliteJSONConcat(query string) string {
	pattern := regexp.MustCompile(`(?i)\b((?:\w+\.)?(?:payload|extra|metadata|warnings|agents|audit))\s*\|\|\s*`)
	for {
		match := pattern.FindStringSubmatchIndex(query)
		if match == nil {
			return query
		}
		start := match[1]
		end := sqliteExpressionEnd(query, start)
		if end <= start {
			return query
		}
		left := query[match[2]:match[3]]
		query = query[:match[0]] + "sw_json_merge(" + left + "," + query[start:end] + ")" + query[end:]
		query = sqliteJSONFunctionConcat(query)
	}
}

func sqliteExpressionEnd(query string, start int) int {
	atom := regexp.MustCompile(`^[\w.$]+`).FindString(query[start:])
	if atom == "" {
		return start
	}
	end := start + len(atom)
	if strings.EqualFold(atom, "CASE") {
		depth := 0
		for _, match := range regexp.MustCompile(`(?i)\b(?:CASE|END)\b`).FindAllStringIndex(query[start:], -1) {
			if strings.EqualFold(query[start+match[0]:start+match[1]], "CASE") {
				depth++
			} else {
				depth--
				if depth == 0 {
					return start + match[1]
				}
			}
		}
		return start
	}
	if end < len(query) && query[end] == '(' {
		closing := sqliteClosingParen(query, end)
		if closing < 0 {
			return start
		}
		end = closing + 1
	}
	return end
}
func sqliteJSONFunctionConcat(query string) string {
	for {
		found := false
		for _, match := range regexp.MustCompile(`(?i)\b(?:sw_json_merge|json_object|json_array)\(`).FindAllStringIndex(query, -1) {
			end := sqliteClosingParen(query, match[1]-1)
			if end < 0 {
				continue
			}
			rest := query[end+1:]
			space := len(rest) - len(strings.TrimLeft(rest, " \n\t\r"))
			offset := end + 1 + space
			if !strings.HasPrefix(query[offset:], "||") {
				continue
			}
			start := offset + 2
			for start < len(query) && strings.ContainsRune(" \n\t\r", rune(query[start])) {
				start++
			}
			rightEnd := sqliteExpressionEnd(query, start)
			if rightEnd <= start {
				continue
			}
			query = query[:match[0]] + "sw_json_merge(" + query[match[0]:end+1] + "," + query[start:rightEnd] + ")" + query[rightEnd:]
			found = true
			break
		}
		if !found {
			return query
		}
	}
}

func sqliteLateralJSON(query string) string {
	pattern := regexp.MustCompile(`(?i)CROSS JOIN LATERAL jsonb_array_elements_text\(`)
	for {
		match := pattern.FindStringIndex(query)
		if match == nil {
			return query
		}
		end := sqliteClosingParen(query, match[1]-1)
		if end < 0 {
			return query
		}
		tail := query[end+1:]
		aliasMatch := regexp.MustCompile(`^\s+(\w+)`).FindStringSubmatchIndex(tail)
		if aliasMatch == nil {
			return query
		}
		alias := tail[aliasMatch[2]:aliasMatch[3]]
		query = query[:match[0]] + "CROSS JOIN json_each(" + query[match[1]:end] + ") AS sw_array_elem" + tail[aliasMatch[1]:]
		query = regexp.MustCompile(`(^|[^.\w"'])`+regexp.QuoteMeta(alias)+`\b`).ReplaceAllString(query, "${1}sw_array_elem.value")
	}
}
