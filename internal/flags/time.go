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
package flags

import (
	"flag"
	"time"

	"github.com/araddon/dateparse"
)

func TimeValue(t *time.Time) flag.Value {
	return &timeValue{t}
}

type timeValue struct {
	Time *time.Time
}

func (v timeValue) String() string {
	if v.Time == nil {
		return ""
	}
	return v.Time.Format(time.RFC3339)
}

func (v timeValue) Set(str string) error {
	if t, err := dateparse.ParseStrict(str); err != nil {
		return err
	} else {
		*v.Time = t
	}
	return nil
}
