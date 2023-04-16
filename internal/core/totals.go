// calendar-stats, a program to compute statistics from Google calendars.
// Copyright (C) 2023 Marcin Owsiany <marcin@owsiany.pl>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package core

import (
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/civil"
	"google.golang.org/api/calendar/v3"
)

type thingType int

const (
	eventStart thingType = iota
	eventEnd
	midnight
)

type thing struct {
	what   thingType
	event  *calendar.Event
	newDay *civil.Date
}

func ComputeTotals(events []*calendar.Event, categories []*Category, location *time.Location) (map[civil.Date]time.Duration, map[CategoryName]time.Duration, []*calendar.Event) {
	moments := computeTimeline(events, location)
	return categorizeTime(moments, categories)
}

func computeTimeline(events []*calendar.Event, currentLocation *time.Location) *timeline {
	t := newTimeline(currentLocation)

	for _, event := range events {
		isAccepted, evStart, evEnd := parseEvent(event)
		if !isAccepted {
			continue
		}
		t.addEvent(event, evStart, evEnd)

		for _, boundary := range []time.Time{evStart, evEnd} {
			boundaryDate := civil.DateOf(boundary)
			t.addMidnight(boundaryDate)
			t.addMidnight(boundaryDate.AddDays(1))
		}
	}
	return t
}

func parseEvent(event *calendar.Event) (bool, time.Time, time.Time) {
	if !shouldConsider(event) {
		return false, time.Time{}, time.Time{}
	}
	evStart, err := time.Parse(time.RFC3339, event.Start.DateTime)
	if err != nil {
		// TODO: Use a logging library
		fmt.Printf("Failed to parse start time [%s] of event %v\n", event.Start.DateTime, event.Summary)
		return false, time.Time{}, time.Time{}
	}
	evEnd, err := time.Parse(time.RFC3339, event.End.DateTime)
	if err != nil {
		fmt.Printf("Failed to parse end time [%s] of event %v\n", event.End.DateTime, event.Summary)
		return false, time.Time{}, time.Time{}
	}
	evStart, evEnd = stretchSpeedyMeetings(evStart, evEnd)
	return true, evStart, evEnd
}

func shouldConsider(event *calendar.Event) bool {
	if event.Start.DateTime == "" {
		// full-day event
		return false
	}
	if event.EventType == "outOfOffice" || event.EventType == "workingLocation" {
		return false
	}
	for _, attendee := range event.Attendees {
		if attendee.Self && attendee.ResponseStatus == "declined" {
			return false
		}
	}
	if event.Organizer != nil && event.Organizer.Self {
		return true
	}
	if event.Creator != nil && event.Creator.Self {
		return true
	}
	for _, attendee := range event.Attendees {
		if attendee.Self {
			return attendee.ResponseStatus == "accepted"
		}
	}
	log.Printf("self not found among attendees of %+v %+v", event.Organizer, event.Creator)
	return false
}

// stretchSpeedyMeetings delays endTime if needed.
// Speedy meetings are a lie. They usually last until the full half hour anyway.
func stretchSpeedyMeetings(evStart, evEnd time.Time) (time.Time, time.Time) {
	d := evEnd.Sub(evStart)
	if d == 50*time.Minute {
		return evStart, evStart.Add(1 * time.Hour)
	} else if d == 40*time.Minute {
		return evStart, evStart.Add(45 * time.Minute)
	} else if d == 25*time.Minute {
		return evStart, evStart.Add(30 * time.Minute)
	} else {
		return evStart, evEnd
	}
}

// categorizeTime returns three values. A map from civil date to time spent on it,
// a map from category name to time spent on it, and a slice of unrecognized calendar events.
func categorizeTime(t *timeline, categories []*Category) (map[civil.Date]time.Duration, map[CategoryName]time.Duration, []*calendar.Event) {
	momentTimes := t.sortedMoments()
	dayTotals := make(map[civil.Date]time.Duration)
	categoryTotals := make(map[CategoryName]time.Duration)
	unrecognized := []*calendar.Event{}
	currentTasks := newSpan(categories)

	for _, momentTime := range momentTimes {
		currentTasks.checkpoint(dayTotals, categoryTotals, momentTime)
		for _, thing := range t.thingsAt(momentTime) {
			switch thing.what {
			case midnight:
				continue
			case eventEnd:
				currentTasks.eventEnd(thing.event)
			case eventStart:
				if ok := currentTasks.eventStart(thing.event); !ok {
					unrecognized = append(unrecognized, thing.event)
				}
			}
		}
	}
	return dayTotals, categoryTotals, unrecognized
}
