// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timeutil

import (
	"fmt"
	"strings"
	"time"
)

type periodicStart struct {
	dayOfWeek      int
	hourOfDay      int
	minuteOfHour   int
	secondOfMinute int
}

// Periodic keeps track of a repeating period of time
type Periodic struct {
	start    *periodicStart
	duration time.Duration
}

// Period is a span of time from Start to End
type Period struct {
	Start time.Time
	End   time.Time
}

// ParsePeriodic returns a Periodic specified as a start and duration.
func ParsePeriodic(start, duration string) (*Periodic, error) {
	var err error

	pc := &Periodic{}

	if pc.start, err = parseStart(start); err != nil {
		return nil, fmt.Errorf("unable to parse start: %v", err)
	}

	if pc.duration, err = time.ParseDuration(duration); err != nil {
		return nil, fmt.Errorf("unable to parse duration: %v", err)
	}

	if pc.duration < time.Duration(0) {
		return nil, fmt.Errorf("duration cannot be negative")
	}

	// check that the duration of the window does not exceed the period.
	if (pc.start.dayOfWeek == -1 && pc.duration >= 24*time.Hour) || pc.duration >= 7*24*time.Hour {
		return nil, fmt.Errorf("duration cannot exceed period")
	}

	return pc, nil
}

var weekdays = map[string]int{
	"sun": int(time.Sunday),
	"mon": int(time.Monday),
	"tue": int(time.Tuesday),
	"wed": int(time.Wednesday),
	"thu": int(time.Thursday),
	"fri": int(time.Friday),
	"sat": int(time.Saturday),
}

// parseStart parses a string into a periodicStart.
func parseStart(start string) (*periodicStart, error) {
	ps := &periodicStart{}
	ps.dayOfWeek = -1
	f := strings.Fields(start)

	switch len(f) {
	case 1: // no day provided
	case 2:
		if dow, ok := weekdays[strings.ToLower(f[0])]; ok {
			ps.dayOfWeek = dow
		} else {
			return nil, fmt.Errorf("invalid day of week %q", f[0])
		}
		// shift
		f = f[1:]
	default:
		return nil, fmt.Errorf("wrong number of fields")
	}

	n, err := fmt.Sscanf(f[0], "%d:%d", &ps.hourOfDay, &ps.minuteOfHour)

	// check Sscanf failure
	if n != 2 || err != nil {
		return nil, fmt.Errorf("invalid time of day %q: %v", f[0], err)
	}

	// check hour range
	if ps.hourOfDay < 0 || ps.hourOfDay > 23 {
		return nil, fmt.Errorf("invalid time of day %q: hour must be >= 0 and <= 23", f[0])
	}

	// check minute range
	if ps.minuteOfHour < 0 || ps.minuteOfHour > 59 {
		return nil, fmt.Errorf("invalid time of day %q: minute must be >= 0 and <= 59", f[0])
	}

	return ps, nil
}

// DurationToStart returns the duration between the supplied time and the start
// of Periodic's relevant period.
// If we're in a period, a value <= 0 is returned, indicating how
// deep into period we are.
// If we're outside a period, a value > 0 is returned, indicating how long
// before the next period starts.
func (pc *Periodic) DurationToStart(ref time.Time) time.Duration {
	prev := pc.Previous(ref)
	if prev.End.After(ref) || prev.End.Equal(ref) {
		return prev.Start.Sub(ref)
	}
	return pc.Next(ref).Start.Sub(ref)
}

func (pc *Periodic) shiftTimeByDays(ref time.Time, daydiff int) time.Time {
	rt := time.Date(ref.Year(),
		ref.Month(),
		ref.Day()+daydiff,
		pc.start.hourOfDay,
		pc.start.minuteOfHour,
		pc.start.secondOfMinute,
		0,
		ref.Location())
	return rt
}

// Previous returns Periodic's previous Period occurrence relative to ref.
func (pc *Periodic) Previous(ref time.Time) (p *Period) {
	p = &Period{}
	if pc.start.dayOfWeek != -1 { // Weekly
		if pc.cmpDayOfWeek(ref) >= 0 {
			// this week
			p.Start = pc.shiftTimeByDays(ref, -(int(ref.Weekday()) - pc.start.dayOfWeek))
		} else {
			// last week
			p.Start = pc.shiftTimeByDays(ref, -(int(ref.Weekday()) + (7 - pc.start.dayOfWeek)))
		}
	} else if pc.start.hourOfDay != -1 { // Daily
		if pc.cmpHourOfDay(ref) >= 0 {
			// today
			p.Start = pc.shiftTimeByDays(ref, 0)
		} else {
			// yesterday
			p.Start = pc.shiftTimeByDays(ref, -1)
		}
	} // XXX(mischief): other intervals unsupported atm.

	p.End = p.Start.Add(pc.duration)
	return
}

// Next returns Periodic's next Period occurrence relative to ref.
func (pc *Periodic) Next(ref time.Time) (p *Period) {
	p = &Period{}
	if pc.start.dayOfWeek != -1 { // Weekly
		if pc.cmpDayOfWeek(ref) < 0 {
			// This week
			p.Start = pc.shiftTimeByDays(ref, pc.start.dayOfWeek-int(ref.Weekday()))
		} else {
			// Next week
			p.Start = pc.shiftTimeByDays(ref, (7-int(ref.Weekday()))+pc.start.dayOfWeek)
		}
	} else if pc.start.hourOfDay != -1 { // Daily
		if pc.cmpHourOfDay(ref) < 0 {
			// Today
			p.Start = pc.shiftTimeByDays(ref, 0)
		} else {
			// Tomorrow
			p.Start = pc.shiftTimeByDays(ref, 1)
		}
	} // XXX(mischief): other intervals unsupported atm.

	p.End = p.Start.Add(pc.duration)
	return
}

// cmpDayOfWeek compares ref to Periodic occurring in the same week as ref.
// The return value is less than, equal to, or greater than zero if ref occurs
// before, equal to, or after the start of Periodic within the same week.
func (pc *Periodic) cmpDayOfWeek(ref time.Time) time.Duration {
	pStart := pc.shiftTimeByDays(ref, -int(ref.Weekday())+pc.start.dayOfWeek)
	return ref.Sub(pStart)
}

// cmpHourOfDay compares ref to Periodic occurring in the same day as ref.
// The return value is less than, equal to, or greater than zero if ref occurs
// before, equal to, or after the start of Periodic in the same day.
func (pc *Periodic) cmpHourOfDay(ref time.Time) time.Duration {
	pStart := pc.shiftTimeByDays(ref, 0)
	return ref.Sub(pStart)
}
