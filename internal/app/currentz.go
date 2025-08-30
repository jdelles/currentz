package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jdelles/currentz/internal/config"
	"github.com/jdelles/currentz/internal/database"
	"github.com/jdelles/currentz/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinanceApp struct {
	service *service.FinanceService
	pool *pgxpool.Pool
}

func NewFinanceApp(cfg *config.Config) (*FinanceApp, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	queries := database.New(pool) // sqlc-generated constructor

	return &FinanceApp{
		pool:    pool,
		service: service.NewFinanceService(queries),
	}, nil
}

func (fa *FinanceApp) Close() error {
	if fa.pool != nil {
		fa.pool.Close()
	}
	return nil
}

func (fa *FinanceApp) Run() error {
	fmt.Println("💵 Personal Finance Cash Flow Forecaster")
	fmt.Println("========================================")

	ctx := context.Background()

	// Check and setup starting balance
	startingBalance, err := fa.service.GetStartingBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get starting balance: %w", err)
	}

	if startingBalance == 0 {
		if err := fa.setupStartingBalance(ctx); err != nil {
			return err
		}
	} else {
		fmt.Printf("Current starting balance: $%.2f\n", startingBalance)
	}

	return fa.mainLoop(ctx)
}

func (fa *FinanceApp) setupStartingBalance(ctx context.Context) error {
	balanceStr := getUserInput("Enter your current account balance: $")
	balance, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		fmt.Println("Invalid balance. Using $0.")
		balance = 0
	}

	return fa.service.SetStartingBalance(ctx, balance)
}

