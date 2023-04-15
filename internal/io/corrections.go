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
package io

import (
	"context"
	"os"

	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v3"
)

type Corrections struct {
	Corrections []*Correction `yaml:"corrections"`
}

type Correction struct {
	Id        string `yaml:"id"`
	Summary   string `yaml:"summary"`
	Organizer string `yaml:"organizer"`
}

func LoadCorrections(fileName string) (*Corrections, error) {
	c := &Corrections{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func MaybeUpdateSummary(ctx context.Context, source, id, summary string) error {
	srv, err := newCalendarService(ctx)
	if err != nil {
		return err
	}
	_, err = srv.Events.Patch(source, id, &calendar.Event{Summary: summary}).SendUpdates("none").Do()
	return err
}

func SaveUnrecognized(correctionsFileName string, unrecognized []*calendar.Event) error {
	un := &Corrections{}
	for _, e := range unrecognized {
		organizer := e.Organizer.DisplayName
		if organizer == "" {
			organizer = e.Organizer.Email
		}
		un.Corrections = append(un.Corrections, &Correction{
			Id:        e.Id,
			Summary:   e.Summary,
			Organizer: organizer,
		})
	}
	data, err := yaml.Marshal(un)
	if err != nil {
		return err
	}
	return os.WriteFile(correctionsFileName, data, os.ModePerm)
}
