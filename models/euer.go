package models

import "fmt"

type CategoryStat struct {
	Name       string
	Total      float64
	Percentage float64
}

type EuerStats struct {
	Year          int
	TotalIncomeNet   float64
	TotalIncomeVat   float64
	TotalIncomeGross float64
	TotalExpensesNet   float64
	TotalExpensesTax   float64
	TotalExpensesGross float64
	Profit        float64
	VatPayable    float64 // Zahllast (Income VAT - Expense Tax)
	Expenses      []Expense
	Invoices      []Invoice
	CategoryStats []CategoryStat
}

// GetEuerStats returns income/expense statistics filtered by year.
func (s *Store) GetEuerStats(year int) (*EuerStats, error) {
	stats := &EuerStats{Year: year}

	// 1. Calculate Income (Paid Invoices)
	var dateFilter string
	if year > 0 {
		dateFilter = fmt.Sprintf(" AND date LIKE '%d-%%'", year)
	}

	query := fmt.Sprintf(`
		SELECT id, invoice_number, date, recipient_name, tax_rate, is_small_business, status
		FROM invoices
		WHERE status = 'Bezahlt'%s
		ORDER BY date DESC
	`, dateFilter)

	rows, err := s.DB.Query(query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var i Invoice
			if err := rows.Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.RecipientName, &i.TaxRate, &i.IsSmallBusiness, &i.Status); err != nil {
				continue
			}

			fullInv, err := s.GetInvoice(i.ID)
			if err == nil {
				stats.TotalIncomeNet += fullInv.TotalNet()
				stats.TotalIncomeVat += fullInv.TaxAmount()
				stats.TotalIncomeGross += fullInv.TotalGross()
				stats.Invoices = append(stats.Invoices, *fullInv)
			}
		}
	}

	// 1b. Calculate Credits (Credit Notes)
	creditQuery := fmt.Sprintf(`
		SELECT id, credit_note_number, date, recipient_name, tax_rate, is_small_business, status
		FROM credit_notes
		WHERE status != 'Entwurf'%s
		ORDER BY date DESC
	`, dateFilter)

	cRows, err := s.DB.Query(creditQuery)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cn CreditNote
			if err := cRows.Scan(&cn.ID, &cn.CreditNoteNumber, &cn.Date, &cn.RecipientName, &cn.TaxRate, &cn.IsSmallBusiness, &cn.Status); err != nil {
				continue
			}

			fullCn, err := s.GetCreditNote(cn.ID)
			if err == nil {
				stats.TotalIncomeNet += fullCn.TotalNet()
				stats.TotalIncomeVat += fullCn.TaxAmount()
				stats.TotalIncomeGross += fullCn.TotalGross()
			}
		}
	}

	// 2. Load Expenses list
	stats.Expenses, _ = s.ListExpenses(year)
	for _, e := range stats.Expenses {
		stats.TotalExpensesNet += e.Net()
		stats.TotalExpensesTax += e.Tax()
		stats.TotalExpensesGross += e.Amount
	}

	// 3. Category Stats (using Net)
	catSums := make(map[string]float64)
	for _, e := range stats.Expenses {
		catName := e.CategoryName
		if catName == "" {
			catName = "Sonstige"
		}
		catSums[catName] += e.Net()
	}

	for name, sum := range catSums {
		cs := CategoryStat{Name: name, Total: sum}
		if stats.TotalExpensesNet > 0 {
			cs.Percentage = (cs.Total / stats.TotalExpensesNet) * 100
		}
		stats.CategoryStats = append(stats.CategoryStats, cs)
	}

	stats.Profit = stats.TotalIncomeNet - stats.TotalExpensesNet
	stats.VatPayable = stats.TotalIncomeVat - stats.TotalExpensesTax

	return stats, nil
}

// GetAvailableYears returns all years that have invoices or expenses.
func (s *Store) GetAvailableYears() ([]int, error) {
	rows, err := s.DB.Query(`
		SELECT DISTINCT year FROM (
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM invoices WHERE date != ''
			UNION
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM expenses WHERE date != ''
            UNION
            SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM credit_notes WHERE date != ''
		) ORDER BY year DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			continue
		}
		if y > 0 {
			years = append(years, y)
		}
	}
	return years, nil
}
