package query

import (
	"fmt"
	"regexp"
	"strings"

	domainQuery "CredChain_Golang/domain/query"
)

var operatorMap map[string]domainQuery.Operator
var sortMap map[string]domainQuery.SortOrder
var operatorRegex *regexp.Regexp

func init() {
	operatorMap = buildOperatorMap()
	sortMap = buildSortMap()
	operatorRegex = buildOperatorRegex()
}

func buildOperatorMap() map[string]domainQuery.Operator {
	m := make(map[string]domainQuery.Operator)
	for _, op := range domainQuery.AllOperators {
		m[string(op)] = op
	}
	return m
}

func buildSortMap() map[string]domainQuery.SortOrder {
	m := make(map[string]domainQuery.SortOrder)
	for _, order := range domainQuery.AllSortOrders {
		m[string(order)] = order
		m[strings.ToLower(string(order))] = order
		m[strings.ToUpper(string(order))] = order
	}
	return m
}

func buildOperatorRegex() *regexp.Regexp {
	ops := make([]string, 0, len(domainQuery.AllOperators))
	for _, op := range domainQuery.AllOperators {
		ops = append(ops, string(op))
	}

	for i := 0; i < len(ops)-1; i++ {
		for j := i + 1; j < len(ops); j++ {
			if len(ops[i]) < len(ops[j]) {
				ops[i], ops[j] = ops[j], ops[i]
			}
		}
	}

	var patterns []string
	for _, op := range ops {
		patterns = append(patterns, regexp.QuoteMeta(op))
	}

	pattern := fmt.Sprintf(`^([a-zA-Z_][a-zA-Z0-9_]*)(%s)(.*)$`, strings.Join(patterns, "|"))
	return regexp.MustCompile(pattern)
}

func parseFilterString(filter string) (string, domainQuery.Operator, []string, error) {
	if filter == "" {
		return "", "", nil, fmt.Errorf("filter is empty")
	}

	matches := operatorRegex.FindStringSubmatch(filter)

	if len(matches) != 4 {
		return "", "", nil, fmt.Errorf("filter has invalid syntax")
	}

	column := matches[1]
	opStr := matches[2]
	valuesStr := strings.TrimSpace(matches[3])

	operator, ok := operatorMap[opStr]
	if !ok {
		return "", "", nil, fmt.Errorf("unsupported operator: %s", opStr)
	}

	if operator == domainQuery.OperatorNull || operator == domainQuery.OperatorNotNull {
		return column, operator, nil, nil
	}

	if operator == domainQuery.OperatorBetween || operator == domainQuery.OperatorNotBetween {
		var values []string
		if strings.Contains(strings.ToLower(valuesStr), " and ") {
			parts := regexp.MustCompile(`(?i)\s+and\s+`).Split(valuesStr, -1)
			for _, p := range parts {
				values = append(values, strings.TrimSpace(p))
			}
		} else {
			values = strings.Split(valuesStr, ",")
		}
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		if len(values) != 2 {
			return "", "", nil, fmt.Errorf("BETWEEN operator requires exactly 2 values")
		}
		return column, operator, values, nil
	}

	if operator == domainQuery.OperatorIn || operator == domainQuery.OperatorNotIn {
		if valuesStr == "" {
			return "", "", nil, fmt.Errorf("IN operator requires at least one value")
		}
		values := strings.Split(valuesStr, ",")
		nonEmpty := []string{}
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v != "" {
				nonEmpty = append(nonEmpty, v)
			}
		}
		if len(nonEmpty) == 0 {
			return "", "", nil, fmt.Errorf("IN operator requires at least one value")
		}
		return column, operator, nonEmpty, nil
	}

	if valuesStr == "" {
		return "", "", nil, fmt.Errorf("operator %s requires a value", opStr)
	}

	return column, operator, []string{valuesStr}, nil
}

func parseSortString(sort string) (string, domainQuery.SortOrder, error) {
	if sort == "" {
		return "", "", fmt.Errorf("sort is empty")
	}

	column := strings.TrimSpace(sort)
	order := domainQuery.SortAsc

	if strings.HasPrefix(column, "-") {
		column = strings.TrimPrefix(column, "-")
		order = domainQuery.SortDesc
	} else if strings.HasPrefix(column, "+") {
		column = strings.TrimPrefix(column, "+")
		order = domainQuery.SortAsc
	}

	if column == "" {
		return "", "", fmt.Errorf("column name is empty")
	}

	return column, order, nil
}
