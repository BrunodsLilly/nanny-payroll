package functionaltests

import (
	"nannypayroll/adapters/ui"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A user submits hours worked and hourly rate,
// and receives a paycheck breakdown with gross, deductions, and net pay.
func TestCalculatePaycheck(t *testing.T) {
	server := ui.NewServer()
	req := httptest.NewRequest(http.MethodGet, "/paycheck?hours=40&rate=20", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, `"gross":800`) {
		t.Errorf("expected gross pay 800 in response, got %s", body)
	}
	if !strings.Contains(body, `"net_pay":587.6`) {
		t.Errorf("expected net pay 587.6 in response, got %s", body)
	}
}
