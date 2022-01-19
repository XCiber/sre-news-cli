package news

import (
	"strings"
	"time"
)

type NasdaqHoliday struct {
	Date  string
	Title string
}

type NasdaqHolidays []NasdaqHoliday

func (nh *NasdaqHolidays) CheckDate(t time.Time) string {
	titles := make(map[string][]string)
	for _, holiday := range *nh {
		titles[holiday.Date] = append(titles[holiday.Date], holiday.Title)
	}
	keyDate := t.Format("02.01.2006")
	return strings.Join(titles[keyDate], ", ")
}
