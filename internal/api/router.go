package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jdelles/currentz/internal/service"
)

type API struct {
	svc *service.FinanceService
}

func New(svc *service.FinanceService) *API { return &API{svc: svc} }

func (a *API) Router() http.Handler {
	r := chi.NewRouter()

	// CORS for any frontend (tighten for prod)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/forecast", a.handleForecast)
	r.Get("/transactions", a.handleTransactionsWindow)
	r.Get("/recurrings", a.handleListRecurrings)
	r.Post("/recurrings", a.handleCreateRecurring)

	r.Get("/settings/starting-balance", a.handleGetStartingBalance)
	r.Put("/settings/starting-balance", a.handleSetStartingBalance)

	return r
}

func (a *API) handleForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := 90
	if q := r.URL.Query().Get("days"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 365 {
			days = v
		}
	}
	starting, err := a.svc.GetStartingBalance(ctx)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}

	// reuse your 90-day and just slice if smaller
	fc, err := a.svc.Calculate90DayForecast(ctx, starting)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}
	if days < len(fc) {
		fc = fc[:days]
	}

	writeJSON(w, http.StatusOK, fc)
}

func (a *API) handleTransactionsWindow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	// default: past 30 → next 30 days
	start := time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	end := time.Now().UTC().AddDate(0, 0, 30).Truncate(24 * time.Hour)

	if startStr != "" {
		if t, err := parseDate(startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := parseDate(endStr); err == nil {
			end = t
		}
	}

	items, err := a.svc.GetTransactionsWithRecurringsBetween(ctx, start, end)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) handleListRecurrings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rs, err := a.svc.ListRecurring(ctx)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

type createRecurringReq struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`                  // "income" | "expense"
	Amount      float64 `json:"amount"`                // positive
	StartDate   string  `json:"start_date"`            // "YYYY-MM-DD"
	Interval    string  `json:"interval"`              // weekly|biweekly|monthly|yearly
	DayOfWeek   *int    `json:"day_of_week,omitempty"` // 0..6
	DayOfMonth  *int    `json:"day_of_month,omitempty"`
	EndDate     *string `json:"end_date,omitempty"` // "YYYY-MM-DD"
	Active      *bool   `json:"active,omitempty"`   // default true
}

func (a *API) handleCreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req createRecurringReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, err)
		return
	}
	start, err := parseDate(req.StartDate)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, err)
		return
	}

	var end *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		t, err := parseDate(*req.EndDate)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		end = &t
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	rec, err := a.svc.CreateRecurringSimple(r.Context(), service.RecurringInput{
		Description: req.Description,
		Type:        req.Type,
		Amount:      req.Amount,
		StartDate:   start,
		Interval:    req.Interval,
		DayOfWeek:   req.DayOfWeek,
		DayOfMonth:  req.DayOfMonth,
		EndDate:     end,
		Active:      active,
	})
	if err != nil {
		errorJSON(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, rec)
}

func (a *API) handleGetStartingBalance(w http.ResponseWriter, r *http.Request) {
	val, err := a.svc.GetStartingBalance(r.Context())
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]float64{"starting_balance": val})
}

type setStartingBalanceReq struct {
	StartingBalance float64 `json:"starting_balance"`
}

func (a *API) handleSetStartingBalance(w http.ResponseWriter, r *http.Request) {
	var req setStartingBalanceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, err)
		return
	}
	if err := a.svc.SetStartingBalance(r.Context(), req.StartingBalance); err != nil {
		errorJSON(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errorJSON(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"error": err.Error()})
}

func parseDate(s string) (time.Time, error) {
	// accept YYYY-MM-DD; normalize to UTC midnight
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(time.UTC).Truncate(24 * time.Hour), nil
}
