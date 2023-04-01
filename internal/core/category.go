package core

import (
	"regexp"

	"google.golang.org/api/calendar/v3"
)

type CategoryName string

const Uncategorized = CategoryName("")

type Category struct {
	Name     CategoryName
	Patterns []*regexp.Regexp
}

func (c *Category) recognizes(event *calendar.Event) bool {
	for _, pattern := range c.Patterns {
		if pattern.MatchString(event.Summary) {
			return true
		}
	}
	return false
}
