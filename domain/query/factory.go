package query

func NewFilter(column string, operator Operator, values ...string) Filter {
	return Filter{
		Column:   column,
		Operator: operator,
		Values:   values,
	}
}

func NewSort(column string, order SortOrder) Sort {
	return Sort{
		Column: column,
		Order:  order,
	}
}
