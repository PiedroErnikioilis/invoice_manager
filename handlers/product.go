package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	Store *models.Store
}

func NewProductHandler(store *models.Store) *ProductHandler {
	return &ProductHandler{Store: store}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.Store.ListProducts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.ProductList(products).Render(r.Context(), w)
}

func (h *ProductHandler) New(w http.ResponseWriter, r *http.Request) {
	views.ProductForm(&models.Product{}, nil).Render(r.Context(), w)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	initialStock, _ := strconv.Atoi(r.FormValue("stock"))

	product := models.Product{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       price,
		Stock:       0, // Stock is set via RecordStockMovement below to avoid double counting
		Unit:        r.FormValue("unit"),
	}

	id, err := h.Store.CreateProduct(product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Initial stock movement if > 0
	if initialStock > 0 {
		h.Store.RecordStockMovement(id, initialStock, "INITIAL", "Anfangsbestand")
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	product, err := h.Store.GetProduct(id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	movements, _ := h.Store.ListStockMovements(id)

	views.ProductForm(product, movements).Render(r.Context(), w)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	// Stock is not updated here anymore, only basic info

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	// stock, _ := strconv.Atoi(r.FormValue("stock")) // Ignored

	// We need to fetch current stock to preserve it if we use UpdateProduct
	// Or we just update fields excluding stock.
	// UpdateProduct in models updates everything.

	existing, err := h.Store.GetProduct(id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	product := models.Product{
		ID:          id,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       price,
		Stock:       existing.Stock, // Preserve
		Unit:        r.FormValue("unit"),
	}

	err = h.Store.UpdateProduct(product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.Store.DeleteProduct(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	h.handleStockMovement(w, r, 1)
}

func (h *ProductHandler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	h.handleStockMovement(w, r, -1)
}

func (h *ProductHandler) handleStockMovement(w http.ResponseWriter, r *http.Request, multiplier int) {
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

	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	note := r.FormValue("note")

	if quantity <= 0 {
		http.Redirect(w, r, "/products/"+idStr+"/edit", http.StatusSeeOther)
		return
	}

	movementType := "IN"
	if multiplier < 0 {
		movementType = "OUT"
	}

	err = h.Store.RecordStockMovement(id, quantity*multiplier, movementType, note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Book as expense if requested and it is stock addition (IN)
	if multiplier > 0 && r.FormValue("book_expense") == "on" {
		product, _ := h.Store.GetProduct(id)

		cost, _ := strconv.ParseFloat(r.FormValue("cost_total"), 64)
		if cost > 0 {
			h.Store.CreateExpense(models.Expense{
				Description: "Warenzugang: " + product.Name,
				Amount:      cost,
				Date:        time.Now().Format("2006-01-02"),
				Category:    "Warenkauf",
			})
		}
	}

	http.Redirect(w, r, "/products/"+idStr+"/edit", http.StatusSeeOther)
}
