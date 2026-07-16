package engine

import (
	"regexp"
	"strings"
)

var (
	reCreateTable = regexp.MustCompile(`(?is)\bCREATE\s+TABLE\b.*?\bAS\b.*?;`)
	reWith        = regexp.MustCompile(`(?is)\bWITH\b.*?;`)
	reSelect      = regexp.MustCompile(`(?is)\bSELECT\b.*?;`)
	reCodeSQL     = regexp.MustCompile("(?is)```sql\\s*(.*?)```")
	reCodeAny     = regexp.MustCompile("(?is)```\\s*(.*?)```")
)

func ExtractSQL(response string) string {
	if sql := lastMatch(reCreateTable, response); sql != "" {
		return strings.TrimSpace(sql)
	}
	if sql := lastMatch(reWith, response); sql != "" {
		return strings.TrimSpace(sql)
	}
	if sql := lastMatch(reSelect, response); sql != "" {
		return strings.TrimSpace(sql)
	}
	if m := reCodeSQL.FindAllStringSubmatch(response, -1); len(m) > 0 {
		return strings.TrimSpace(m[len(m)-1][1])
	}
	if m := reCodeAny.FindAllStringSubmatch(response, -1); len(m) > 0 {
		return strings.TrimSpace(m[len(m)-1][1])
	}
	return strings.TrimSpace(response)
}

func IsIntermediateSQL(sql string) bool {
	return strings.Contains(strings.ToLower(sql), "intermediate_sql")
}

func lastMatch(re *regexp.Regexp, s string) string {
	matches := re.FindAllString(s, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func IsSQLValid(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "INSERT") ||
		strings.HasPrefix(upper, "UPDATE") ||
		strings.HasPrefix(upper, "DELETE") ||
		strings.HasPrefix(upper, "CREATE") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "DESCRIBE")
}
