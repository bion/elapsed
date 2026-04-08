package internal

import (
	"fmt"
	"log"
	"math"
	"time"
)

type DayHistory struct {
	Event       Event
	firstSunday time.Time
	grid        [][]time.Time
}

func (dh DayHistory) PrintGrid() {
	if !dh.HasHistory() {
		return
	}
	fmt.Println("***** " + dh.Event.Title)

	grid := dh.DayGrid()

	for i, row := range grid {
		switch i {
		case 0:
			fmt.Print("SUN: ")
		case 1:
			fmt.Print("MON: ")
		case 2:
			fmt.Print("TUE: ")
		case 3:
			fmt.Print("WED: ")
		case 4:
			fmt.Print("THU: ")
		case 5:
			fmt.Print("FRI: ")
		case 6:
			fmt.Print("SAT: ")
		}

		for _, day := range row {
			if isToday(day) {
				if day.IsZero() {
					fmt.Print("t")
				} else {
					fmt.Print("T")
				}
			} else if day.IsZero() {
				fmt.Print("-")
			} else {
				fmt.Print("X")
			}
		}
		fmt.Println()
	}
}

func setToDayStart(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Now().Location())
}

func sum[T float32 | float64 | int | int64](s []T) T {
	var total T
	for _, n := range s {
		total += n
	}
	return total
}

func (dh DayHistory) CenteredAveragesGrid() [][]float32 {
	averages := dh.CenteredAverages()
	grid := make([][]float32, 7)
	numWeeks := dh.NumWeeks()

	for i := range grid {
		grid[i] = make([]float32, numWeeks)
	}

	for i, v := range averages {
		week := i / 7
		day := i % 7
		grid[day][week] = v
	}

	return grid
}

func (dh DayHistory) CenteredAverages() []float32 {
	totalDays := int(math.Ceil(time.Since(dh.GetFirstSunday()).Hours() / 24))
	dayCounts := make([]float32, totalDays)

	startDay := setToDayStart(dh.GetFirstSunday())

	for _, occ := range dh.Event.Occurrences {
		o := time.UnixMilli(occ.TimeUnixMillis)
		offset := int(o.Sub(startDay).Hours() / 24)
		dayCounts[offset]++
	}

	averages := make([]float32, totalDays)
	for i := range len(averages) {
		if i == 0 {
			averages[i] = sum(dayCounts[0:2]) / 2
		} else if i == len(averages)-1 {
			averages[i] = sum(dayCounts[i-1:i]) / 2
		} else {
			averages[i] = sum(dayCounts[i-1:i+2]) / 3
		}
	}

	return averages
}

func (dh DayHistory) DayGrid() [][]time.Time {
	if dh.grid != nil {
		return dh.grid
	}

	grid := make([][]time.Time, 7)
	numWeeks := dh.NumWeeks()

	for i := range grid {
		grid[i] = make([]time.Time, numWeeks)
	}

	for _, occ := range dh.Event.Occurrences {
		o := time.UnixMilli(occ.TimeUnixMillis)
		week := int(math.Floor(o.Sub(dh.GetFirstSunday()).Hours() / (24 * 7)))
		weekday := int(o.Weekday())

		grid[weekday][week] = o
	}

	return grid
}

func (dh DayHistory) NumWeeks() int {
	d := time.Since(dh.GetFirstSunday())
	return int(math.Ceil(d.Hours() / (24 * 7)))
}

func (dh DayHistory) HasHistory() bool {
	return len(dh.Event.Occurrences) > 0
}

func (dh DayHistory) GetFirstSunday() time.Time {
	if !dh.firstSunday.IsZero() {
		return dh.firstSunday
	}

	if !dh.HasHistory() {
		log.Fatalf("Event %d has no history!", dh.Event.Id)
	}

	t := time.UnixMilli(dh.Event.Occurrences[len(dh.Event.Occurrences)-1].TimeUnixMillis)
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	weekday := t.Weekday()

	daysSinceSunday := int(weekday)
	dh.firstSunday = t.AddDate(0, 0, -daysSinceSunday)

	return dh.firstSunday
}

func isToday(t time.Time) bool {
	now := time.Now()
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
