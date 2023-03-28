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
package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"cloud.google.com/go/civil"
	"github.com/porridge/calendar-tracker/internal/core"
	"github.com/porridge/calendar-tracker/internal/io"
)

var notice string = `
Copyright (C) 2023 Marcin Owsiany <marcin@owsiany.pl>
Copyright (C) Google Inc.
Copyright (C) 2011 Google LLC.
This program comes with ABSOLUTELY NO WARRANTY.
This is free software, and you are welcome to redistribute it under the terms
of the terms of the GNU General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version.

Google Calendar is a trademark of Google LLC.
`


func main() {
	source := flag.String("source", "primary", "Name of Google Calendar to read.")
	weekCount := flag.Int("weeks", 0, "How many weeks before the current one to look at.")
	cacheFileName := flag.String("cache", "", "If not empty, name of json file to use as event cache. "+
		"If file does not exist, it will be created and fetched events will be stored there. "+
		"Otherwise, events will be loaded from this file rather than fetched from Google Calendar.")
	decimalOutput := flag.Bool("decimal-output", false, "If true, print totals as decimal fractions rather than XhYmZs Duration format.")

	origUsage := flag.Usage
	flag.Usage = func() {
		origUsage()
		fmt.Fprint(flag.CommandLine.Output(), notice)
	}

	flag.Parse()
	events, err := io.GetEvents(*source, *weekCount, *cacheFileName)
	if err != nil {
		log.Fatalf("Failed to retrieve events: %s", err)
	}

	fmt.Println("Past events this week:")
	if len(events.Items) == 0 {
		fmt.Println("No events found.")
	} else {
		dayTotals := core.ComputeTotals(events)
		days := sortedKeysOf(dayTotals)
		for _, day := range days {
			value := newFunction(decimalOutput, dayTotals[day])
			fmt.Printf("%v: %s\n", day, value)
		}
	}
}

func newFunction(decimalOutput *bool, d time.Duration) string {
	if *decimalOutput {
		return fmt.Sprintf("%f", float64(d)/float64(time.Hour))
	} else {
		return d.String()
	}
}

func sortedKeysOf[V any](aMap map[civil.Date]V) []civil.Date {
	keys := make([]civil.Date, len(aMap))
	i := 0
	for k := range aMap {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })
	return keys
}
