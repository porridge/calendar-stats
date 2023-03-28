// calendar-tracker, a program to compute statistics from Google calendars.
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
	"sort"
	"time"

	"cloud.google.com/go/civil"
	"google.golang.org/api/calendar/v3"
)

type timeline struct {
	moments  map[time.Time][]thing
	location *time.Location
}

func newTimeline(location *time.Location) *timeline {
	return &timeline{
		moments:  make(map[time.Time][]thing),
		location: location,
	}
}

func (t *timeline) addEvent(event *calendar.Event, start, end time.Time) {
	t.moments[start] = append(t.moments[start], thing{what: eventStart, event: event})
	t.moments[end] = append(t.moments[end], thing{what: eventEnd, event: event})
}

func (t *timeline) addMidnightAt(midnightTime time.Time, date civil.Date) {
	t.moments[midnightTime] = append(t.moments[midnightTime], thing{what: midnight, newDay: &date})
}

func (t *timeline) addMidnight(date civil.Date) {
	localMidnight := date.In(t.location)
	t.addMidnightAt(localMidnight, date)
}

func (t *timeline) sortedMoments() []time.Time {
	keys := make([]time.Time, len(t.moments))
	i := 0
	for k := range t.moments {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })
	return keys
}

func (t *timeline) thingsAt(moment time.Time) []thing {
	return t.moments[moment]
}
