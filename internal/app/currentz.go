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
	"github.com/jdelles/currentz/internal/service"
)

type FinanceApp struct {
	service *service.FinanceService
}

func NewFinanceApp(cfg *config.Config) (*FinanceApp, error) {
	ctx := context.Background()
	svc, err := service.NewFinanceServiceFromURL(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init service: %w", err)
	}
	return &FinanceApp{service: svc}, nil
}

func (fa *FinanceApp) Close() error {
	if fa.service != nil {
		fa.service.Close()
	}
	return nil
}

func (fa *FinanceApp) Run() error {
	fmt.Println("💵 Personal Finance Cash Flow Forecaster")
	fmt.Println("========================================")

	ctx := context.Background()

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
		fmt.Println("7. Manage Recurring Transactions")
		fmt.Println("8. Exit")

		choice := getUserInput("Choose an option (1-8): ")

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
			if err := fa.manageRecurring(ctx); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "8":
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
	start := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	end := time.Now().AddDate(0, 0, 30).Truncate(24 * time.Hour)

	transactions, err := fa.service.GetTransactionsWithRecurringsBetween(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to load transactions: %w", err)
	}

	if len(transactions) == 0 {
		fmt.Println("No transactions in the last/next 30 days.")
		return nil
	}

	fmt.Println("\n📋 Transactions (Past 30 days → Next 30 days)")
	fmt.Println("=" + strings.Repeat("=", 70))

	for _, tx := range transactions {
		symbol := "💰"
		amount, _ := service.NumericToFloat64(tx.Amount)
		displayAmount := amount

		if tx.Type == "expense" {
			symbol = "💸"
			displayAmount = -amount
		}

		id := tx.ID
		idLabel := fmt.Sprintf("%d", id)
		if id == 0 {
			idLabel = "R"
		}

		fmt.Printf("[%s] %s %s | $%8.2f | %s\n",
			idLabel,
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

func (fa *FinanceApp) manageRecurring(ctx context.Context) error {
	fmt.Println("\nRecurring Menu:")
	fmt.Println("1. List")
	fmt.Println("2. Add")
	fmt.Println("3. Delete")
	fmt.Println("4. Toggle Active")
	choice := getUserInput("Choose (1-4): ")

	switch choice {
	case "1":
		rs, err := fa.service.ListRecurring(ctx)
		if err != nil {
			return err
		}
		if len(rs) == 0 {
			fmt.Println("No recurring transactions.")
			return nil
		}
		for _, r := range rs {
			active := "✅"
			if !r.Active {
				active = "⏸"
			}
			amt, err := service.NumericToFloat64(r.Amount)
			if err != nil {
				fmt.Printf("⚠️  could not parse amount for id=%d (%q): %v; using $0.00\n",
					r.ID, r.Description, err)
				amt = 0
			}
			freq := string(r.Interval)
			fmt.Printf("[%d] %s | %-7s | $%8.2f | %-9s | start %s | %s\n",
				r.ID, active, r.Type, amt, freq, r.StartDate.Time.Format("2006-01-02"), r.Description)
		}
	case "2":
		desc := getUserInput("Description: ")
		typ := strings.ToLower(getUserInput("Type (income/expense): "))

		amtStr := getUserInput("Amount (e.g., 1500.00): ")
		amt, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}

		startStr := getUserInput("Start date (YYYY-MM-DD): ")
		start, err := parseDate(startStr)
		if err != nil {
			return fmt.Errorf("invalid start date: %w", err)
		}

		interval := strings.ToLower(getUserInput("Interval (weekly/biweekly/monthly/yearly): "))

		var dow *int
		var dom *int
		if interval == "weekly" || interval == "biweekly" {
			s := strings.TrimSpace(getUserInput("Day of week (0=Sun..6=Sat, blank=use start_date): "))
			if s != "" {
				v, err := strconv.Atoi(s)
				if err != nil || v < 0 || v > 6 {
					return fmt.Errorf("invalid day_of_week: %q", s)
				}
				dow = &v
			}
		}
		if interval == "monthly" || interval == "yearly" {
			s := strings.TrimSpace(getUserInput("Day of month (1..31, blank=use start_date): "))
			if s != "" {
				v, err := strconv.Atoi(s)
				if err != nil || v < 1 || v > 31 {
					return fmt.Errorf("invalid day_of_month: %q", s)
				}
				dom = &v
			}
		}

		var end *time.Time
		endStr := strings.TrimSpace(getUserInput("End date (YYYY-MM-DD, blank = none): "))
		if endStr != "" {
			e, err := parseDate(endStr)
			if err != nil {
				return fmt.Errorf("invalid end date: %w", err)
			}
			end = &e
		}

		_, err = fa.service.CreateRecurringSimple(ctx, service.RecurringInput{
			Description: desc,
			Type:        typ,
			Amount:      amt,
			StartDate:   start,
			Interval:    interval,
			DayOfWeek:   dow,
			DayOfMonth:  dom,
			EndDate:     end,
			Active:      true,
		})
		if err != nil {
			return err
		}
		fmt.Println("✅ Recurring saved.")

	case "3":
		idStr := getUserInput("ID to delete: ")
		id, _ := strconv.Atoi(idStr)
		if err := fa.service.DeleteRecurring(ctx, int32(id)); err != nil {
			return err
		}
		fmt.Println("✅ Deleted.")
	case "4":
		idStr := getUserInput("ID to toggle: ")
		id, _ := strconv.Atoi(idStr)
		activeStr := strings.ToLower(getUserInput("Active? (y/n): "))
		active := activeStr == "y" || activeStr == "yes"
		if err := fa.service.SetRecurringActive(ctx, int32(id), active); err != nil {
			return err
		}
		fmt.Println("✅ Updated.")
	default:
		fmt.Println("Cancelled.")
	}
	return nil
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
