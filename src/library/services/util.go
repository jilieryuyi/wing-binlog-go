package services

import "regexp"

func MatchFilters(filters []string, table string) bool {
	if filters == nil || len(filters) <= 0 {
		return true
	}
	for _, f := range filters {
		match, err := regexp.MatchString(f, table)
		if match && err == nil {
			return true
		}
	}
	return false
}
