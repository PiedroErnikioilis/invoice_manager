package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"net/http"
)

type StatsHandler struct {
	Store *models.Store
}

func NewStatsHandler(store *models.Store) *StatsHandler {
	return &StatsHandler{Store: store}
}

func (h *StatsHandler) View(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Store.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.StatsDashboard(stats).Render(r.Context(), w)
}
