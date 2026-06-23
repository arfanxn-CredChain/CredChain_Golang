package overview

import (
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

func extractDateRange(q *domainQuery.Query) (time.Time, time.Time) {
	if q == nil {
		return defaultDateRange()
	}
	for _, f := range q.Filters {
		if f.Column == "date" && f.Operator == domainQuery.OperatorBetween && len(f.Values) == 2 {
			from, err1 := time.Parse("2006-01-02", f.Values[0])
			to, err2 := time.Parse("2006-01-02", f.Values[1])
			if err1 == nil && err2 == nil {
				return from, to.Add(24*time.Hour - time.Second)
			}
		}
	}
	return defaultDateRange()
}

func defaultDateRange() (time.Time, time.Time) {
	return time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
}
