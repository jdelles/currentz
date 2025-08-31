package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jdelles/currentz/internal/database"
)

type Recurring = database.RecurringTransactions

type RecurringInput struct {
	Description string
	Type        string
	Amount      float64
	StartDate   time.Time
	Interval    string
	DayOfWeek   *int
	DayOfMonth  *int
	EndDate     *time.Time
	Active      bool
}

func (fs *FinanceService) CreateRecurringSimple(ctx context.Context, in RecurringInput) (Recurring, error) {
	ival, err := parseIntervalEnum(in.Interval)
	if err != nil {
		return Recurring{}, err
	}

	var dow, dom pgtype.Int4
	if in.DayOfWeek != nil {
		dow = pgtype.Int4{Int32: int32(*in.DayOfWeek), Valid: true}
	}
	if in.DayOfMonth != nil {
		dom = pgtype.Int4{Int32: int32(*in.DayOfMonth), Valid: true}
	}
	var end pgtype.Date
	if in.EndDate != nil {
		end = makePgDate(*in.EndDate)
	}

	params := database.CreateRecurringParams{
		Description: in.Description,
		Type:        in.Type,
		Amount:      makePgNumeric(in.Amount),
		StartDate:   makePgDate(in.StartDate),
		Interval:    ival,
		DayOfWeek:   dow,
		DayOfMonth:  dom,
		EndDate:     end,
		Active:      in.Active,
	}
	return fs.db.CreateRecurring(ctx, params)
}

func (fs *FinanceService) CreateRecurring(ctx context.Context, r database.CreateRecurringParams) (Recurring, error) {
	return fs.db.CreateRecurring(ctx, r)
}
func (fs *FinanceService) ListRecurring(ctx context.Context) ([]Recurring, error) {
	return fs.db.ListRecurring(ctx)
}
func (fs *FinanceService) DeleteRecurring(ctx context.Context, id int32) error {
	return fs.db.DeleteRecurring(ctx, id)
}
func (fs *FinanceService) SetRecurringActive(ctx context.Context, id int32, active bool) error {
	return fs.db.SetRecurringActive(ctx, database.SetRecurringActiveParams{ID: id, Active: active})
}

func (fs *FinanceService) ExpandRecurringBetween(ctx context.Context, start, end time.Time) ([]Transaction, error) {
	rs, err := fs.db.ListActiveRecurring(ctx)
	if err != nil {
		return nil, err
	}

	var out []Transaction
	for _, r := range rs {
		occ := expandOne(r, start, end)
		out = append(out, occ...)
	}
	return out, nil
}

func expandOne(r Recurring, start, end time.Time) []Transaction {
	if r.StartDate.Time.After(end) {
		return nil
	}
	if r.EndDate.Valid && r.EndDate.Time.Before(start) {
		return nil
	}

	winStart := maxDate(start, r.StartDate.Time)
	winEnd := end
	if r.EndDate.Valid && r.EndDate.Time.Before(end) {
		winEnd = r.EndDate.Time
	}

	var instances []Transaction
	switch r.Interval {
	case "weekly", "biweekly":
		instances = expandWeeklyLike(r, winStart, winEnd)
	case "monthly":
		instances = expandMonthly(r, winStart, winEnd)
	case "yearly":
		instances = expandYearly(r, winStart, winEnd)
	}
	return instances
}

func expandWeeklyLike(r Recurring, start, end time.Time) []Transaction {
	var out []Transaction
	step := 7
	if r.Interval == "biweekly" {
		step = 14
	}
	anchor := truncateDay(r.StartDate.Time)

	wantDOW := int(anchor.Weekday())
	if r.DayOfWeek.Valid {
		wantDOW = int(r.DayOfWeek.Int32)
	}
	first := alignToNextOnPhase(anchor, start, step, wantDOW)

	for d := first; !d.After(end); d = d.AddDate(0, 0, step) {
		if int(d.Weekday()) != wantDOW {
			d = snapToWeekday(d, time.Weekday(wantDOW))
		}
		out = append(out, toTxFromRecurring(r, d))
	}
	return out
}

func expandMonthly(r Recurring, start, end time.Time) []Transaction {
	var out []Transaction
	anchor := truncateDay(r.StartDate.Time)
	day := anchor.Day()
	if r.DayOfMonth.Valid {
		day = int(r.DayOfMonth.Int32)
	}
	y, m := start.Year(), start.Month()
	for d := dateAtDayOrMonthEnd(y, m, day); !d.After(end); {
		if !d.Before(start) && !d.Before(anchor) {
			out = append(out, toTxFromRecurring(r, d))
		}
		if m == 12 {
			y, m = y+1, 1
		} else {
			m++
		}
		d = dateAtDayOrMonthEnd(y, m, day)
	}
	return out
}

func expandYearly(r Recurring, start, end time.Time) []Transaction {
	var out []Transaction
	anchor := truncateDay(r.StartDate.Time)
	day := anchor.Day()
	if r.DayOfMonth.Valid {
		day = int(r.DayOfMonth.Int32)
	}
	month := anchor.Month()
	y := start.Year()
	cand := dateAtDayOrMonthEnd(y, month, day)
	if cand.Before(start) {
		y++
		cand = dateAtDayOrMonthEnd(y, month, day)
	}
	for !cand.After(end) {
		if !cand.Before(anchor) {
			out = append(out, toTxFromRecurring(r, cand))
		}
		y++
		cand = dateAtDayOrMonthEnd(y, month, day)
	}
	return out
}

func toTxFromRecurring(r Recurring, d time.Time) Transaction {
	amt := r.Amount
	if r.Type == "expense" {
		amt = makePgNumeric(-toFloat(r.Amount))
	}
	return Transaction{
		ID:          0,
		Date:        makePgDate(d),
		Amount:      amt,
		Description: r.Description,
		Type:        r.Type,
	}
}

func truncateDay(t time.Time) time.Time { return t.Truncate(24 * time.Hour) }

func maxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func snapToWeekday(d time.Time, w time.Weekday) time.Time {
	diff := int(w) - int(d.Weekday())
	if diff < 0 {
		diff += 7
	}
	return d.AddDate(0, 0, diff)
}

func alignToNextOnPhase(anchor, start time.Time, stepDays int, wantDOW int) time.Time {
	d := anchor
	for d.Before(start) {
		d = d.AddDate(0, 0, stepDays)
	}
	if wantDOW >= 0 {
		d = snapToWeekday(d, time.Weekday(wantDOW))
	}
	return d
}

func dateAtDayOrMonthEnd(y int, m time.Month, day int) time.Time {
	firstNext := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
	last := firstNext.AddDate(0, 0, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func toFloat(n pgtype.Numeric) float64 {
	f, _ := NumericToFloat64(n)
	return f
}

func parseIntervalEnum(s string) (database.RecurrenceInterval, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "weekly":
		return database.RecurrenceIntervalWeekly, nil
	case "biweekly":
		return database.RecurrenceIntervalBiweekly, nil
	case "monthly":
		return database.RecurrenceIntervalMonthly, nil
	case "yearly":
		return database.RecurrenceIntervalYearly, nil
	default:
		return "", fmt.Errorf("invalid interval %q (expected weekly|biweekly|monthly|yearly)", s)
	}
}
