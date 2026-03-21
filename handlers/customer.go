package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	Store *models.Store
}

func NewCustomerHandler(store *models.Store) *CustomerHandler {
	return &CustomerHandler{Store: store}
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing List customers request", "method", r.Method)
	customers, err := h.Store.ListCustomers()
	if err != nil {
		slog.Error("Failed to list customers", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Customers listed successfully", "count", len(customers))
	views.CustomerList(customers).Render(r.Context(), w)
}

func (h *CustomerHandler) New(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing New customer request", "method", r.Method)
	settings, _ := h.Store.GetAppSettings()
	custNum := models.FormatDocumentNumber(settings.CustomerIDSchema, settings.NextCustomerID)
	slog.Debug("Generated next customer number", "customer_number", custNum)

	views.CustomerForm(&models.Customer{
		CustomerNumber: custNum,
	}).Render(r.Context(), w)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing Create customer request", "method", r.Method)
	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse customer form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customer := models.Customer{
		CustomerNumber: r.FormValue("customer_number"),
		Name:           r.FormValue("name"),
		Address:        r.FormValue("address"),
		Email:          r.FormValue("email"),
	}
	slog.Debug("Customer form parsed", "name", customer.Name, "customer_number", customer.CustomerNumber)

	slog.Info("Creating customer", "customer_number", customer.CustomerNumber, "name", customer.Name)

	// Increment customer number if it matches the auto-generated one
	settings, _ := h.Store.GetAppSettings()
	expectedNum := models.FormatDocumentNumber(settings.CustomerIDSchema, settings.NextCustomerID)
	if customer.CustomerNumber == expectedNum {
		slog.Debug("Auto-incrementing next customer ID", "current", expectedNum)
		h.Store.IncrementNextCustomerID()
	}

	id, err := h.Store.CreateCustomer(&customer)
	if err != nil {
		slog.Error("Failed to create customer", "customer_number", customer.CustomerNumber, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Customer created successfully", "id", id, "customer_number", customer.CustomerNumber)
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Invalid ID for customer edit", "id", idStr)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	slog.Debug("Processing Edit customer request", "id", id, "method", r.Method)
	customer, err := h.Store.GetCustomer(id)
	if err != nil {
		slog.Error("Customer not found for edit", "id", id, "error", err)
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}
	slog.Debug("Customer loaded for edit", "id", id, "name", customer.Name)

	views.CustomerForm(customer).Render(r.Context(), w)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Invalid ID for customer update", "id", idStr)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	slog.Debug("Processing Update customer request", "id", id, "method", r.Method)
	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse customer update form", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customer := models.Customer{
		ID:             id,
		CustomerNumber: r.FormValue("customer_number"),
		Name:           r.FormValue("name"),
		Address:        r.FormValue("address"),
		Email:          r.FormValue("email"),
	}
	slog.Debug("Customer update form parsed", "id", id, "name", customer.Name)

	slog.Info("Updating customer", "id", id, "customer_number", customer.CustomerNumber)
	err = h.Store.UpdateCustomer(&customer)
	if err != nil {
		slog.Error("Failed to update customer", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Customer updated successfully", "id", id)
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Invalid ID for customer delete", "id", idStr)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	slog.Debug("Processing Delete customer request", "id", id, "method", r.Method)
	slog.Info("Deleting customer", "id", id)
	err = h.Store.DeleteCustomer(id)
	if err != nil {
		slog.Error("Failed to delete customer", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Customer deleted successfully", "id", id)
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}
