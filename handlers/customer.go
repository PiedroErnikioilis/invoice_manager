package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
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
	customers, err := h.Store.ListCustomers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.CustomerList(customers).Render(r.Context(), w)
}

func (h *CustomerHandler) New(w http.ResponseWriter, r *http.Request) {
	views.CustomerForm(&models.Customer{}).Render(r.Context(), w)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customer := models.Customer{
		Name:    r.FormValue("name"),
		Address: r.FormValue("address"),
		Email:   r.FormValue("email"),
	}

	_, err := h.Store.CreateCustomer(customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	customer, err := h.Store.GetCustomer(id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	views.CustomerForm(customer).Render(r.Context(), w)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customer := models.Customer{
		ID:      id,
		Name:    r.FormValue("name"),
		Address: r.FormValue("address"),
		Email:   r.FormValue("email"),
	}

	err = h.Store.UpdateCustomer(customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.Store.DeleteCustomer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}
