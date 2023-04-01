package core

import (
	"time"

	"cloud.google.com/go/civil"
	"google.golang.org/api/calendar/v3"
)

type span struct {
	start      time.Time
	events     map[*calendar.Event]CategoryName
	categories []*Category
}

func newSpan(categories []*Category) *span {
	return &span{categories: categories, events: make(map[*calendar.Event]CategoryName)}
}

func (s *span) checkpoint(dayTotals map[civil.Date]time.Duration, categoryTotals map[CategoryName]time.Duration, end time.Time) {
	timeSpent := end.Sub(s.start)
	eventCount := len(s.events)
	var timePerEvent int64
	if eventCount > 0 {
		timePerEvent = int64(timeSpent) / int64(eventCount)
		dayTotals[civil.DateOf(s.start)] += timeSpent
	}
	for _, categoryName := range s.events {
		categoryTotals[categoryName] += time.Duration(timePerEvent)
	}
	s.start = end
}

func (s *span) eventEnd(event *calendar.Event) {
	delete(s.events, event)
}

// eventStart returns false if the event was not recognized to belong to a category.
func (s *span) eventStart(event *calendar.Event) bool {
	for _, aCategory := range s.categories {
		if aCategory.recognizes(event) {
			s.events[event] = aCategory.Name
			return true
		}
	}
	s.events[event] = Uncategorized
	return false
}
