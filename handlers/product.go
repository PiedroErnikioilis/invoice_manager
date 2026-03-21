package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"log/slog"
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
	slog.Debug("Processing List products request", "method", r.Method)
	products, err := h.Store.ListProducts()
	if err != nil {
		slog.Error("Failed to list products", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Successfully listed products", "count", len(products))
	views.ProductList(products).Render(r.Context(), w)
}

func (h *ProductHandler) DownloadInventoryPDF(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing DownloadInventoryPDF request", "method", r.Method)
	slog.Info("Generating inventory PDF")
	products, err := h.Store.ListProducts()
	if err != nil {
		slog.Error("Failed to list products for inventory PDF", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		slog.Error("Failed to load settings for inventory PDF", "error", err)
		http.Error(w, "Could not load settings", http.StatusInternalServerError)
		return
	}

	path, err := services.GenerateInventoryPDFHTML(products, &settings)
	if err != nil {
		slog.Error("Failed to generate inventory PDF", "error", err)
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=inventarliste.pdf")
	slog.Debug("Serving inventory PDF", "path", path)
	http.ServeFile(w, r, path)
}

func (h *ProductHandler) New(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing New product request", "method", r.Method)
	views.ProductForm(&models.Product{}, nil).Render(r.Context(), w)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing Create product request", "method", r.Method)
	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse product form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed product form")

	price := parseDecimal(r.FormValue("price"))
	initialStock, _ := strconv.Atoi(r.FormValue("stock"))
	minStock, _ := strconv.Atoi(r.FormValue("min_stock"))

	product := models.Product{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       price,
		Stock:       0, // Stock is set via RecordStockMovement below to avoid double counting
		MinStock:    minStock,
		Unit:        r.FormValue("unit"),
	}

	slog.Info("Creating product", "name", product.Name, "price", product.Price)
	id, err := h.Store.CreateProduct(product)
	if err != nil {
		slog.Error("Failed to create product", "name", product.Name, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Initial stock movement if > 0
	if initialStock > 0 {
		slog.Info("Recording initial stock movement", "id", id, "quantity", initialStock)
		h.Store.RecordStockMovement(id, initialStock, "INITIAL", "Anfangsbestand")
	}

	slog.Info("Product created successfully", "id", id, "name", product.Name)
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse product ID for edit", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing Edit product request", "id", id, "method", r.Method)

	slog.Debug("Fetching product for edit", "id", id)
	product, err := h.Store.GetProduct(id)
	if err != nil {
		slog.Error("Product not found for edit", "id", id, "error", err)
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	slog.Debug("Fetching stock movements for product edit", "id", id)
	movements, _ := h.Store.ListStockMovements(id)

	slog.Debug("Successfully fetched product and movements for edit", "id", id, "movements_count", len(movements))
	views.ProductForm(product, movements).Render(r.Context(), w)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse product ID for update", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing Update product request", "id", id, "method", r.Method)

	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse product update form", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed product update form", "id", id)

	// Stock is not updated here anymore, only basic info

	price := parseDecimal(r.FormValue("price"))
	minStock, _ := strconv.Atoi(r.FormValue("min_stock"))

	slog.Debug("Fetching existing product for update", "id", id)
	existing, err := h.Store.GetProduct(id)
	if err != nil {
		slog.Error("Product not found for update", "id", id, "error", err)
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	product := models.Product{
		ID:          id,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       price,
		Stock:       existing.Stock, // Preserve
		MinStock:    minStock,
		Unit:        r.FormValue("unit"),
	}

	slog.Info("Updating product", "id", id, "name", product.Name)
	err = h.Store.UpdateProduct(product)
	if err != nil {
		slog.Error("Failed to update product", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Product updated successfully", "id", id)
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse product ID for deletion", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing Delete product request", "id", id, "method", r.Method)

	slog.Info("Deleting product", "id", id)
	err = h.Store.DeleteProduct(id)
	if err != nil {
		slog.Error("Failed to delete product", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Product deleted successfully", "id", id)
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *ProductHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing AddStock request", "method", r.Method)
	h.handleStockMovement(w, r, 1)
}

func (h *ProductHandler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing RemoveStock request", "method", r.Method)
	h.handleStockMovement(w, r, -1)
}

func (h *ProductHandler) handleStockMovement(w http.ResponseWriter, r *http.Request, multiplier int) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse product ID for stock movement", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Handling stock movement", "id", id, "multiplier", multiplier)

	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse stock movement form", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed stock movement form", "id", id)

	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	note := r.FormValue("note")

	if quantity <= 0 {
		slog.Debug("Skipping stock movement for zero or negative quantity", "id", id, "quantity", quantity)
		http.Redirect(w, r, "/products/"+idStr+"/edit", http.StatusSeeOther)
		return
	}

	movementType := "IN"
	if multiplier < 0 {
		movementType = "OUT"
	}

	slog.Info("Recording manual stock movement", "id", id, "quantity", quantity*multiplier, "type", movementType, "note", note)
	err = h.Store.RecordStockMovement(id, quantity*multiplier, movementType, note)
	if err != nil {
		slog.Error("Failed to record stock movement", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Book as expense if requested and it is stock addition (IN)
	if multiplier > 0 && r.FormValue("book_expense") == "on" {
		slog.Debug("Booking stock addition as expense", "product_id", id)
		product, _ := h.Store.GetProduct(id)

		cost := parseDecimal(r.FormValue("cost_total"))
		if cost > 0 {
			slog.Info("Booking stock addition as expense", "product_id", id, "cost", cost)
			expense := models.Expense{
				Description: "Warenzugang: " + product.Name,
				Amount:      cost,
				Date:        time.Now().Format("2006-01-02"),
			}
			if catID, err := h.Store.CreateExpenseCategory("Warenkauf"); err == nil {
				expense.CategoryID = &catID
				slog.Debug("Created/resolved 'Warenkauf' category for stock addition expense", "id", catID)
			}
			expID, err := h.Store.CreateExpense(expense)
			if err != nil {
				slog.Error("Failed to create expense for stock addition", "product_id", id, "error", err)
			} else {
				slog.Info("Successfully booked stock addition as expense", "product_id", id, "expense_id", expID)
			}
		} else {
			slog.Debug("Skipping expense booking for zero cost", "product_id", id)
		}
	}

	http.Redirect(w, r, "/products/"+idStr+"/edit", http.StatusSeeOther)
}
