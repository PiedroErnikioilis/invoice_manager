package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"log/slog"
	"net/http"
)

type StatsHandler struct {
	Store *models.Store
}

func NewStatsHandler(store *models.Store) *StatsHandler {
	return &StatsHandler{Store: store}
}

func (h *StatsHandler) View(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing View statistics request", "method", r.Method)
	stats, err := h.Store.GetStats()
	if err != nil {
		slog.Error("Failed to get stats", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Statistics loaded successfully")
	views.StatsDashboard(stats).Render(r.Context(), w)
}
