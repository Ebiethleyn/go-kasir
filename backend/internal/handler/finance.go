package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-kasir/backend/internal/middleware"
	"github.com/go-kasir/backend/internal/pkg/response"
	"github.com/go-kasir/backend/internal/service"
)

func (h *Handlers) FinanceSummary(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Finance.Summary(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

func (h *Handlers) ListFinanceEntries(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	entries, err := h.Finance.ListEntries(from, to, atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": entries})
}

func (h *Handlers) CreateFinanceEntry(w http.ResponseWriter, r *http.Request) {
	var input service.FinanceEntryInput
	if err := decodeJSON(r, &input); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	entry, err := h.Finance.CreateEntry(middleware.GetOwnerID(r), input)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, "catatan keuangan dibuat", entry)
}

func (h *Handlers) DeleteFinanceEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(w, "id catatan tidak valid")
		return
	}
	if err := h.Finance.DeleteEntry(uint(id), middleware.GetOwnerID(r)); err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "catatan keuangan dihapus", nil)
}
