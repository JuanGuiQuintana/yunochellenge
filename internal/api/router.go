package api

import (
	"net/http"

	"github.com/juanatsap/chargeback-api/internal/api/handler"
	"github.com/juanatsap/chargeback-api/internal/api/middleware"
)

// NewRouter creates the HTTP mux with all routes registered.
// IMPORTANT: /chargebacks/summary MUST be registered before /chargebacks/{chargeback_id}
// because Go 1.22 net/http uses specificity-based matching:
// exact path segments beat wildcard segments.
func NewRouter(
	ingestH *handler.IngestHandler,
	chargebackH *handler.ChargebackHandler,
	merchantH *handler.MerchantHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// Ingestion
	mux.HandleFunc("POST /api/v1/chargebacks/ingest/{processor_name}", ingestH.Handle)

	// Chargebacks — summary BEFORE {id} to avoid shadowing
	mux.HandleFunc("GET /api/v1/chargebacks/summary", chargebackH.Summary)
	mux.HandleFunc("GET /api/v1/chargebacks/{chargeback_id}", chargebackH.GetByID)
	mux.HandleFunc("GET /api/v1/chargebacks", chargebackH.List)

	// Merchants
	mux.HandleFunc("GET /api/v1/merchants/{merchant_id}/chargebacks", merchantH.ListChargebacks)
	mux.HandleFunc("GET /api/v1/merchants/{merchant_id}/stats", merchantH.GetStats)

	// Apply middleware chain: Recovery → RequestID → Logging
	return middleware.Recovery(middleware.RequestID(middleware.Logging(mux)))
}
