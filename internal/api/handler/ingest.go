package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/juanatsap/chargeback-api/internal/api/response"
	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/service"
)

// IngestHandler handles chargeback ingestion requests.
type IngestHandler struct {
	ingestSvc *service.IngestService
}

// NewIngestHandler constructs an IngestHandler with the given service.
func NewIngestHandler(ingestSvc *service.IngestService) *IngestHandler {
	return &IngestHandler{ingestSvc: ingestSvc}
}

// ingestResponse is the 202 Accepted response body.
type ingestResponse struct {
	ChargebackID   string                `json:"chargeback_id"`
	RiskScore      int                   `json:"risk_score"`
	ScoreBreakdown domain.ScoreBreakdown `json:"score_breakdown"`
	Flags          []string              `json:"flags"`
}

// Handle processes POST /api/v1/chargebacks/ingest/{processor_name}.
func (h *IngestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	processorName := r.PathValue("processor_name")
	if processorName == "" {
		response.WriteError(w, http.StatusBadRequest, "processor_name is required")
		return
	}

	signature := r.Header.Get("X-Processor-Signature")

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	result, err := h.ingestSvc.Ingest(r.Context(), processorName, signature, body)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			response.WriteError(w, http.StatusUnauthorized, "invalid signature")
		case errors.Is(err, domain.ErrRateLimitExceeded):
			response.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
		case errors.Is(err, domain.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "processor or resource not found")
		case errors.Is(err, domain.ErrInvalidInput):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	flags := result.Flags
	if flags == nil {
		flags = []string{}
	}

	resp := ingestResponse{
		ChargebackID:   result.ChargebackID.String(),
		RiskScore:      result.RiskScore,
		ScoreBreakdown: result.ScoreBreakdown,
		Flags:          flags,
	}

	// Idempotent: 200 for duplicate, 202 for new
	status := http.StatusAccepted
	if result.IsDuplicate {
		status = http.StatusOK
	}

	response.WriteJSON(w, status, resp)
}
