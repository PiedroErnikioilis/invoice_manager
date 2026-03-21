package models

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

// SeedDemoData füllt eine leere Datenbank mit realistischen Beispieldaten.
// Gibt einen Fehler zurück, falls bereits Daten vorhanden sind (Sicherheitscheck).
func (s *Store) SeedDemoData() error {
	slog.Info("Starting demo data seeding")
	// Sicherheitscheck: Nur in leere DB einfügen
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM customers").Scan(&count); err != nil {
		slog.Error("Security check failed for customers during seeding", "error", err)
		return fmt.Errorf("Sicherheitscheck fehlgeschlagen: %w", err)
	}
	if count > 0 {
		slog.Warn("Seeding aborted: database already contains customers", "count", count)
		return fmt.Errorf("Datenbank enthält bereits %d Kunden – Demo-Daten werden nicht eingefügt", count)
	}
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM invoices").Scan(&count); err != nil {
		slog.Error("Security check failed for invoices during seeding", "error", err)
		return fmt.Errorf("Sicherheitscheck fehlgeschlagen: %w", err)
	}
	if count > 0 {
		slog.Warn("Seeding aborted: database already contains invoices", "count", count)
		return fmt.Errorf("Datenbank enthält bereits %d Rechnungen – Demo-Daten werden nicht eingefügt", count)
	}

	slog.Debug("Creating demo settings")
	if err := s.seedSettings(); err != nil {
		return fmt.Errorf("Settings: %w", err)
	}

	slog.Debug("Creating demo expense categories")
	categoryIDs, err := s.seedExpenseCategories()
	if err != nil {
		return fmt.Errorf("Ausgabenkategorien: %w", err)
	}

	slog.Debug("Creating demo customers")
	customerIDs, err := s.seedCustomers()
	if err != nil {
		return fmt.Errorf("Kunden: %w", err)
	}

	slog.Debug("Creating demo products")
	productIDs, err := s.seedProducts()
	if err != nil {
		return fmt.Errorf("Produkte: %w", err)
	}

	slog.Debug("Creating demo invoices")
	if err := s.seedInvoices(customerIDs, productIDs); err != nil {
		return fmt.Errorf("Rechnungen: %w", err)
	}

	slog.Debug("Creating big demo invoice with 20 positions")
	if err := s.seedBigInvoice(customerIDs, productIDs); err != nil {
		return fmt.Errorf("Große Rechnung: %w", err)
	}

	slog.Debug("Creating demo quotes")
	if err := s.seedQuotes(customerIDs, productIDs); err != nil {
		return fmt.Errorf("Angebote: %w", err)
	}

	slog.Debug("Creating demo credit notes")
	if err := s.seedCreditNotes(customerIDs, productIDs); err != nil {
		return fmt.Errorf("Gutschriften: %w", err)
	}

	slog.Debug("Creating demo expenses")
	if err := s.seedExpenses(categoryIDs); err != nil {
		return fmt.Errorf("Ausgaben: %w", err)
	}

	slog.Debug("Creating demo recurring expenses")
	if err := s.seedRecurringExpenses(categoryIDs); err != nil {
		return fmt.Errorf("Wiederkehrende Ausgaben: %w", err)
	}

	slog.Info("Demo data seeding completed successfully")
	return nil
}

func (s *Store) seedSettings() error {
	settings := AppSettings{
		SenderName:           "Demo GmbH",
		SenderAddress:        "Musterstraße 1\n12345 Berlin\nDeutschland",
		NextInvoiceNumber:    100,
		BankName:             "Demo Bank AG",
		IBAN:                 "DE89 3704 0044 0532 0130 00",
		BIC:                  "COBADEFFXXX",
		Website:              "https://demo-gmbh.example.de",
		Email:                "info@demo-gmbh.example.de",
		PDFOutputPath:        "./invoices/",
		DefaultSmallBusiness: false,
		BackupPath:           "./backups",
		BackupMaxCount:       10,
		AutoBackupEnabled:    true,
		BackupMinIntervalHours: 24,
	}
	return s.SaveAppSettings(settings)
}

