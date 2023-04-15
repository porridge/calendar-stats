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
	"regexp"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/calendar/v3"
)

func TestComputeTotals(t *testing.T) {
	type args struct {
		events     []*calendar.Event
		categories []*Category
	}

	tests := []struct {
		name             string
		args             args
		wantTotals       map[civil.Date]time.Duration
		wantCategories   map[CategoryName]time.Duration
		wantUnrecognized []*calendar.Event
	}{
		{
			name:             "empty",
			args:             args{events: []*calendar.Event{}},
			wantTotals:       make(map[civil.Date]time.Duration),
			wantCategories:   map[CategoryName]time.Duration{},
			wantUnrecognized: []*calendar.Event{},
		},
		{
			name: "separate events",
			args: args{
				events: []*calendar.Event{
					newEvent("2023-03-25T13:00:00+01:00", "2023-03-25T13:30:00+01:00"),
					newEvent("2023-03-25T14:00:00+01:00", "2023-03-25T14:15:00+01:00", "m/s"),
				},
				categories: []*Category{
					{
						Name: "communications",
						Patterns: []*regexp.Regexp{
							regexp.MustCompile("m/s"),
						},
					},
				},
			},
			wantTotals: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 45 * time.Minute,
			},
			wantCategories: map[CategoryName]time.Duration{
				"communications": 15 * time.Minute,
				"":               30 * time.Minute,
			},
			wantUnrecognized: []*calendar.Event{
				newEvent("2023-03-25T13:00:00+01:00", "2023-03-25T13:30:00+01:00"),
			},
		},
		{
			name: "overlapping events",
			args: args{events: []*calendar.Event{
				// aligned at end, 30m total
				newEvent("2023-03-25T13:00:00+01:00", "2023-03-25T13:30:00+01:00"),
				newEvent("2023-03-25T13:15:00+01:00", "2023-03-25T13:30:00+01:00"),
				// aligned at beginning, 30m total
				newEvent("2023-03-25T14:00:00+01:00", "2023-03-25T14:30:00+01:00"),
				newEvent("2023-03-25T14:00:00+01:00", "2023-03-25T14:15:00+01:00"),
				// aligned at both ends, 30m total
				newEvent("2023-03-25T15:00:00+01:00", "2023-03-25T15:30:00+01:00"),
				newEvent("2023-03-25T15:00:00+01:00", "2023-03-25T15:30:00+01:00"),
				// one completely covered, 45m total
				newEvent("2023-03-25T16:15:00+01:00", "2023-03-25T16:30:00+01:00"),
				newEvent("2023-03-25T16:00:00+01:00", "2023-03-25T16:45:00+01:00"),
				// partially overlapping, 45m total
				newEvent("2023-03-25T17:00:00+01:00", "2023-03-25T17:30:00+01:00"),
				newEvent("2023-03-25T17:15:00+01:00", "2023-03-25T17:45:00+01:00"),
			},
			},
			wantTotals: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 3 * time.Hour,
			},
		},
		{
			name: "event spanning days",
			args: args{events: []*calendar.Event{
				newEvent("2023-03-25T23:00:00+01:00", "2023-03-26T01:30:00+01:00"),
			},
			},
			wantTotals: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 60 * time.Minute,
				{Year: 2023, Month: 03, Day: 26}: 90 * time.Minute,
			},
		},
		{
			name: "parallel events",
			args: args{
				events: []*calendar.Event{
					newEvent("2023-03-25T11:00:00+01:00", "2023-03-25T11:30:00+01:00", "m/s"),
					newEvent("2023-03-25T11:00:00+01:00", "2023-03-25T11:30:00+01:00", "rev PR"),
				},
				categories: []*Category{
					{
						Name: "communications",
						Patterns: []*regexp.Regexp{
							regexp.MustCompile("m/s"),
						},
					},
					{
						Name: "reviews",
						Patterns: []*regexp.Regexp{
							regexp.MustCompile("rev"),
						},
					},
				},
			},
			wantTotals: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 30 * time.Minute,
			},
			wantCategories: map[CategoryName]time.Duration{
				"communications": 15 * time.Minute,
				"reviews":        15 * time.Minute,
			},
			wantUnrecognized: []*calendar.Event{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotals, gotCategories, gotUnrecognized := ComputeTotals(tt.args.events, tt.args.categories)
			assert.Equal(t, tt.wantTotals, gotTotals)
			if tt.wantCategories != nil {
				assert.Equal(t, tt.wantCategories, gotCategories)
			}
			if tt.wantUnrecognized != nil {
				assert.Equal(t, tt.wantUnrecognized, gotUnrecognized)
			}
		})
	}
}

func newEvent(startTime string, endTime string, title ...string) *calendar.Event {
	e := &calendar.Event{
		Organizer: &calendar.EventOrganizer{Self: true},
		Start: &calendar.EventDateTime{
			DateTime: startTime,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
		},
	}
	if len(title) > 0 {
		e.Summary = title[0]
	}
	return e
}
