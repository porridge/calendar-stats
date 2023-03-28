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
package io

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/porridge/calendar-tracker/internal/auth"

	"github.com/snabb/isoweek"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func GetEvents(source string, weekCount int, cacheFilename string) (*calendar.Events, error) {
	ctx := context.Background()
	if cacheFilename != "" {
		events, err := readFromFile(cacheFilename)
		if err != nil {
			log.Printf("Failed to read events from %q, fetching them and saving first: %s", cacheFilename, err)
			events, err = fetchFromCalendar(ctx, source, weekCount)
			if err != nil {
				return nil, fmt.Errorf("unable to fetch events from calendar: %s", err)
			}
			if err = writeToFile(cacheFilename, events); err != nil {
				return nil, fmt.Errorf("failed to write events to %q: %s", cacheFilename, err)
			}
		}
		return events, nil
	} else {
		events, err := fetchFromCalendar(ctx, source, weekCount)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch events from calendar: %s", err)
		}
		return events, nil
	}
}

func writeToFile(s string, events *calendar.Events) error {
	eventsJson, err := json.Marshal(events)
	if err != nil {
		return err
	}
	return os.WriteFile(s, eventsJson, os.ModePerm)
}

func readFromFile(s string) (*calendar.Events, error) {
	events := &calendar.Events{}
	eventBytes, err := os.ReadFile(s)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(eventBytes, events)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func fetchFromCalendar(ctx context.Context, source string, weekCount int) (*calendar.Events, error) {
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := auth.GetClient(ctx, config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	t := time.Now().Add(time.Hour * (-24) * 7 * time.Duration(weekCount))
	year, week := t.ISOWeek()
	weekStart := isoweek.StartTime(year, week, time.Local)
	events, err := srv.Events.List(source).
		SingleEvents(true).
		TimeMin(weekStart.Format(time.RFC3339)).
		TimeMax(time.Now().Format(time.RFC3339)).
		// TODO: put some padding in time min/max to include
		// events which span week boundaries.
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve user's events: %v", err)
	}
	if events.NextPageToken != "" {
		return nil, fmt.Errorf("incomplete list of events returned, pagination support not implemented yet")
	}
	return events, nil
}
