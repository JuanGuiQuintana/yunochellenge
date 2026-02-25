package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/juanatsap/chargeback-api/internal/api/response"
	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/service"
)

// ChargebackHandler handles chargeback query requests.
type ChargebackHandler struct {
	querySvc *service.QueryService
}

// NewChargebackHandler constructs a ChargebackHandler with the given service.
func NewChargebackHandler(querySvc *service.QueryService) *ChargebackHandler {
	return &ChargebackHandler{querySvc: querySvc}
}

// GetByID handles GET /api/v1/chargebacks/{chargeback_id}.
func (h *ChargebackHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chargeback_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid chargeback_id")
		return
	}

	cb, err := h.querySvc.GetChargeback(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "chargeback not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, cb)
}

// List handles GET /api/v1/chargebacks.
func (h *ChargebackHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseChargebackFilter(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	chargebacks, total, err := h.querySvc.ListChargebacks(r.Context(), filter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, response.NewPaginated(chargebacks, total, filter.Page, filter.PerPage))
}

// Summary handles GET /api/v1/chargebacks/summary.
func (h *ChargebackHandler) Summary(w http.ResponseWriter, r *http.Request) {
	var merchantID *uuid.UUID
	if midStr := r.URL.Query().Get("merchant_id"); midStr != "" {
		id, err := uuid.Parse(midStr)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid merchant_id")
			return
		}
		merchantID = &id
	}

	summary, err := h.querySvc.Summary(r.Context(), merchantID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, summary)
}

// parseChargebackFilter parses query params into a ChargebackFilter.
func parseChargebackFilter(r *http.Request) (domain.ChargebackFilter, error) {
	q := r.URL.Query()
	filter := domain.ChargebackFilter{}

	if v := q.Get("merchant_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filter, errors.New("invalid merchant_id")
		}
		filter.MerchantID = &id
	}
	if v := q.Get("score_min"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, errors.New("invalid score_min")
		}
		filter.ScoreMin = &n
	}
	if v := q.Get("deadline_hours"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, errors.New("invalid deadline_hours")
		}
		filter.DeadlineHours = &n
	}
	if v := q.Get("deadline_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, errors.New("invalid deadline_before: use RFC3339 format")
		}
		filter.DeadlineBefore = &t
	}
	if v := q.Get("flags"); v != "" {
		filter.Flags = splitFlags(v)
	}
	if v := q.Get("flags_match"); v != "" {
		filter.FlagsMatch = domain.FlagsMatch(v)
	}
	if v := q.Get("reason_code"); v != "" {
		filter.ReasonCode = &v
	}
	if v := q.Get("processor_name"); v != "" {
		filter.ProcessorName = &v
	}
	if v := q.Get("status"); v != "" {
		s := domain.ChargebackStatus(v)
		filter.Status = &s
	}
	if v := q.Get("currency"); v != "" {
		filter.Currency = &v
	}
	if v := q.Get("amount_min"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filter, errors.New("invalid amount_min")
		}
		filter.AmountMin = &n
	}
	if v := q.Get("amount_max"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filter, errors.New("invalid amount_max")
		}
		filter.AmountMax = &n
	}
	if v := q.Get("sort_by"); v != "" {
		filter.SortBy = domain.SortField(v)
	}
	if v := q.Get("sort_order"); v != "" {
		filter.SortOrder = domain.SortOrder(v)
	}
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, errors.New("invalid page")
		}
		filter.Page = n
	}
	if v := q.Get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, errors.New("invalid per_page")
		}
		filter.PerPage = n
	}

	filter.Normalize()
	return filter, nil
}

func splitFlags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
