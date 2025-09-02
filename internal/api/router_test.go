package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jdelles/currentz/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFinanceService struct {
	mock.Mock
}

func (m *MockFinanceService) GetAllTransactions(ctx context.Context) ([]service.Transaction, error) {
	args := m.Called(ctx)
	return args.Get(0).([]service.Transaction), args.Error(1)
}

func (m *MockFinanceService) AddIncome(ctx context.Context, date time.Time, amount float64, description string) error {
	args := m.Called(ctx, date, amount, description)
	return args.Error(0)
}

func (m *MockFinanceService) AddExpense(ctx context.Context, date time.Time, amount float64, description string) error {
	args := m.Called(ctx, date, amount, description)
	return args.Error(0)
}

func (m *MockFinanceService) DeleteTransaction(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFinanceService) GetStartingBalance(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockFinanceService) SetStartingBalance(ctx context.Context, balance float64) error {
	args := m.Called(ctx, balance)
	return args.Error(0)
}

func (m *MockFinanceService) CreateRecurringSimple(ctx context.Context, input service.RecurringInput) (service.Recurring, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(service.Recurring), args.Error(1)
}

func (m *MockFinanceService) ListRecurring(ctx context.Context) ([]service.Recurring, error) {
	args := m.Called(ctx)
	return args.Get(0).([]service.Recurring), args.Error(1)
}

func (m *MockFinanceService) DeleteRecurring(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFinanceService) SetRecurringActive(ctx context.Context, id int32, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockFinanceService) Calculate90DayForecast(ctx context.Context, startingBalance float64) ([]service.DailyCashFlow, error) {
	args := m.Called(ctx, startingBalance)
	return args.Get(0).([]service.DailyCashFlow), args.Error(1)
}

func (m *MockFinanceService) FindLowestPoint(forecast []service.DailyCashFlow) (service.DailyCashFlow, int) {
	args := m.Called(forecast)
	return args.Get(0).(service.DailyCashFlow), args.Get(1).(int)
}

func (m *MockFinanceService) GetUpcomingTransactions(ctx context.Context, days int) ([]service.Transaction, error) {
	args := m.Called(ctx, days)
	return args.Get(0).([]service.Transaction), args.Error(1)
}

func (m *MockFinanceService) GetTransactionsWithRecurringsBetween(ctx context.Context, start, end time.Time) ([]service.Transaction, error) {
	args := m.Called(ctx, start, end)
	return args.Get(0).([]service.Transaction), args.Error(1)
}

// Test helper to create a test server
func setupTestServer(mockService FinanceServiceInterface) *httptest.Server {
	// Create an API server that uses our mock interface
	apiServer := NewAPIServer(mockService)
	router := apiServer.SetupRoutes()
	return httptest.NewServer(router)
}

// Test structures for table-driven tests
type testCase struct {
	name           string
	method         string
	path           string
	body           interface{}
	mockSetup      func(*MockFinanceService)
	expectedStatus int
	validateBody   func(*testing.T, []byte)
}

func TestTransactionEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "GET /api/transactions - success",
			method: "GET",
			path:   "/api/transactions",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetAllTransactions", mock.Anything).Return([]service.Transaction{
					{ID: 1, Description: "Test transaction"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var transactions []service.Transaction
				err := json.Unmarshal(body, &transactions)
				require.NoError(t, err)
				assert.Len(t, transactions, 1)
				assert.Equal(t, "Test transaction", transactions[0].Description)
			},
		},
		{
			name:   "GET /api/transactions - service error",
			method: "GET",
			path:   "/api/transactions",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetAllTransactions", mock.Anything).Return([]service.Transaction{}, fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body []byte) {
				var errResp ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)
				assert.Contains(t, errResp.Error, "database error")
			},
		},
		{
			name:   "POST /api/transactions/income - success",
			method: "POST",
			path:   "/api/transactions/income",
			body: AddTransactionRequest{
				Date:        "2025-09-15",
				Amount:      1000.50,
				Description: "Salary",
			},
			mockSetup: func(m *MockFinanceService) {
				expectedDate, _ := time.Parse("2006-01-02", "2025-09-15")
				m.On("AddIncome", mock.Anything, expectedDate, 1000.50, "Salary").Return(nil)
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, "success", resp["status"])
			},
		},
		{
			name:   "POST /api/transactions/income - invalid date",
			method: "POST",
			path:   "/api/transactions/income",
			body: AddTransactionRequest{
				Date:        "invalid-date",
				Amount:      1000.50,
				Description: "Salary",
			},
			mockSetup:      func(m *MockFinanceService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)
				assert.Contains(t, errResp.Error, "unable to parse date")
			},
		},
		{
			name:   "POST /api/transactions/expense - success",
			method: "POST",
			path:   "/api/transactions/expense",
			body: AddTransactionRequest{
				Date:        "2025-09-15",
				Amount:      500.25,
				Description: "Groceries",
			},
			mockSetup: func(m *MockFinanceService) {
				expectedDate, _ := time.Parse("2006-01-02", "2025-09-15")
				m.On("AddExpense", mock.Anything, expectedDate, 500.25, "Groceries").Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "DELETE /api/transactions/123 - success",
			method: "DELETE",
			path:   "/api/transactions/123",
			mockSetup: func(m *MockFinanceService) {
				m.On("DeleteTransaction", mock.Anything, int32(123)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE /api/transactions/invalid - bad ID",
			method:         "DELETE",
			path:           "/api/transactions/invalid",
			mockSetup:      func(m *MockFinanceService) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFinanceService)
			tt.mockSetup(mockService)

			server := setupTestServer(mockService)
			defer server.Close()

			var body []byte
			var err error
			if tt.body != nil {
				body, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBuffer(body))
			require.NoError(t, err)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				var respBody bytes.Buffer
				_, err := respBody.ReadFrom(resp.Body)
				require.NoError(t, err)
				tt.validateBody(t, respBody.Bytes())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBalanceEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "GET /api/balance - success",
			method: "GET",
			path:   "/api/balance",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetStartingBalance", mock.Anything).Return(5000.75, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]float64
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, 5000.75, resp["balance"])
			},
		},
		{
			name:   "PUT /api/balance - success",
			method: "PUT",
			path:   "/api/balance",
			body: SetBalanceRequest{
				Balance: 10000.00,
			},
			mockSetup: func(m *MockFinanceService) {
				m.On("SetStartingBalance", mock.Anything, 10000.00).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFinanceService)
			tt.mockSetup(mockService)

			server := setupTestServer(mockService)
			defer server.Close()

			var body []byte
			var err error
			if tt.body != nil {
				body, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBuffer(body))
			require.NoError(t, err)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				var respBody bytes.Buffer
				_, err := respBody.ReadFrom(resp.Body)
				require.NoError(t, err)
				tt.validateBody(t, respBody.Bytes())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestRecurringEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "GET /api/recurring - success",
			method: "GET",
			path:   "/api/recurring",
			mockSetup: func(m *MockFinanceService) {
				m.On("ListRecurring", mock.Anything).Return([]service.Recurring{
					{ID: 1, Description: "Monthly rent"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var recurring []service.Recurring
				err := json.Unmarshal(body, &recurring)
				require.NoError(t, err)
				assert.Len(t, recurring, 1)
				assert.Equal(t, "Monthly rent", recurring[0].Description)
			},
		},
		{
			name:   "POST /api/recurring - success",
			method: "POST",
			path:   "/api/recurring",
			body: RecurringTransactionRequest{
				Description: "Monthly rent",
				Type:        "expense",
				Amount:      1200.00,
				StartDate:   "2025-09-01",
				Interval:    "monthly",
				DayOfMonth:  intPtr(1),
				Active:      true,
			},
			mockSetup: func(m *MockFinanceService) {
				expectedStartDate, _ := time.Parse("2006-01-02", "2025-09-01")
				expectedInput := service.RecurringInput{
					Description: "Monthly rent",
					Type:        "expense",
					Amount:      1200.00,
					StartDate:   expectedStartDate,
					Interval:    "monthly",
					DayOfMonth:  intPtr(1),
					Active:      true,
				}
				m.On("CreateRecurringSimple", mock.Anything, expectedInput).Return(service.Recurring{
					ID: 1, Description: "Monthly rent",
				}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "DELETE /api/recurring/1 - success",
			method: "DELETE",
			path:   "/api/recurring/1",
			mockSetup: func(m *MockFinanceService) {
				m.On("DeleteRecurring", mock.Anything, int32(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "PUT /api/recurring/1/active - success",
			method: "PUT",
			path:   "/api/recurring/1/active",
			body: SetActiveRequest{
				Active: false,
			},
			mockSetup: func(m *MockFinanceService) {
				m.On("SetRecurringActive", mock.Anything, int32(1), false).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFinanceService)
			tt.mockSetup(mockService)

			server := setupTestServer(mockService)
			defer server.Close()

			var body []byte
			var err error
			if tt.body != nil {
				body, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBuffer(body))
			require.NoError(t, err)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				var respBody bytes.Buffer
				_, err := respBody.ReadFrom(resp.Body)
				require.NoError(t, err)
				tt.validateBody(t, respBody.Bytes())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestForecastEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "GET /api/forecast - success",
			method: "GET",
			path:   "/api/forecast",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetStartingBalance", mock.Anything).Return(5000.00, nil)
				m.On("Calculate90DayForecast", mock.Anything, 5000.00).Return([]service.DailyCashFlow{
					{Date: time.Now(), Balance: 5000.00, Change: 0},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var forecast []service.DailyCashFlow
				err := json.Unmarshal(body, &forecast)
				require.NoError(t, err)
				assert.Len(t, forecast, 1)
				assert.Equal(t, 5000.00, forecast[0].Balance)
			},
		},
		{
			name:   "GET /api/forecast/lowest - success",
			method: "GET",
			path:   "/api/forecast/lowest",
			mockSetup: func(m *MockFinanceService) {
				forecast := []service.DailyCashFlow{
					{Date: time.Now(), Balance: 5000.00, Change: 0},
				}
				m.On("GetStartingBalance", mock.Anything).Return(5000.00, nil)
				m.On("Calculate90DayForecast", mock.Anything, 5000.00).Return(forecast, nil)
				m.On("FindLowestPoint", forecast).Return(forecast[0], 0)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Contains(t, resp, "lowest_point")
				assert.Contains(t, resp, "day_index")
				assert.Equal(t, float64(0), resp["day_index"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFinanceService)
			tt.mockSetup(mockService)

			server := setupTestServer(mockService)
			defer server.Close()

			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				var respBody bytes.Buffer
				_, err := respBody.ReadFrom(resp.Body)
				require.NoError(t, err)
				tt.validateBody(t, respBody.Bytes())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestQueryParameterEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "GET /api/transactions/upcoming - default days",
			method: "GET",
			path:   "/api/transactions/upcoming",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetUpcomingTransactions", mock.Anything, 30).Return([]service.Transaction{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET /api/transactions/upcoming?days=7",
			method: "GET",
			path:   "/api/transactions/upcoming?days=7",
			mockSetup: func(m *MockFinanceService) {
				m.On("GetUpcomingTransactions", mock.Anything, 7).Return([]service.Transaction{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET /api/transactions/between - success",
			method: "GET",
			path:   "/api/transactions/between?start=2025-09-01&end=2025-09-30",
			mockSetup: func(m *MockFinanceService) {
				start, _ := time.Parse("2006-01-02", "2025-09-01")
				end, _ := time.Parse("2006-01-02", "2025-09-30")
				m.On("GetTransactionsWithRecurringsBetween", mock.Anything, start, end).Return([]service.Transaction{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/transactions/between - missing parameters",
			method:         "GET",
			path:           "/api/transactions/between?start=2025-09-01",
			mockSetup:      func(m *MockFinanceService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var errResp ErrorResponse
				err := json.Unmarshal(body, &errResp)
				require.NoError(t, err)
				assert.Contains(t, strings.ToLower(errResp.Error), "required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFinanceService)
			tt.mockSetup(mockService)

			server := setupTestServer(mockService)
			defer server.Close()

			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.validateBody != nil {
				var respBody bytes.Buffer
				_, err := respBody.ReadFrom(resp.Body)
				require.NoError(t, err)
				tt.validateBody(t, respBody.Bytes())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	mockService := new(MockFinanceService)
	server := setupTestServer(mockService)
	defer server.Close()

	// Test OPTIONS request
	req, err := http.NewRequest("OPTIONS", server.URL+"/api/transactions", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

// Helper function for int pointers
func intPtr(i int) *int {
	return &i
}
