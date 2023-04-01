package ordererd

import (
	"sort"

	"cloud.google.com/go/civil"
)

func CivilDates(keys []civil.Date, i, j int) bool {
	return keys[i].Before(keys[j])
}

func KeysOfMap[K comparable, V any](aMap map[K]V, lessFunc func(s []K, i, j int) bool) []K {
	keys := make([]K, len(aMap))
	i := 0
	for k := range aMap {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return lessFunc(keys, i, j) })
	return keys
}
