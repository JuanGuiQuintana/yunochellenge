package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/juanatsap/chargeback-api/internal/api/response"
	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/service"
)

// MerchantHandler handles merchant-scoped query requests.
type MerchantHandler struct {
	querySvc *service.QueryService
}

// NewMerchantHandler constructs a MerchantHandler with the given service.
func NewMerchantHandler(querySvc *service.QueryService) *MerchantHandler {
	return &MerchantHandler{querySvc: querySvc}
}

// ListChargebacks handles GET /api/v1/merchants/{merchant_id}/chargebacks.
func (h *MerchantHandler) ListChargebacks(w http.ResponseWriter, r *http.Request) {
	merchantID, err := parseMerchantID(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid merchant_id")
		return
	}

	filter, err := parseChargebackFilter(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	filter.MerchantID = &merchantID

	chargebacks, total, err := h.querySvc.ListChargebacks(r.Context(), filter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, response.NewPaginated(chargebacks, total, filter.Page, filter.PerPage))
}

// GetStats handles GET /api/v1/merchants/{merchant_id}/stats.
func (h *MerchantHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	merchantID, err := parseMerchantID(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid merchant_id")
		return
	}

	stats, err := h.querySvc.GetMerchantStats(r.Context(), merchantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "merchant not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response.WriteJSON(w, http.StatusOK, stats)
}

func parseMerchantID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("merchant_id"))
}
