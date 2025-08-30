package service

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

func (fs *FinanceService) GetStartingBalance(ctx context.Context) (float64, error) {
	value, err := fs.db.GetSetting(ctx, "starting_balance")
	if err != nil {
		return 0, nil // treat missing as 0
	}
	return strconv.ParseFloat(value, 64)
}

func (fs *FinanceService) SetStartingBalance(ctx context.Context, balance float64) error {
	return fs.db.UpsertSetting(ctx, database.UpsertSettingParams{
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
		Amount:      makePgNumeric(-amount), // store negative for expenses
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
	transactions, err := fs.db.GetAllTransactions(ctx)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	forecast := make([]DailyCashFlow, 90)
	currentBalance := startingBalance

	for i := 0; i < 90; i++ {
		date := today.AddDate(0, 0, i)
		dailyChange := 0.0

		for _, tx := range transactions {
			// tx.Date is pgtype.Date; use its Time directly
			if tx.Date.Time.Truncate(24*time.Hour).Equal(date) {
				amt, err := NumericToFloat64(tx.Amount)
				if err != nil {
					continue
				}
				dailyChange += amt
			}
		}

		currentBalance += dailyChange
		forecast[i] = DailyCashFlow{
			Date:    date,
			Balance: currentBalance,
			Change:  dailyChange,
		}
	}

	return forecast, nil
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
	today := time.Now().Truncate(24 * time.Hour)
	endDate := today.AddDate(0, 0, days)

	return fs.db.GetTransactionsByDateRange(ctx, database.GetTransactionsByDateRangeParams{
		Date:   makePgDate(today),
		Date_2: makePgDate(endDate),
	})
}

/*** helpers ***/

// makePgDate builds a pgtype.Date via Scan to avoid relying on struct fields.
func makePgDate(t time.Time) pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(t) // pgtype.Date implements sql.Scanner
	return d
}

// makePgNumeric builds a pgtype.Numeric via Scan of a decimal string.
func makePgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

// numericToFloat64 reconstructs a float64 from pgtype.Numeric's Int and Exp.
func NumericToFloat64(n pgtype.Numeric) (float64, error) {
	// If not present or no integer component, treat as zero.
	if n.Int == nil {
		return 0, nil
	}

	r := new(big.Rat).SetInt(n.Int)
	if n.Exp > 0 {
		// multiply by 10^Exp
		factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		r.Mul(r, new(big.Rat).SetInt(factor))
	} else if n.Exp < 0 {
		// divide by 10^{-Exp}
		factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		r.Quo(r, new(big.Rat).SetInt(factor))
	}

	f, _ := r.Float64()
	return f, nil
}
