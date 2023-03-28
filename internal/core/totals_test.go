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
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/calendar/v3"
)

func TestComputeTotals(t *testing.T) {
	type args struct {
		events *calendar.Events
	}

	tests := []struct {
		name string
		args args
		want map[civil.Date]time.Duration
	}{
		{
			name: "empty",
			args: args{events: &calendar.Events{
				Items: []*calendar.Event{},
			}},
			want: make(map[civil.Date]time.Duration),
		},
		{
			name: "separate events",
			args: args{events: &calendar.Events{
				Items: []*calendar.Event{
					newEvent("2023-03-25T13:00:00+01:00", "2023-03-25T13:30:00+01:00"),
					newEvent("2023-03-25T14:00:00+01:00", "2023-03-25T14:30:00+01:00"),
				},
			}},
			want: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 60 * time.Minute,
			},
		},
		{
			name: "overlapping events",
			args: args{events: &calendar.Events{
				Items: []*calendar.Event{
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
			}},
			want: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 3 * time.Hour,
			},
		},
		{
			name: "event spanning days",
			args: args{events: &calendar.Events{
				Items: []*calendar.Event{
					newEvent("2023-03-25T23:00:00+01:00", "2023-03-26T01:30:00+01:00"),
				},
			}},
			want: map[civil.Date]time.Duration{
				{Year: 2023, Month: 03, Day: 25}: 60 * time.Minute,
				{Year: 2023, Month: 03, Day: 26}: 90 * time.Minute,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ComputeTotals(tt.args.events))
		})
	}
}

func newEvent(startTime string, endTime string) *calendar.Event {
	x := &calendar.Event{
		Organizer: &calendar.EventOrganizer{Self: true},
		Start: &calendar.EventDateTime{
			DateTime: startTime,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
		},
	}
	return x
}
