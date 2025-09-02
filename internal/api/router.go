package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jdelles/currentz/internal/service"
)

// FinanceServiceInterface defines the interface that our API depends on
type FinanceServiceInterface interface {
	GetAllTransactions(ctx context.Context) ([]service.Transaction, error)
	AddIncome(ctx context.Context, date time.Time, amount float64, description string) error
	AddExpense(ctx context.Context, date time.Time, amount float64, description string) error
	DeleteTransaction(ctx context.Context, id int32) error
	GetStartingBalance(ctx context.Context) (float64, error)
	SetStartingBalance(ctx context.Context, balance float64) error
	CreateRecurringSimple(ctx context.Context, input service.RecurringInput) (service.Recurring, error)
	ListRecurring(ctx context.Context) ([]service.Recurring, error)
	DeleteRecurring(ctx context.Context, id int32) error
	SetRecurringActive(ctx context.Context, id int32, active bool) error
	Calculate90DayForecast(ctx context.Context, startingBalance float64) ([]service.DailyCashFlow, error)
	FindLowestPoint(forecast []service.DailyCashFlow) (service.DailyCashFlow, int)
	GetUpcomingTransactions(ctx context.Context, days int) ([]service.Transaction, error)
	GetTransactionsWithRecurringsBetween(ctx context.Context, start, end time.Time) ([]service.Transaction, error)
}

type APIServer struct {
	financeService FinanceServiceInterface
}

func NewAPIServer(financeService FinanceServiceInterface) *APIServer {
	return &APIServer{
		financeService: financeService,
	}
}

// JSON request/response types
type AddTransactionRequest struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type SetBalanceRequest struct {
	Balance float64 `json:"balance"`
}

type RecurringTransactionRequest struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	StartDate   string  `json:"start_date"`
	Interval    string  `json:"interval"`
	DayOfWeek   *int    `json:"day_of_week,omitempty"`
	DayOfMonth  *int    `json:"day_of_month,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Active      bool    `json:"active"`
}

type SetActiveRequest struct {
	Active bool `json:"active"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Helper functions
func (s *APIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

func (s *APIServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, ErrorResponse{Error: message})
}

func parseDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// Transaction endpoints
func (s *APIServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := s.financeService.GetAllTransactions(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, transactions)
}

func (s *APIServer) handleAddIncome(w http.ResponseWriter, r *http.Request) {
	var req AddTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	date, err := parseDate(req.Date)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.financeService.AddIncome(r.Context(), date, req.Amount, req.Description); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]string{"status": "success"})
}

func (s *APIServer) handleAddExpense(w http.ResponseWriter, r *http.Request) {
	var req AddTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	date, err := parseDate(req.Date)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.financeService.AddExpense(r.Context(), date, req.Amount, req.Description); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]string{"status": "success"})
}

func (s *APIServer) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	if err := s.financeService.DeleteTransaction(r.Context(), int32(id)); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Balance endpoints
func (s *APIServer) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	balance, err := s.financeService.GetStartingBalance(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]float64{"balance": balance})
}

func (s *APIServer) handleSetBalance(w http.ResponseWriter, r *http.Request) {
	var req SetBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := s.financeService.SetStartingBalance(r.Context(), req.Balance); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Recurring transaction endpoints
func (s *APIServer) handleCreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req RecurringTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid start date: %s", err.Error()))
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		ed, err := parseDate(*req.EndDate)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid end date: %s", err.Error()))
			return
		}
		endDate = &ed
	}

	input := service.RecurringInput{
		Description: req.Description,
		Type:        req.Type,
		Amount:      req.Amount,
		StartDate:   startDate,
		Interval:    req.Interval,
		DayOfWeek:   req.DayOfWeek,
		DayOfMonth:  req.DayOfMonth,
		EndDate:     endDate,
		Active:      req.Active,
	}

	recurring, err := s.financeService.CreateRecurringSimple(r.Context(), input)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, recurring)
}

func (s *APIServer) handleListRecurring(w http.ResponseWriter, r *http.Request) {
	recurring, err := s.financeService.ListRecurring(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, recurring)
}

func (s *APIServer) handleDeleteRecurring(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid recurring transaction ID")
		return
	}

	if err := s.financeService.DeleteRecurring(r.Context(), int32(id)); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *APIServer) handleSetRecurringActive(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid recurring transaction ID")
		return
	}

	var req SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := s.financeService.SetRecurringActive(r.Context(), int32(id), req.Active); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Forecast endpoints
