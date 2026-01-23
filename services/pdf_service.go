package services

import (
	"din-invoice/models"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/johnfercher/maroto/pkg/color"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
)

// DIN 5008 Form B measurements (approximate for Maroto grid system)
// Top margin: 0 (we handle it manually with spacer)
// Address field top: 45mm
// Fold marks: 87mm, 192mm
// Punch mark: 148.5mm

func GenerateInvoicePDF(inv *models.Invoice, settings *models.AppSettings) (string, error) {
	m := pdf.NewMaroto(consts.Portrait, consts.A4)

	// Set very small margins to control positioning precisely with spacers
	m.SetPageMargins(10, 0, 10) // Left/Right 20mm typically, but Maroto adds its own.
	// Actually, DIN 5008 left margin is 25mm.
	m.SetPageMargins(20, 0, 10)

	// --- Fold Marks & Punch Mark (Background) ---
	// Maroto doesn't easily support absolute background layers, but we can try to draw lines at specific "Rows".
	// Alternatively, we just assume standard vertical flow.
	// 87mm from top.

	// We will use a header to draw the fold marks "absolutely" if possible, or just ignore for now as Maroto v1 is limited.
	// Actually, let's try to be precise with Rows.

	// 1. Top Spacer to reach 45mm for Address Field
	// But first, we might want a Header/Logo area.
	// DIN 5008 Form B allows header up to 45mm.

	// Return Address (Sender) - 5mm height, starts at 45mm (bottom aligned?)
	// No, Return address is usually part of the address window (Zone: 40mm high, top 5mm is return address).
	// So "Address Field" starts at 45mm.

	// Let's create a spacer for the Header Area (0mm - 45mm)
	m.Row(45, func() {
		if settings.LogoPath != "" {
			m.Col(12, func() {
				_ = m.FileImage(settings.LogoPath, props.Rect{
					Left:    0,
					Top:     5,
					Percent: 50,    // Adjust sizing?
					Center:  false, // Right align? Usually logo is top right or top left.
				})
			})
		}
	})

	// 2. Address Field (45mm - 85mm)
	// Height 40mm.
	// Width 85mm. Left margin 20mm (from page edge).
	// Maroto left margin is set to 20.

	m.Row(40, func() {
		m.Col(5, func() { // Approx 85mm wide column (A4 width 210 - 20 - 10 = 180. 5/12 * 180 = 75mm. A bit narrow.)
			// Let's use 6/12 = 90mm.
			// Return Address Line (Tiny)
			m.Text(fmt.Sprintf("%s • %s", inv.SenderName, inv.SenderAddress), props.Text{
				Size:  6,
				Top:   2, // Inside the field
				Color: color.Color{Red: 100, Green: 100, Blue: 100},
				Style: consts.Italic,
			})

			// Recipient Address
			m.Text(inv.RecipientName, props.Text{
				Size: 10,
				Top:  10,
			})
			m.Text(inv.RecipientAddress, props.Text{
				Size: 10,
				Top:  15,
			})
		})

		// Info Block (Right Side)
		// Starts usually at 50mm vertical? We are in the 45-85mm row.
		m.Col(6, func() {
			m.Text("RECHNUNG", props.Text{
				Size:  14,
				Style: consts.Bold,
				Align: consts.Right,
				Top:   0,
			})

			data := [][]string{
				{"Rechnungs-Nr:", inv.InvoiceNumber},
				{"Datum:", inv.Date},
				{"Kunden-Nr:", fmt.Sprintf("KD-%d", inv.ID+1000)},
			}

			curTop := 10.0
			for _, row := range data {
				m.Text(row[0], props.Text{
					Size:  9,
					Align: consts.Right,
					Right: 30,
					Top:   curTop,
					Style: consts.Bold,
				})
				m.Text(row[1], props.Text{
					Size:  9,
					Align: consts.Right,
					Top:   curTop,
				})
				curTop += 4.5
			}
		})
	})

	// 3. Spacer to reach content start (approx 98mm or 105mm)
	// We are at 45+40 = 85mm.
	// Fold mark 1 is at 87mm.

	m.Row(15, func() {
		// We can draw the fold mark here?
		// It's tricky with Maroto v1 to draw absolute lines.
	})

	// 4. Subject Line
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text(fmt.Sprintf("Rechnung Nr. %s", inv.InvoiceNumber), props.Text{
				Size:  12,
				Style: consts.Bold,
			})
		})
	})

	// 5. Intro Text
	m.Row(15, func() {
		m.Col(12, func() {
			m.Text("Sehr geehrte Damen und Herren,", props.Text{Size: 10, Top: 0})
			m.Text("vielen Dank für Ihren Auftrag. Wir stellen Ihnen folgende Leistungen in Rechnung:", props.Text{Size: 10, Top: 5})
		})
	})

	m.Row(5, func() {})

	// --- Items Table ---
	headers := []string{"Pos", "Beschreibung", "Menge", "Preis", "Gesamt"}
	contents := [][]string{}

	for i, item := range inv.Items {
		contents = append(contents, []string{
			strconv.Itoa(i + 1),
			item.Description,
			strconv.Itoa(item.Quantity),
			fmt.Sprintf("%.2f €", item.PricePerUnit),
			fmt.Sprintf("%.2f €", float64(item.Quantity)*item.PricePerUnit),
		})
	}

	m.TableList(headers, contents, props.TableList{
		HeaderProp: props.TableListContent{
			Size:      9,
			Style:     consts.Bold,
			GridSizes: []uint{1, 6, 2, 2, 1},
			Color:     color.Color{Red: 240, Green: 240, Blue: 240},
		},
		ContentProp: props.TableListContent{
			Size:      9,
			GridSizes: []uint{1, 6, 2, 2, 1},
		},
		Align:              consts.Left,
		HeaderContentSpace: 2,
	})

	m.Row(2, func() {})

	// --- Totals ---
	m.Row(5, func() {
		m.Col(9, func() {})
		m.Col(2, func() {
			m.Text("Nettobetrag:", props.Text{Align: consts.Right, Style: consts.Bold, Size: 9})
		})
		m.Col(1, func() {
			m.Text(fmt.Sprintf("%.2f €", inv.TotalNet()), props.Text{Align: consts.Right, Size: 9, Family: consts.Courier})
		})
	})

	if !inv.IsSmallBusiness {
		m.Row(5, func() {
			m.Col(9, func() {})
			m.Col(2, func() {
				m.Text(fmt.Sprintf("MwSt (%.0f%%):", inv.TaxRate), props.Text{Align: consts.Right, Style: consts.Bold, Size: 9})
			})
			m.Col(1, func() {
				m.Text(fmt.Sprintf("%.2f €", inv.TaxAmount()), props.Text{Align: consts.Right, Size: 9, Family: consts.Courier})
			})
		})
	}

	m.Line(1.0, props.Line{Color: color.Color{Red: 200, Green: 200, Blue: 200}})

	m.Row(10, func() {
		m.Col(9, func() {})
		m.Col(2, func() {
			m.Text("Gesamtbetrag:", props.Text{Align: consts.Right, Style: consts.Bold, Size: 11, Top: 2})
		})
		m.Col(1, func() {
			m.Text(fmt.Sprintf("%.2f €", inv.TotalGross()), props.Text{Align: consts.Right, Style: consts.Bold, Size: 11, Top: 2, Family: consts.Courier})
		})
	})

	// --- Footer Notes ---
	m.Row(20, func() {
		m.Col(12, func() {
			top := 5.0
			if inv.IsSmallBusiness {
				m.Text("Gemäß § 19 UStG wird keine Umsatzsteuer berechnet.", props.Text{Size: 8, Top: top, Style: consts.Italic})
				top += 10
			} else {
				top += 5
			}

			m.Text("Bitte überweisen Sie den Betrag innerhalb von 14 Tagen ohne Abzug.", props.Text{Size: 9, Top: top})
		})
	})

	// --- Footer Columns ---
	m.RegisterFooter(func() {
		// Draw Fold Marks (Absolute hack using footer top negative?)
		// Maroto v1 doesn't allow easy absolute drawing outside row context.
		// We will just draw the footer content properly.

		m.Line(1.0, props.Line{Color: color.Color{Red: 200, Green: 200, Blue: 200}})
		m.Row(15, func() {
			m.Col(4, func() {
				m.Text(inv.SenderName, props.Text{Style: consts.Bold, Size: 8, Top: 2})
				m.Text(inv.SenderAddress, props.Text{Size: 7, Top: 6})
			})
			m.Col(4, func() {
				m.Text("Bankverbindung", props.Text{Style: consts.Bold, Size: 8, Top: 2})
				m.Text(settings.BankName, props.Text{Size: 7, Top: 6})
				m.Text(fmt.Sprintf("IBAN: %s", settings.IBAN), props.Text{Size: 7, Top: 9})
				m.Text(fmt.Sprintf("BIC: %s", settings.BIC), props.Text{Size: 7, Top: 12})
			})
			m.Col(4, func() {
				m.Text("Kontakt", props.Text{Style: consts.Bold, Size: 8, Top: 2})
				m.Text(settings.Website, props.Text{Size: 7, Top: 6})
				m.Text(settings.Email, props.Text{Size: 7, Top: 9})
			})
		})
	})

	// Output Path
	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	filename := filepath.Join(outDir, fmt.Sprintf("rechnung_%s.pdf", inv.InvoiceNumber))
	err := m.OutputFileAndClose(filename)
	return filename, err
}