func (s *Store) seedExpenseCategories() ([]int, error) {
	categories := []string{
		"Büromaterial",
		"Software & Lizenzen",
		"Miete & Nebenkosten",
		"Versicherungen",
		"Reisekosten",
		"Telefon & Internet",
		"Fortbildung",
		"Fahrzeugkosten",
		"Marketing & Werbung",
		"Bewirtung",
	}

	var ids []int
	for _, name := range categories {
		id, err := s.CreateExpenseCategory(name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) seedCustomers() ([]int, error) {
	customers := []Customer{
		{Name: "Müller & Söhne GmbH", Address: "Industriestr. 15\n80333 München", Email: "kontakt@mueller-soehne.example.de"},
		{Name: "Schmidt Consulting AG", Address: "Königsallee 42\n40212 Düsseldorf", Email: "info@schmidt-consulting.example.de"},
		{Name: "Weber Elektronik", Address: "Technikweg 7\n70173 Stuttgart", Email: "bestellung@weber-elektronik.example.de"},
		{Name: "Fischer & Partner", Address: "Hafenstr. 23\n20457 Hamburg", Email: "office@fischer-partner.example.de"},
		{Name: "Bauer Maschinenbau GmbH", Address: "Werksgelände 3\n90402 Nürnberg", Email: "einkauf@bauer-maschinenbau.example.de"},
		{Name: "Hofmann IT-Services", Address: "Softwarepark 11\n76131 Karlsruhe", Email: "service@hofmann-it.example.de"},
		{Name: "Klein & Groß OHG", Address: "Marktplatz 8\n50667 Köln", Email: "bestellung@klein-gross.example.de"},
		{Name: "Schneider Logistik", Address: "Frachtweg 55\n28195 Bremen", Email: "logistik@schneider-log.example.de"},
		{Name: "Wagner Architekturbüro", Address: "Baumeisterstr. 19\n01067 Dresden", Email: "planung@wagner-architektur.example.de"},
		{Name: "Becker Medizintechnik", Address: "Gesundheitsallee 4\n30159 Hannover", Email: "vertrieb@becker-medtech.example.de"},
		{Name: "Zimmermann Textil GmbH", Address: "Stoffweg 12\n04109 Leipzig", Email: "order@zimmermann-textil.example.de"},
		{Name: "Krüger Gastronomie", Address: "Genussmeile 6\n60311 Frankfurt", Email: "info@krueger-gastro.example.de"},
		{Name: "Lehmann Druck & Verlag", Address: "Gutenbergstr. 21\n55116 Mainz", Email: "auftrag@lehmann-druck.example.de"},
		{Name: "Hartmann Sicherheitstechnik", Address: "Schutzweg 9\n44135 Dortmund", Email: "kontakt@hartmann-sicher.example.de"},
		{Name: "Schmitz Garten- und Landschaftsbau", Address: "Grüner Weg 33\n53111 Bonn", Email: "anfrage@schmitz-gala.example.de"},
		{Name: "Wolf Elektrotechnik", Address: "Stromstr. 17\n99084 Erfurt", Email: "info@wolf-elektro.example.de"},
		{Name: "Braun Möbelmanufaktur", Address: "Tischlerstr. 5\n18055 Rostock", Email: "werkstatt@braun-moebel.example.de"},
		{Name: "Schröder Immobilien", Address: "Maklergasse 14\n24103 Kiel", Email: "immo@schroeder-immo.example.de"},
		{Name: "Neumann Software Solutions", Address: "Codestr. 88\n10115 Berlin", Email: "dev@neumann-software.example.de"},
		{Name: "Schwarz Lebensmittel GmbH", Address: "Frischeweg 2\n68159 Mannheim", Email: "einkauf@schwarz-food.example.de"},
	}

	var ids []int
	for i, c := range customers {
		c.CustomerNumber = fmt.Sprintf("KD-%04d", i+1)
		id, err := s.CreateCustomer(&c)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	// Update next customer id
	_ = s.SetSetting("next_customer_id", "21")
	return ids, nil
}

func (s *Store) seedProducts() ([]int, error) {
	products := []Product{
		{Name: "Webseiten-Redesign", Description: "Komplettes Redesign einer Unternehmenswebseite", Price: 4500.00, Stock: 0, MinStock: 0, Unit: "Projekt"},
		{Name: "SEO-Optimierung", Description: "Suchmaschinenoptimierung (monatlich)", Price: 850.00, Stock: 0, MinStock: 0, Unit: "Monat"},
		{Name: "Logo-Design", Description: "Professionelles Logodesign inkl. Styleguide", Price: 1200.00, Stock: 0, MinStock: 0, Unit: "Stk"},
		{Name: "Visitenkarten (500 Stk)", Description: "Druck Visitenkarten, beidseitig, matt", Price: 89.90, Stock: 200, MinStock: 50, Unit: "Packung"},
		{Name: "Flyer DIN A5 (1000 Stk)", Description: "Flyerdruck, 4-farbig, 135g", Price: 149.00, Stock: 150, MinStock: 30, Unit: "Packung"},
		{Name: "Hosting-Paket Business", Description: "Webhosting inkl. SSL, 50GB, E-Mail", Price: 29.90, Stock: 0, MinStock: 0, Unit: "Monat"},
		{Name: "WordPress-Plugin Lizenz", Description: "Premium-Plugin Jahreslizenz", Price: 79.00, Stock: 100, MinStock: 20, Unit: "Lizenz"},
		{Name: "IT-Support Stunde", Description: "Technischer Support je Stunde", Price: 95.00, Stock: 0, MinStock: 0, Unit: "Std"},
		{Name: "Server-Wartung", Description: "Monatliche Serverwartung und Updates", Price: 250.00, Stock: 0, MinStock: 0, Unit: "Monat"},
		{Name: "SSL-Zertifikat", Description: "Wildcard SSL-Zertifikat (1 Jahr)", Price: 199.00, Stock: 50, MinStock: 10, Unit: "Stk"},
		{Name: "Domainregistrierung .de", Description: "Domain .de für 1 Jahr", Price: 12.00, Stock: 0, MinStock: 0, Unit: "Jahr"},
		{Name: "E-Mail Marketing Kampagne", Description: "Newsletter-Kampagne Setup und Versand", Price: 350.00, Stock: 0, MinStock: 0, Unit: "Kampagne"},
		{Name: "Social Media Betreuung", Description: "Monatliche Social-Media-Betreuung", Price: 650.00, Stock: 0, MinStock: 0, Unit: "Monat"},
		{Name: "Fotoshooting (halber Tag)", Description: "Professionelles Produktfotoshooting", Price: 750.00, Stock: 0, MinStock: 0, Unit: "Halbtag"},
		{Name: "USB-Stick 32GB (bedruckt)", Description: "Werbe-USB-Sticks mit Logoaufdruck", Price: 8.50, Stock: 500, MinStock: 100, Unit: "Stk"},
	}

	var ids []int
	for _, p := range products {
		id, err := s.CreateProduct(p)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
		// Initiale Lagerbewegung für Produkte mit Bestand
		if p.Stock > 0 {
			if err := s.RecordStockMovement(id, 0, "PURCHASE", "Anfangsbestand Demo"); err != nil {
				return nil, err
			}
			// Stock is already set in CreateProduct, RecordStockMovement with 0 just logs it
			// Actually we need to NOT double-count. CreateProduct sets stock directly.
			// So we just record a movement with 0 for audit trail.
		}
	}
	return ids, nil
}

func (s *Store) seedInvoices(customerIDs, productIDs []int) error {
	r := rand.New(rand.NewSource(42)) // Deterministic for reproducibility
	senderName := "Demo GmbH"
	senderAddr := "Musterstraße 1\n12345 Berlin\nDeutschland"

	statuses := []string{"Entwurf", "Offen", "Bezahlt", "Bezahlt", "Bezahlt", "Storniert"}
	now := time.Now()

	for i := 0; i < 50; i++ {
		custIdx := r.Intn(len(customerIDs))
		custID := customerIDs[custIdx]
		cust, err := s.GetCustomer(custID)
		if err != nil {
			return err
		}

		// Date spread over the last 12 months
		daysAgo := r.Intn(365)
		date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		status := statuses[r.Intn(len(statuses))]
		invoiceNum := fmt.Sprintf("RE-%04d", i+1)

		numItems := 1 + r.Intn(4) // 1-4 items
		var items []InvoiceItem
		for j := 0; j < numItems; j++ {
			prodIdx := r.Intn(len(productIDs))
			prod, err := s.GetProduct(productIDs[prodIdx])
			if err != nil {
				return err
			}
			prodID := productIDs[prodIdx]
			qty := 1 + r.Intn(5)
			items = append(items, InvoiceItem{
				Description:  prod.Name,
				Quantity:     qty,
				PricePerUnit: prod.Price,
				ProductID:    &prodID,
			})
		}

		inv := &Invoice{
			InvoiceNumber:    invoiceNum,
			Date:             date,
			SenderName:       senderName,
			SenderAddress:    senderAddr,
			RecipientName:    cust.Name,
			RecipientAddress: cust.Address,
			TaxRate:          19.0,
			Status:           status,
			CustomerID:       &custID,
			Items:            items,
			InternalNote:     fmt.Sprintf("Demo-Rechnung erstellt am %s. Kundenzufriedenheit: Hoch.", now.Format("02.01.2006")),
			DocumentNote:     "Vielen Dank für die angenehme Zusammenarbeit! Bitte begleichen Sie den Betrag fristgerecht.",
		}

		if _, err := s.CreateInvoice(inv); err != nil {
			return fmt.Errorf("Rechnung %d: %w", i+1, err)
		}
	}

	// Update next invoice number
	return s.SetSetting("next_invoice_number", "150")
}

func (s *Store) seedQuotes(customerIDs, productIDs []int) error {
	r := rand.New(rand.NewSource(99))
	senderName := "Demo GmbH"
	senderAddr := "Musterstraße 1\n12345 Berlin\nDeutschland"

	statuses := []string{"Entwurf", "Verschickt", "Angenommen", "Abgelehnt", "Verschickt"}
	now := time.Now()

	for i := 0; i < 15; i++ {
		custIdx := r.Intn(len(customerIDs))
		custID := customerIDs[custIdx]
		cust, err := s.GetCustomer(custID)
		if err != nil {
			return err
		}

		daysAgo := r.Intn(180)
		date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		status := statuses[r.Intn(len(statuses))]
		quoteNum := fmt.Sprintf("AN-%04d", i+1)

		numItems := 1 + r.Intn(3)
		var items []QuoteItem
		for j := 0; j < numItems; j++ {
			prodIdx := r.Intn(len(productIDs))
			prod, err := s.GetProduct(productIDs[prodIdx])
			if err != nil {
				return err
			}
			prodID := productIDs[prodIdx]
			qty := 1 + r.Intn(10)
			items = append(items, QuoteItem{
				Description:  prod.Name,
				Quantity:     qty,
				PricePerUnit: prod.Price,
				ProductID:    &prodID,
			})
		}

		q := &Quote{
			QuoteNumber:      quoteNum,
			Date:             date,
			SenderName:       senderName,
			SenderAddress:    senderAddr,
			RecipientName:    cust.Name,
			RecipientAddress: cust.Address,
			TaxRate:          19.0,
			Status:           status,
			CustomerID:       &custID,
			Items:            items,
			InternalNote:     "Nachfass-Aktion für dieses Angebot geplant für nächste Woche.",
			DocumentNote:     "Dieses Angebot ist freibleibend. Wir freuen uns auf Ihre Rückmeldung.",
		}

		if _, err := s.CreateQuote(q); err != nil {
			return fmt.Errorf("Angebot %d: %w", i+1, err)
		}
	}
	return nil
}

func (s *Store) seedCreditNotes(customerIDs, productIDs []int) error {
	r := rand.New(rand.NewSource(77))
	senderName := "Demo GmbH"
	senderAddr := "Musterstraße 1\n12345 Berlin\nDeutschland"

	statuses := []string{"Entwurf", "Offen", "Abgeschlossen"}
	now := time.Now()

	for i := 0; i < 5; i++ {
		custIdx := r.Intn(len(customerIDs))
		custID := customerIDs[custIdx]
		cust, err := s.GetCustomer(custID)
		if err != nil {
			return err
		}

		daysAgo := r.Intn(120)
		date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		status := statuses[r.Intn(len(statuses))]
		cnNum := fmt.Sprintf("GS-%04d", i+1)

		prodIdx := r.Intn(len(productIDs))
		prod, err := s.GetProduct(productIDs[prodIdx])
		if err != nil {
			return err
		}
		prodID := productIDs[prodIdx]

		cn := &CreditNote{
			CreditNoteNumber: cnNum,
			Date:             date,
			SenderName:       senderName,
			SenderAddress:    senderAddr,
			RecipientName:    cust.Name,
			RecipientAddress: cust.Address,
			TaxRate:          19.0,
			Status:           status,
			CustomerID:       &custID,
			InternalNote:     "Gutschrift aufgrund von Reklamation (Transportschaden).",
			DocumentNote:     "Hiermit schreiben wir Ihnen den Betrag für die beschädigte Ware gut.",
			Items: []CreditNoteItem{
				{
					Description:  prod.Name + " (Gutschrift)",
					Quantity:     1 + r.Intn(3),
					PricePerUnit: prod.Price,
					ProductID:    &prodID,
				},
			},
		}

		if _, err := s.CreateCreditNote(cn); err != nil {
			return fmt.Errorf("Gutschrift %d: %w", i+1, err)
		}
	}
	return nil
}

func (s *Store) seedExpenses(categoryIDs []int) error {
	r := rand.New(rand.NewSource(55))
	now := time.Now()

	expenses := []struct {
		desc    string
		amount  float64
		taxRate float64
		catIdx  int
	}{
		{"Druckerpapier A4 (5 Karton)", 45.90, 19.0, 0},
		{"Toner schwarz HP LaserJet", 89.99, 19.0, 0},
		{"Kugelschreiber (100 Stk)", 24.50, 19.0, 0},
		{"Adobe Creative Cloud (Jahresabo)", 713.86, 19.0, 1},
		{"Microsoft 365 Business", 264.00, 19.0, 1},
		{"JetBrains IntelliJ IDEA", 499.00, 19.0, 1},
		{"Slack Business (Jahresabo)", 336.00, 19.0, 1},
		{"Büromiete Januar", 1200.00, 19.0, 2},
		{"Büromiete Februar", 1200.00, 19.0, 2},
		{"Büromiete März", 1200.00, 19.0, 2},
		{"Nebenkostenabrechnung 2025", 480.00, 19.0, 2},
		{"Betriebshaftpflicht (Jahr)", 890.00, 19.0, 3},
		{"Berufshaftpflicht IT", 650.00, 19.0, 3},
		{"Bahnfahrt München (Kundentermin)", 129.00, 7.0, 4},
		{"Hotel München (2 Nächte)", 238.00, 7.0, 4},
		{"Flug Hamburg (Konferenz)", 189.00, 19.0, 4},
		{"Taxi zum Flughafen", 45.00, 7.0, 4},
		{"Vodafone Business Tarif", 49.99, 19.0, 5},
		{"Telekom Festnetz + Internet", 59.95, 19.0, 5},
		{"AWS-Hosting (Quartal)", 342.00, 19.0, 5},
		{"Go-Konferenz Ticket", 499.00, 19.0, 6},
		{"Online-Kurs Kubernetes", 129.00, 19.0, 6},
		{"Fachbuch Clean Architecture", 39.99, 7.0, 6},
		{"Tankfüllung Firmenwagen", 85.00, 19.0, 7},
		{"KFZ-Versicherung (Quartal)", 220.00, 19.0, 7},
		{"TÜV Hauptuntersuchung", 119.00, 19.0, 7},
		{"Google Ads Kampagne Jan", 500.00, 19.0, 8},
		{"Google Ads Kampagne Feb", 500.00, 19.0, 8},
		{"Messestand-Miete IT-Expo", 2400.00, 19.0, 8},
		{"Geschäftsessen mit Kunde Fischer", 156.80, 19.0, 9},
		{"Teamessen Jahresabschluss", 289.50, 19.0, 9},
		{"Kaffee und Getränke (Büro)", 67.40, 7.0, 9},
	}

	for _, e := range expenses {
		daysAgo := r.Intn(365)
		date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		catID := categoryIDs[e.catIdx]

		_, err := s.CreateExpense(Expense{
			Description: e.desc,
			Amount:      e.amount,
			TaxRate:     e.taxRate,
			Date:        date,
			CategoryID:  &catID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seedRecurringExpenses(categoryIDs []int) error {
	recurring := []struct {
		desc     string
		amount   float64
		taxRate  float64
		interval string
		catIdx   int
	}{
		{"Büromiete", 1200.00, 19.0, "monthly", 2},
		{"Vodafone Business", 49.99, 19.0, "monthly", 5},
		{"Telekom Internet", 59.95, 19.0, "monthly", 5},
		{"Adobe Creative Cloud", 59.49, 19.0, "monthly", 1},
		{"Betriebshaftpflicht", 890.00, 19.0, "yearly", 3},
		{"KFZ-Versicherung", 220.00, 19.0, "quarterly", 7},
	}

	for _, re := range recurring {
		catID := categoryIDs[re.catIdx]
		_, err := s.CreateRecurringExpense(RecurringExpense{
			Description: re.desc,
			Amount:      re.amount,
			TaxRate:     re.taxRate,
			Interval:    re.interval,
			CategoryID:  &catID,
			StartDate:   "2025-01-01",
			IsActive:    true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seedBigInvoice(customerIDs, productIDs []int) error {
	senderName := "Demo GmbH"
	senderAddr := "Musterstraße 1\n12345 Berlin\nDeutschland"
	now := time.Now()

	custID := customerIDs[0] // Müller & Söhne
	cust, _ := s.GetCustomer(custID)

	var items []InvoiceItem
	// Create 20 items
	for i := 0; i < 20; i++ {
		prodIdx := i % len(productIDs)
		prod, _ := s.GetProduct(productIDs[prodIdx])
		prodID := prod.ID
		items = append(items, InvoiceItem{
			Description:  fmt.Sprintf("%s (Pos %d)", prod.Name, i+1),
			Quantity:     1 + (i % 3),
			PricePerUnit: prod.Price,
			ProductID:    &prodID,
		})
	}

	inv := &Invoice{
		InvoiceNumber:    "RE-BIG-2026",
		Date:             now.Format("2006-01-02"),
		SenderName:       senderName,
		SenderAddress:    senderAddr,
		RecipientName:    cust.Name,
		RecipientAddress: cust.Address,
		TaxRate:          19.0,
		Status:           "Offen",
		CustomerID:       &custID,
		Items:            items,
		InternalNote:     "Diese Rechnung dient zum Testen von mehrseitigen PDFs und dem Layout bei vielen Positionen.",
		DocumentNote:     "Vielen Dank für Ihre umfangreiche Bestellung! Wir haben Ihnen einen Sonderrabatt auf alle Positionen gewährt.",
	}

	_, err := s.CreateInvoice(inv)
	return err
}