func (fa *FinanceApp) mainLoop(ctx context.Context) error {
	for {
		fmt.Println("\nOptions:")
		fmt.Println("1. Add Income")
		fmt.Println("2. Add Expense")
		fmt.Println("3. View Transactions")
		fmt.Println("4. Delete Transaction")
		fmt.Println("5. Generate Forecast")
		fmt.Println("6. Update Starting Balance")
		fmt.Println("7. Exit")

		choice := getUserInput("Choose an option (1-7): ")

		switch choice {
		case "1":
			if err := fa.addIncome(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "2":
			if err := fa.addExpense(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "3":
			if err := fa.viewTransactions(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "4":
			if err := fa.deleteTransaction(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "5":
			if err := fa.generateForecast(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "6":
			if err := fa.updateStartingBalance(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "7":
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

func (fa *FinanceApp) addIncome(ctx context.Context) error {
	dateStr := getUserInput("Enter date (YYYY-MM-DD or MM/DD/YYYY): ")
	date, err := parseDate(dateStr)
	if err != nil {
		return fmt.Errorf("error parsing date: %w", err)
	}

	amountStr := getUserInput("Enter income amount: $")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	description := getUserInput("Enter description: ")

	if err := fa.service.AddIncome(ctx, date, amount, description); err != nil {
		return fmt.Errorf("failed to add income: %w", err)
	}

	fmt.Printf("✅ Added income: $%.2f on %s\n", amount, date.Format("Jan 2, 2006"))
	return nil
}

func (fa *FinanceApp) addExpense(ctx context.Context) error {
	dateStr := getUserInput("Enter date (YYYY-MM-DD or MM/DD/YYYY): ")
	date, err := parseDate(dateStr)
	if err != nil {
		return fmt.Errorf("error parsing date: %w", err)
	}

	amountStr := getUserInput("Enter expense amount: $")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	description := getUserInput("Enter description: ")

	if err := fa.service.AddExpense(ctx, date, amount, description); err != nil {
		return fmt.Errorf("failed to add expense: %w", err)
	}

	fmt.Printf("✅ Added expense: $%.2f on %s\n", amount, date.Format("Jan 2, 2006"))
	return nil
}

func (fa *FinanceApp) viewTransactions(ctx context.Context) error {
	transactions, err := fa.service.GetAllTransactions(ctx)
	if err != nil {
		return fmt.Errorf("failed to load transactions: %w", err)
	}

	if len(transactions) == 0 {
		fmt.Println("No transactions recorded yet.")
		return nil
	}

	fmt.Println("\n📋 Recorded Transactions")
	fmt.Println("=" + strings.Repeat("=", 70))

	for _, tx := range transactions {
		symbol := "💰"
		amount, _ := service.NumericToFloat64(tx.Amount)
		displayAmount := amount
		
		if tx.Type == "expense" {
			symbol = "💸"
			displayAmount = -amount // Show positive amount for display
		}

		fmt.Printf("[%d] %s %s | $%8.2f | %s\n",
			tx.ID,
			symbol,
			tx.Date.Time.Format("Jan 02, 2006"),
			displayAmount,
			tx.Description)
	}
	return nil
}

func (fa *FinanceApp) deleteTransaction(ctx context.Context) error {
	if err := fa.viewTransactions(ctx); err != nil {
		return err
	}

	transactions, err := fa.service.GetAllTransactions(ctx)
	if err != nil || len(transactions) == 0 {
		return nil
	}

	idStr := getUserInput("\nEnter transaction ID to delete (or 0 to cancel): ")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		return fmt.Errorf("invalid ID")
	}

	if id == 0 {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := fa.service.DeleteTransaction(ctx, int32(id)); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	fmt.Printf("✅ Transaction %d deleted successfully.\n", id)
	return nil
}

func (fa *FinanceApp) updateStartingBalance(ctx context.Context) error {
	currentBalance, err := fa.service.GetStartingBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}

	fmt.Printf("Current starting balance: $%.2f\n", currentBalance)
	balanceStr := getUserInput("Enter new starting balance: $")
	balance, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		return fmt.Errorf("invalid balance: %w", err)
	}

	if err := fa.service.SetStartingBalance(ctx, balance); err != nil {
		return fmt.Errorf("failed to save starting balance: %w", err)
	}

	fmt.Printf("✅ Starting balance updated to $%.2f\n", balance)
	return nil
}

func (fa *FinanceApp) generateForecast(ctx context.Context) error {
	startingBalance, err := fa.service.GetStartingBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get starting balance: %w", err)
	}

	forecast, err := fa.service.Calculate90DayForecast(ctx, startingBalance)
	if err != nil {
		return fmt.Errorf("failed to generate forecast: %w", err)
	}

	DisplayChart(forecast)
	DisplaySummary(forecast, startingBalance, fa.service)

	// Show upcoming transactions
	fmt.Println("\n📅 Upcoming Transactions (Next 30 Days)")
	fmt.Println("=" + strings.Repeat("=", 50))

	upcoming, err := fa.service.GetUpcomingTransactions(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to get upcoming transactions: %w", err)
	}

	if len(upcoming) == 0 {
		fmt.Println("No transactions scheduled for the next 30 days.")
		return nil
	}

	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].Date.Time.Before(upcoming[j].Date.Time)
	})

	today := time.Now()
	for _, tx := range upcoming {
		symbol := "💰"
		amount, _ := service.NumericToFloat64(tx.Amount)
		displayAmount := amount
		
		if tx.Type == "expense" {
			symbol = "💸"
			displayAmount = -amount
		}

		daysFromNow := int(tx.Date.Time.Sub(today).Hours() / 24)
		fmt.Printf("%s %s (%d days) | $%8.2f | %s\n",
			symbol,
			tx.Date.Time.Format("Jan 02"),
			daysFromNow,
			displayAmount,
			tx.Description)
	}

	return nil
}

// Display functions
func DisplayChart(forecast []service.DailyCashFlow) {
	fmt.Println("\n📊 90-Day Cash Flow Forecast")
	fmt.Println("=" + strings.Repeat("=", 60))

	if len(forecast) == 0 {
		fmt.Println("No forecast data available.")
		return
	}

	// Find min and max for scaling
	minBalance := forecast[0].Balance
	maxBalance := forecast[0].Balance

	for _, day := range forecast {
		if day.Balance < minBalance {
			minBalance = day.Balance
		}
		if day.Balance > maxBalance {
			maxBalance = day.Balance
		}
	}

	// Create a simple ASCII chart
	chartWidth := 50
	fmt.Printf("\nBalance Range: $%.2f to $%.2f\n\n", minBalance, maxBalance)

	// Show every 7th day (weekly view)
	for i := 0; i < len(forecast); i += 7 {
		day := forecast[i]

		// Calculate position in chart (0 to chartWidth)
		var position int
		if maxBalance != minBalance {
			position = int(((day.Balance - minBalance) / (maxBalance - minBalance)) * float64(chartWidth))
		} else {
			position = chartWidth / 2
		}

		// Create the bar
		bar := strings.Repeat(" ", position) + "█"
		if position < chartWidth {
			bar += strings.Repeat(".", chartWidth-position)
		}

		fmt.Printf("%s │%s│ $%8.2f\n",
			day.Date.Format("Jan 02"),
			bar,
			day.Balance)
	}

	fmt.Println(strings.Repeat(" ", 7) + "└" + strings.Repeat("─", chartWidth+2) + "┘")
}

func DisplaySummary(forecast []service.DailyCashFlow, startingBalance float64, fs *service.FinanceService) {
	if len(forecast) == 0 {
		fmt.Println("No forecast data available.")
		return
	}

	lowest, lowestDay := fs.FindLowestPoint(forecast)

	fmt.Println("\n💰 Financial Summary")
	fmt.Println("=" + strings.Repeat("=", 40))

	fmt.Printf("Starting Balance: $%.2f\n", startingBalance)
	fmt.Printf("Ending Balance:   $%.2f\n", forecast[len(forecast)-1].Balance)
	fmt.Printf("Net Change:       $%.2f\n", forecast[len(forecast)-1].Balance-startingBalance)

	fmt.Println("\n⚠️  LOWEST POINT ANALYSIS")
	fmt.Printf("Lowest Balance:   $%.2f\n", lowest.Balance)
	fmt.Printf("Date:            %s\n", lowest.Date.Format("January 2, 2006"))
	fmt.Printf("Days from today: %d\n", lowestDay)

	if lowest.Balance < 0 {
		fmt.Printf("🚨 WARNING: You will go negative by $%.2f!\n", -lowest.Balance)
	} else if lowest.Balance < 1000 {
		fmt.Printf("⚠️  CAUTION: Balance drops below $1,000\n")
	}
}

// Utility functions
func parseDate(input string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"Jan 2, 2006",
		"January 2, 2006",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, input); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", input)
}

func getUserInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}