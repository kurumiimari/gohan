package api

import (
	"net/url"
	"strconv"
)

func GetIntFromQuery(query url.Values, key string, initial int) int {
	valStr := query.Get(key)
	if valStr == "" {
		return initial
	}
	valI, err := strconv.Atoi(valStr)
	if err != nil {
		return initial
	}
	return valI
}

func PaginationQuery(count, offset int) url.Values {
	return url.Values{
		"count":  []string{strconv.Itoa(count)},
		"offset": []string{strconv.Itoa(offset)},
	}
}
