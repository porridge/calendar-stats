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
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/porridge/calendar-stats/internal/config"
	"github.com/porridge/calendar-stats/internal/core"
	"github.com/porridge/calendar-stats/internal/flags"
	"github.com/porridge/calendar-stats/internal/io"
	"github.com/porridge/calendar-stats/internal/ordererd"
	"github.com/snabb/isoweek"
	"google.golang.org/api/calendar/v3"
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
	configFile := flag.String("config", "config.yaml", "Name of configuration file to read.")
	source := flag.String("source", "primary", "Name of Google Calendar to read.")
	weekCount := flag.Int("weeks", 0, "Shortcut way to set -start to beginning of week this many weeks before the current one. If set to non-zero value, takes precedecnce over -start.")
	end := time.Now()
	start := getWeekStart(0, end)
	flag.Var(flags.TimeValue(&start), "start", "Start time. Defaults to beginning of current week. Use any unambiguous format supported by https://github.com/araddon/dateparse")
	flag.Var(flags.TimeValue(&end), "end", "End time. Defaults to now. Use any unambiguous format supported by https://github.com/araddon/dateparse")

	cacheFileName := flag.String("cache", "", "If not empty, name of json file to use as event cache. "+
		"If file does not exist, it will be created and fetched events will be stored there. "+
		"Otherwise, events will be loaded from this file rather than fetched from Google Calendar.")
	decimalOutput := flag.Bool("decimal-output", false, "If true, print daily totals as decimal fractions rather than XhYmZs Duration format.")
	classificationDetails := flag.Bool("classification-details", false, "If true, print to which category each event was classified.")
	correctionsFileName := flag.String("corrections", "", "Name of file to: apply event summary corrections from at start, and save unrecognized events to at the end.")

	flags.Parse(notice)

	if *weekCount != 0 {
		start = getWeekStart(*weekCount, end)
	}

	ctx := context.Background()
	err := maybeApplyCorrections(ctx, *source, *correctionsFileName)
	if err != nil {
		log.Fatalf("Failed to apply corrections: %s", err)
	}
	events, err := io.GetEvents(ctx, *source, start, end, *cacheFileName)
	if err != nil {
		log.Fatalf("Failed to retrieve events: %s", err)
	}
	if len(events) == 0 {
		fmt.Println("No events found.")
		return
	}
	categories, err := config.Read(*configFile)
	if os.IsNotExist(err) {
		log.Printf("Could not read config file %q, cannot categorize events: %s", *configFile, err)
	} else if err != nil {
		log.Fatalf("Could not read config file %q: %s", *configFile, err)
	}

	unrecognized := analyzeAndPrint(events, categories, *decimalOutput, *classificationDetails)

	if *correctionsFileName != "" {
		err = io.SaveUnrecognized(*correctionsFileName, unrecognized)
		if err != nil {
			log.Fatalf("Failed to save unrecognized events: %s", err)
		}
	}
}

func analyzeAndPrint(events []*calendar.Event, categories []*core.Category, decimalOutput bool, classificationDetails bool) []*calendar.Event {
	dayTotals, categoryTotals, categoryDetails, unrecognized := core.ComputeTotals(events, categories, time.Local)
	days := ordererd.KeysOfMap(dayTotals, ordererd.CivilDates)
	var total time.Duration
	if len(days) > 0 {
		fmt.Println("Time spent per day:")
	}
	for _, day := range days {
		total += dayTotals[day]
		value := formatDayTotal(decimalOutput, dayTotals[day])
		fmt.Printf("%v: %s\n", day, value)
	}
	if len(categories) == 0 {
		return unrecognized
	}
	fmt.Println("Time spent per category:")
	for _, category := range categories {
		catName := category.Name
		val := categoryTotals[catName]
		fraction := (float64(val) / float64(total)) * 100
		if catName == core.Uncategorized {
			catName = "(uncategorized)"
		}
		fmt.Printf("%4.1f%% %s\n", fraction, catName)
		if classificationDetails {
			for _, eventSummary := range categoryDetails[catName] {
				fmt.Println(" -", eventSummary)
			}
		}
	}
	if len(unrecognized) > 0 {
		fmt.Println("Unrecognized:")
		for _, un := range unrecognized {
			fmt.Println(formatUnrecognizedEvent(un))
		}
	}
	return unrecognized
}

// getWeekStart returns the time of beginning of week that is weekCount weeks before end.
func getWeekStart(weekCount int, end time.Time) time.Time {
	weekCountDuration := time.Hour * 24 * 7 * time.Duration(weekCount)
	year, week := end.Add(-weekCountDuration).ISOWeek()
	return isoweek.StartTime(year, week, time.Local)
}

func maybeApplyCorrections(ctx context.Context, source, correctionsFileName string) error {
	if correctionsFileName == "" {
		return nil
	}
	corrections, err := io.LoadCorrections(correctionsFileName)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(corrections.Corrections) == 0 {
		return nil
	}
	log.Printf("Updating summary of %d events...\n", len(corrections.Corrections))
	for _, correction := range corrections.Corrections {
		err = io.MaybeUpdateSummary(ctx, source, correction.Id, correction.Summary)
		if err != nil {
			return fmt.Errorf("failed to update summary of event %q: %w", correction.Id, err)
		}
	}
	log.Println("Summaries updated.")
	return nil
}

func formatUnrecognizedEvent(event *calendar.Event) string {
	if event.Start == nil || event.End == nil {
		return "?"
	}
	start, err1 := time.Parse(time.RFC3339, event.Start.DateTime)
	end, err2 := time.Parse(time.RFC3339, event.End.DateTime)
	if err1 != nil || err2 != nil {
		return "?"
	}
	return fmt.Sprintf("%s %10s  %s", event.Start.DateTime, end.Sub(start).String(), event.Summary)
}

func formatDayTotal(decimalOutput bool, d time.Duration) string {
	if decimalOutput {
		return fmt.Sprintf("%f", float64(d)/float64(time.Hour))
	} else {
		return d.String()
	}
}
