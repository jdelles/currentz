package service

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jdelles/currentz/internal/database"
)

type Transaction = database.Transactions

type DailyCashFlow struct {
	Date    time.Time `json:"date"`
	Balance float64   `json:"balance"`
	Change  float64   `json:"change"`
}

type FinanceService struct {
	db database.Querier
}

func NewFinanceService(db database.Querier) *FinanceService {
	return &FinanceService{db: db}
}

func NewFinanceServiceFromURL(ctx context.Context, dbURL string) (*FinanceService, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	return &FinanceService{
		db: database.New(pool),
	}, nil
}

func (fs *FinanceService) GetStartingBalance(ctx context.Context) (float64, error) {
	value, err := fs.db.GetSetting(ctx, "starting_balance")
	if err != nil {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

func (fs *FinanceService) SetStartingBalance(ctx context.Context, balance float64) error {
	return fs.db.UpdateSetting(ctx, database.UpdateSettingParams{
		Key:   "starting_balance",
		Value: fmt.Sprintf("%.2f", balance),
	})
}

func (fs *FinanceService) AddIncome(ctx context.Context, date time.Time, amount float64, description string) error {
	return fs.db.CreateTransaction(ctx, database.CreateTransactionParams{
		Date:        makePgDate(date),
		Amount:      makePgNumeric(amount),
		Description: description,
		Type:        "income",
	})
}

func (fs *FinanceService) AddExpense(ctx context.Context, date time.Time, amount float64, description string) error {
	return fs.db.CreateTransaction(ctx, database.CreateTransactionParams{
		Date:        makePgDate(date),
		Amount:      makePgNumeric(-amount),
		Description: description,
		Type:        "expense",
	})
}

func (fs *FinanceService) GetAllTransactions(ctx context.Context) ([]Transaction, error) {
	return fs.db.GetAllTransactions(ctx)
}

func (fs *FinanceService) DeleteTransaction(ctx context.Context, id int32) error {
	return fs.db.DeleteTransaction(ctx, id)
}

func (fs *FinanceService) Calculate90DayForecast(ctx context.Context, startingBalance float64) ([]DailyCashFlow, error) {
	// 1) window (UTC midnight to avoid time drift)
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, 89)

	// 2) one-offs from DB
	oneOffs, err := fs.db.GetAllTransactions(ctx)
	if err != nil {
		return nil, err
	}

	// 3) expanded recurrings inside the window
	recs, err := fs.ExpandRecurringBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// 4) sum daily deltas
	daily := make(map[time.Time]float64, 100)
	for _, tx := range append(oneOffs, recs...) {
		// normalize to UTC day key
		day := tx.Date.Time.In(time.UTC).Truncate(24 * time.Hour)
		amt, err := NumericToFloat64(tx.Amount)
		if err != nil {
			continue
		}
		daily[day] += amt
	}

	// 5) accumulate into balances
	fc := make([]DailyCashFlow, 90)
	bal := startingBalance
	for i := 0; i < 90; i++ {
		day := start.AddDate(0, 0, i)
		change := daily[day]
		bal += change
		fc[i] = DailyCashFlow{Date: day, Balance: bal, Change: change}
	}
	return fc, nil
}

func (fs *FinanceService) FindLowestPoint(forecast []DailyCashFlow) (DailyCashFlow, int) {
	if len(forecast) == 0 {
		return DailyCashFlow{}, -1
	}
	lowest := forecast[0]
	lowestIndex := 0
	for i, day := range forecast {
		if day.Balance < lowest.Balance {
			lowest = day
			lowestIndex = i
		}
	}
	return lowest, lowestIndex
}

func (fs *FinanceService) GetUpcomingTransactions(ctx context.Context, days int) ([]Transaction, error) {
	start := time.Now().Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, days)
	return fs.GetTransactionsWithRecurringsBetween(ctx, start, end)
}

func makePgDate(t time.Time) pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(t)
	return d
}

func makePgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func NumericToFloat64(n pgtype.Numeric) (float64, error) {
	if n.Int == nil {
		return 0, nil
	}
	r := new(big.Rat).SetInt(n.Int)
	if n.Exp > 0 {
		factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		r.Mul(r, new(big.Rat).SetInt(factor))
	} else if n.Exp < 0 {
		factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		r.Quo(r, new(big.Rat).SetInt(factor))
	}
	f, _ := r.Float64()
	return f, nil
}

func (fs *FinanceService) GetTransactionsWithRecurringsBetween(ctx context.Context, start, end time.Time) ([]Transaction, error) {
	oneOffs, err := fs.db.GetTransactionsByDateRange(ctx, database.GetTransactionsByDateRangeParams{
		Date:   makePgDate(start),
		Date_2: makePgDate(end),
	})
	if err != nil {
		return nil, err
	}
	recs, err := fs.ExpandRecurringBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	all := append(oneOffs, recs...)
	sort.SliceStable(all, func(i, j int) bool {
		ti := all[i].Date.Time
		tj := all[j].Date.Time
		if ti.Equal(tj) {
			return all[i].Description < all[j].Description
		}
		return ti.Before(tj)
	})
	return all, nil
}