func (s *APIServer) handleGetForecast(w http.ResponseWriter, r *http.Request) {
	balance, err := s.financeService.GetStartingBalance(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	forecast, err := s.financeService.Calculate90DayForecast(r.Context(), balance)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, forecast)
}

func (s *APIServer) handleGetLowestPoint(w http.ResponseWriter, r *http.Request) {
	balance, err := s.financeService.GetStartingBalance(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	forecast, err := s.financeService.Calculate90DayForecast(r.Context(), balance)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lowest, index := s.financeService.FindLowestPoint(forecast)

	response := map[string]interface{}{
		"lowest_point": lowest,
		"day_index":    index,
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *APIServer) handleGetUpcoming(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30 // default

	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	transactions, err := s.financeService.GetUpcomingTransactions(r.Context(), days)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, transactions)
}

func (s *APIServer) handleGetTransactionsBetween(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		s.writeError(w, http.StatusBadRequest, "Both 'start' and 'end' query parameters are required")
		return
	}

	start, err := parseDate(startStr)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid start date: %s", err.Error()))
		return
	}

	end, err := parseDate(endStr)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid end date: %s", err.Error()))
		return
	}

	transactions, err := s.financeService.GetTransactionsWithRecurringsBetween(r.Context(), start, end)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, transactions)
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *APIServer) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(corsMiddleware)

	// Catch-all OPTIONS handler so preflights always match
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// corsMiddleware already set the headers; just OK it.
		w.WriteHeader(http.StatusOK)
	})

	// Transaction routes
	r.HandleFunc("/api/transactions", s.handleGetTransactions).Methods("GET")
	r.HandleFunc("/api/transactions/income", s.handleAddIncome).Methods("POST")
	r.HandleFunc("/api/transactions/expense", s.handleAddExpense).Methods("POST")
	r.HandleFunc("/api/transactions/{id:[0-9]+}", s.handleDeleteTransaction).Methods("DELETE")
	r.HandleFunc("/api/transactions/between", s.handleGetTransactionsBetween).Methods("GET")
	r.HandleFunc("/api/transactions/upcoming", s.handleGetUpcoming).Methods("GET")

	// Balance routes
	r.HandleFunc("/api/balance", s.handleGetBalance).Methods("GET")
	r.HandleFunc("/api/balance", s.handleSetBalance).Methods("PUT")

	// Recurring transaction routes
	r.HandleFunc("/api/recurring", s.handleCreateRecurring).Methods("POST")
	r.HandleFunc("/api/recurring", s.handleListRecurring).Methods("GET")
	r.HandleFunc("/api/recurring/{id:[0-9]+}", s.handleDeleteRecurring).Methods("DELETE")
	r.HandleFunc("/api/recurring/{id:[0-9]+}/active", s.handleSetRecurringActive).Methods("PUT")

	// Forecast routes
	r.HandleFunc("/api/forecast", s.handleGetForecast).Methods("GET")
	r.HandleFunc("/api/forecast/lowest", s.handleGetLowestPoint).Methods("GET")

	return r
}

func (s *APIServer) Start(addr string) error {
	router := s.SetupRoutes()

	log.Printf("Starting API server on %s", addr)
	log.Println("Available endpoints:")
	log.Println("  GET    /api/transactions - Get all transactions")
	log.Println("  POST   /api/transactions/income - Add income")
	log.Println("  POST   /api/transactions/expense - Add expense")
	log.Println("  DELETE /api/transactions/{id} - Delete transaction")
	log.Println("  GET    /api/transactions/between?start=DATE&end=DATE - Get transactions in range")
	log.Println("  GET    /api/transactions/upcoming?days=N - Get upcoming transactions")
	log.Println("  GET    /api/balance - Get starting balance")
	log.Println("  PUT    /api/balance - Set starting balance")
	log.Println("  POST   /api/recurring - Create recurring transaction")
	log.Println("  GET    /api/recurring - List recurring transactions")
	log.Println("  DELETE /api/recurring/{id} - Delete recurring transaction")
	log.Println("  PUT    /api/recurring/{id}/active - Set recurring transaction active status")
	log.Println("  GET    /api/forecast - Get 90-day forecast")
	log.Println("  GET    /api/forecast/lowest - Get lowest balance point in forecast")

	return http.ListenAndServe(addr, router)
}
