# Currentz

**Currentz** is a modern personal finance app inspired by classics like Microsoft Money — rebuilt with today's tools.  
Its primary focus is **cash flow**: helping you see not just where your money went, but where it’s going.

---

## ✨ Features (MVP)

- ✅ Interactive **CLI menu** to manage finances  
- ➕ Add income & expenses  
- 📋 View and delete transactions  
- 💰 Update your starting balance  
- 🔮 Generate a 90-day cash flow forecast with ASCII chart + summary  

---

## 🚀 Quickstart

1. **Prerequisites**
   - Go 1.21+
   - PostgreSQL 13+ (running locally)  
   ```bash
   brew services start postgresql

2. **Clone:**
```bash
git clone https://github.com/jdelles/currentz.git
cd currentz
```

3. **Generate sqlc code, setup, and other important things:**
```bash
make dev-setup
```

4. **Run the application:**
```bash
make run
```

## Using the CLI

When you run the app you'll see: 

💵 Personal Finance Cash Flow Forecaster
========================================

Options:
1. Add Income
2. Add Expense
3. View Transactions
4. Delete Transaction
5. Generate Forecast
6. Update Starting Balance
7. Exit

Example Transaction list: 

📋 Recorded Transactions
=======================================================================
[1] 💰 Jan 02, 2025 | $   500.00 | Paycheck  
[2] 💸 Jan 05, 2025 | $  -150.00 | Groceries  

Example forecast chart: 

📊 90-Day Cash Flow Forecast
============================================================

Balance Range: $350.00 to $500.00

Jan 02 │████████.......................│ $   500.00  
Jan 09 │█████..........................│ $   350.00  

## 🛠 Tech Stack

Go for application logic  
PostgreSQL for persistence  
sqlc to generate type-safe queries  
goose for migrations  

## Project Structure
```
currentz/
├── cmd/
│   └── currentz/
│       └── main.go
├── internal/
│   ├── app/          # CLI / TUI layer (menus, prompts, output)
│   ├── config/       # config loading (expects DB_URL)
│   ├── database/     # sqlc-generated code (models & queries)
│   └── service/      # business logic (forecasting, helpers)
├── sql/
│   ├── migrations/   # goose migrations
│   ├── queries/      # sqlc query files
│   └── schema.sql
├── scripts/
│   └── setup.sh
├── Makefile
├── sqlc.yaml
└── go.mod
```

## 🛤 Roadmap

- [ ] Transaction import (CSV/OFX)  
- [ ] Recurring events & bills & PAYCHECKS 💰 
- [ ] More forecasts & charts  
- [ ] An actual UI for richer experience... 
