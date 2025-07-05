package calendar

import (
	"embed"
	"encoding/json"
	"time"
)

const (
	startTradeYear int = 2021
	endTradeYear   int = 2025
)

type TradeDay struct {
	StartTime time.Time
	EndTime   time.Time
}

func (t TradeDay) GetStartDate() time.Time {
	return time.Date(t.StartTime.Year(), t.StartTime.Month(), t.StartTime.Day(), 0, 0, 0, 0, time.Local)
}

func (t TradeDay) GetEndDate() time.Time {
	return time.Date(t.EndTime.Year(), t.EndTime.Month(), t.EndTime.Day(), 0, 0, 0, 0, time.Local)
}

type Calendar interface {
	GetFutureTradeDay() TradeDay
	GetFutureLastTradeDay() TradeDay

	GetStockTradeDay() TradeDay
	GetStockLastTradeDay() TradeDay
}

//go:embed holiday.json
var files embed.FS

type calendar struct {
	holidayTimeMap map[time.Time]struct{}
	tradeDayMap    map[time.Time]struct{}
}

func NewCalendar() Calendar {
	t := &calendar{
		holidayTimeMap: make(map[time.Time]struct{}),
		tradeDayMap:    make(map[time.Time]struct{}),
	}
	t.fillHoliday()
	t.fillTradeDay()
	return t
}

func (t *calendar) GetStockTradeDay() TradeDay {
	var nowTime time.Time
	if time.Now().Hour() >= 14 {
		nowTime = time.Now().AddDate(0, 0, 1)
	} else {
		nowTime = time.Now()
	}
	d := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, time.Local)
	var startTime, endTime time.Time
	for {
		if t.isTradeDay(d) {
			startTime = d.Add(9 * time.Hour)
			endTime = d.Add(13 * time.Hour).Add(30 * time.Minute)
			break
		}
		d = d.AddDate(0, 0, 1)
	}
	return TradeDay{startTime, endTime}
}

func (t *calendar) GetStockLastTradeDay() TradeDay {
	firstDay := t.GetStockTradeDay().GetStartDate()
	for {
		if t.isTradeDay(firstDay.AddDate(0, 0, -1)) {
			startTime := firstDay.AddDate(0, 0, -1).Add(9 * time.Hour)
			endTime := firstDay.AddDate(0, 0, -1).Add(13 * time.Hour).Add(30 * time.Minute)
			return TradeDay{startTime, endTime}
		}
		firstDay = firstDay.AddDate(0, 0, -1)
	}
}

func (t *calendar) GetFutureTradeDay() TradeDay {
	var nowTime time.Time
	if time.Now().Hour() >= 14 {
		nowTime = time.Now().AddDate(0, 0, 1)
	} else {
		nowTime = time.Now()
	}
	var startTime, endTime time.Time
	d := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, time.Local)
	for {
		if t.isTradeDay(d) {
			endTime = d.Add(13 * time.Hour).Add(45 * time.Minute)
			break
		}
		d = d.AddDate(0, 0, 1)
	}
	d = d.AddDate(0, 0, -1)
	for {
		if t.isTradeDay(d) {
			startTime = d.Add(15 * time.Hour)
			break
		}
		d = d.AddDate(0, 0, -1)
	}
	return TradeDay{startTime, endTime}
}

func (t *calendar) GetFutureLastTradeDay() TradeDay {
	firstDay := t.GetFutureTradeDay().GetStartDate()
	for {
		if t.isTradeDay(firstDay.AddDate(0, 0, -1)) {
			startTime := firstDay.AddDate(0, 0, -1).Add(15 * time.Hour)
			endTime := firstDay.Add(13 * time.Hour).Add(45 * time.Minute)
			return TradeDay{startTime, endTime}
		}
		firstDay = firstDay.AddDate(0, 0, -1)
	}
}

type holidayList struct {
	List []string `json:"list"`
}

func (t *calendar) fillHoliday() {
	tmp := holidayList{}
	content, err := files.ReadFile("holiday.json")
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}

	for _, v := range tmp.List {
		tm, pErr := time.ParseInLocation(time.DateOnly, v, time.Local)
		if pErr != nil {
			panic(pErr)
		}

		t.holidayTimeMap[tm] = struct{}{}
	}
}

func (t *calendar) fillTradeDay() {
	tm := time.Date(startTradeYear, 1, 1, 0, 0, 0, 0, time.Local)
	for {
		if tm.Year() > endTradeYear {
			break
		}
		if tm.Weekday() != time.Saturday && tm.Weekday() != time.Sunday && !t.isHoliday(tm) {
			t.tradeDayMap[tm] = struct{}{}
		}
		tm = tm.AddDate(0, 0, 1)
	}
}

func (t *calendar) isHoliday(date time.Time) bool {
	if _, ok := t.holidayTimeMap[date]; ok {
		return true
	}
	return false
}

func (t *calendar) isTradeDay(date time.Time) bool {
	if _, ok := t.tradeDayMap[date]; ok {
		return true
	}
	return false
}
