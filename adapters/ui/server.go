package ui

import (
	"encoding/json"
	"nannypayroll/app"
	"net/http"
	"strconv"
)

type PaycheckResponse struct {
	Gross            float64 `json:"gross"`
	OASDI            float64 `json:"oasdi"`
	Medicare         float64 `json:"medicare"`
	FederalIncomeTax float64 `json:"federal_income_tax"`
	SDI              float64 `json:"sdi"`
	StateIncomeTax   float64 `json:"state_income_tax"`
	NetPay           float64 `json:"net_pay"`
}

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("/paycheck", s.handlePaycheck)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handlePaycheck(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.ParseFloat(r.URL.Query().Get("hours"), 64)
	rate, _ := strconv.ParseFloat(r.URL.Query().Get("rate"), 64)
	payroll_service := app.NewPayrollService()
	p := payroll_service.CalculatePaycheck()
	json.NewEncoder(w).Encode(PaycheckResponse{
		Gross:            p.Gross,
		OASDI:            p.OASDI,
		Medicare:         p.Medicare,
		FederalIncomeTax: p.FederalIncomeTax,
		SDI:              p.StateDisabilityInsurance,
		StateIncomeTax:   p.StateIncomeTax,
		NetPay:           p.NetPay,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

}
