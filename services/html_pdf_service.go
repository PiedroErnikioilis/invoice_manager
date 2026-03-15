package services

import (
	"context"
	"din-invoice/models"
	"din-invoice/views"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func toPtr(f float64) *float64 { return &f }

// pageFooterHTML returns a Chrome footer template with page numbers.
// leftText is shown on the left side, page numbers on the right.
func pageFooterHTML(leftText string) string {
	return fmt.Sprintf(`<div style="font-size:8pt; font-family:Calibri,Arial,sans-serif; color:#999; width:100%%; padding:0 20mm; display:flex; justify-content:space-between;">
		<span>%s</span>
		<span>Seite <span class="pageNumber"></span> von <span class="totalPages"></span></span>
	</div>`, leftText)
}

// renderHTMLToPDF takes an HTML string and produces a PDF file at the given path.
func renderHTMLToPDF(htmlContent, outputPath string) error {
	return renderHTMLToPDFWithFooter(htmlContent, outputPath, "")
}

// renderHTMLToPDFWithFooter renders HTML to PDF with an optional footer template.
// The footerHTML supports Chrome's special classes: pageNumber, totalPages.
func renderHTMLToPDFWithFooter(htmlContent, outputPath, footerHTML string) error {
	slog.Debug("Launching browser for PDF generation", "output", outputPath)
	
	u := launcher.New().NoSandbox(true).Leakless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	if err := page.SetDocumentContent(htmlContent); err != nil {
		slog.Error("Failed to set page content", "error", err)
		return fmt.Errorf("failed to set page content: %w", err)
	}
	page.MustWaitLoad()

	opts := &proto.PagePrintToPDF{
		PaperWidth:      toPtr(8.27),
		PaperHeight:     toPtr(11.69),
		MarginTop:       toPtr(0.0),
		MarginBottom:    toPtr(0.0),
		MarginLeft:      toPtr(0.0),
		MarginRight:     toPtr(0.0),
		PrintBackground: true,
	}

	if footerHTML != "" {
		opts.DisplayHeaderFooter = true
		opts.HeaderTemplate = "<span></span>"
		opts.FooterTemplate = footerHTML
		opts.MarginBottom = toPtr(0.5) // ~12mm space for footer
	}

	slog.Debug("Printing page to PDF", "output", outputPath)
	pdfStream, err := page.PDF(opts)
	if err != nil {
		slog.Error("Failed to generate PDF stream", "error", err)
		return fmt.Errorf("failed to generate pdf: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		slog.Error("Failed to create PDF output directory", "path", filepath.Dir(outputPath), "error", err)
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		slog.Error("Failed to create PDF file", "path", outputPath, "error", err)
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, pdfStream)
	if err != nil {
		slog.Error("Failed to write PDF file", "path", outputPath, "error", err)
		return err
	}

	slog.Info("PDF generated successfully", "path", outputPath, "size_bytes", n)
	return nil
}

// GenerateEuerPDFHTML renders the EÜR overview as a PDF.
func GenerateEuerPDFHTML(stats *models.EuerStats, settings *models.AppSettings) (string, error) {
	slog.Info("Generating EÜR PDF")
	htmlComponent := views.EuerPDF(stats, *settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		slog.Error("Failed to render EÜR HTML", "error", err)
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	filename := filepath.Join(outDir, models.FormatDocumentNumber(settings.EuerFilenameSchema, 0)+".pdf")

	footer := pageFooterHTML(settings.SenderName)
	if err := renderHTMLToPDFWithFooter(htmlBuilder.String(), filename, footer); err != nil {
		return "", err
	}
	return filename, nil
}

// GenerateInventoryPDFHTML renders the inventory list as a PDF.
func GenerateInventoryPDFHTML(products []models.Product, settings *models.AppSettings) (string, error) {
	slog.Info("Generating Inventory PDF")
	htmlComponent := views.InventarPDF(products, *settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		slog.Error("Failed to render inventory HTML", "error", err)
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	filename := filepath.Join(outDir, "inventar_liste.pdf")

	footer := pageFooterHTML(settings.SenderName)
	if err := renderHTMLToPDFWithFooter(htmlBuilder.String(), filename, footer); err != nil {
		return "", err
	}
	return filename, nil
}

// GenerateCreditNotePDFHTML renders the credit note view and converts it to PDF using Rod.
func GenerateCreditNotePDFHTML(note *models.CreditNote, settings *models.AppSettings) (string, error) {
	slog.Info("Generating Credit Note PDF", "credit_note_number", note.CreditNoteNumber)
	
	// Prepare Logo (Base64)
	if settings.LogoPath != "" {
		cleanPath := strings.TrimPrefix(settings.LogoPath, "file://")
		if !filepath.IsAbs(cleanPath) {
			abs, err := filepath.Abs(cleanPath)
			if err == nil {
				cleanPath = abs
			}
		}

		data, err := os.ReadFile(cleanPath)
		if err == nil {
			mimeType := http.DetectContentType(data)
			base64Data := base64.StdEncoding.EncodeToString(data)
			settings.LogoPath = fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
			slog.Debug("Logo embedded as base64 for credit note", "path", cleanPath)
		} else {
			slog.Error("Failed to read logo file for credit note PDF", "path", cleanPath, "error", err)
		}
	}

	htmlComponent := views.CreditNotePDF(note, settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		slog.Error("Failed to render credit note HTML", "error", err)
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	htmlContent := htmlBuilder.String()

	// Setup Rod
	u := launcher.New().NoSandbox(true).Leakless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	if err := page.SetDocumentContent(htmlContent); err != nil {
		slog.Error("Failed to set credit note page content", "credit_note_number", note.CreditNoteNumber, "error", err)
		return "", fmt.Errorf("failed to set page content: %w", err)
	}
	page.MustWaitLoad()

	// Generate PDF
	footer := `<div style="font-size:7pt; font-family:Calibri,Arial,sans-serif; color:#aaa; width:100%; text-align:center;">
		<span>Seite <span class="pageNumber"></span> von <span class="totalPages"></span></span>
	</div>`
	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:           toPtr(8.27),
		PaperHeight:          toPtr(11.69),
		MarginTop:            toPtr(0.0),
		MarginBottom:         toPtr(0.35),
		MarginLeft:           toPtr(0.0),
		MarginRight:          toPtr(0.0),
		PrintBackground:      true,
		DisplayHeaderFooter:  true,
		HeaderTemplate:       "<span></span>",
		FooterTemplate:       footer,
	})
	if err != nil {
		slog.Error("Failed to generate credit note PDF stream", "credit_note_number", note.CreditNoteNumber, "error", err)
		return "", fmt.Errorf("failed to generate pdf: %w", err)
	}

	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		slog.Error("Failed to create PDF output directory", "path", outDir, "error", err)
		return "", err
	}

	filename := filepath.Join(outDir, fmt.Sprintf("gutschrift_%s.pdf", note.CreditNoteNumber))
	f, err := os.Create(filename)
	if err != nil {
		slog.Error("Failed to create credit note PDF file", "path", filename, "error", err)
		return "", err
	}
	defer f.Close()

	n, err := io.Copy(f, pdfStream)
	if err != nil {
		slog.Error("Failed to write credit note PDF file", "path", filename, "error", err)
		return "", err
	}

	slog.Info("Credit Note PDF generated successfully", "path", filename, "size_bytes", n)
	return filename, nil
}

// GenerateInvoicePDFHTML renders the HTML view and converts it to PDF using Rod.
func GenerateInvoicePDFHTML(inv *models.Invoice, settings *models.AppSettings) (string, error) {
	slog.Info("Generating Invoice PDF", "invoice_number", inv.InvoiceNumber)
	
	// 1. Prepare Logo (Base64)
	if settings.LogoPath != "" {
		cleanPath := strings.TrimPrefix(settings.LogoPath, "file://")
		if !filepath.IsAbs(cleanPath) {
			abs, err := filepath.Abs(cleanPath)
			if err == nil {
				cleanPath = abs
			}
		}

		data, err := os.ReadFile(cleanPath)
		if err == nil {
			mimeType := http.DetectContentType(data)
			base64Data := base64.StdEncoding.EncodeToString(data)
			settings.LogoPath = fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
			slog.Debug("Logo embedded as base64", "path", cleanPath)
		} else {
			slog.Error("Failed to read logo file for PDF", "path", cleanPath, "error", err)
		}
	}

	htmlComponent := views.InvoicePDF(inv, settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		slog.Error("Failed to render invoice HTML", "error", err)
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	htmlContent := htmlBuilder.String()

	// 2. Setup Rod (Browser)
	slog.Debug("Launching browser for invoice PDF", "invoice_number", inv.InvoiceNumber)
	u := launcher.New().NoSandbox(true).Leakless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()

	if err := page.SetDocumentContent(htmlContent); err != nil {
		slog.Error("Failed to set invoice page content", "invoice_number", inv.InvoiceNumber, "error", err)
		return "", fmt.Errorf("failed to set page content: %w", err)
	}

	page.MustWaitLoad()

	// 4. Generate PDF
	invoiceFooter := `<div style="font-size:7pt; font-family:Calibri,Arial,sans-serif; color:#aaa; width:100%; text-align:center;">
		<span>Seite <span class="pageNumber"></span> von <span class="totalPages"></span></span>
	</div>`
	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:           toPtr(8.27),  // A4 Width in inches
		PaperHeight:          toPtr(11.69), // A4 Height
		MarginTop:            toPtr(0.0),
		MarginBottom:         toPtr(0.35),  // ~9mm for page number
		MarginLeft:           toPtr(0.0),
		MarginRight:          toPtr(0.0),
		PrintBackground:      true,
		DisplayHeaderFooter:  true,
		HeaderTemplate:       "<span></span>",
		FooterTemplate:       invoiceFooter,
	})
	if err != nil {
		slog.Error("Failed to generate invoice PDF stream", "invoice_number", inv.InvoiceNumber, "error", err)
		return "", fmt.Errorf("failed to generate pdf: %w", err)
	}

	// 5. Save to File
	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		slog.Error("Failed to create PDF output directory", "path", outDir, "error", err)
		return "", err
	}

	filename := filepath.Join(outDir, fmt.Sprintf("rechnung_%s.pdf", inv.InvoiceNumber))
	f, err := os.Create(filename)
	if err != nil {
		slog.Error("Failed to create invoice PDF file", "path", filename, "error", err)
		return "", err
	}
	defer f.Close()

	n, err := io.Copy(f, pdfStream)
	if err != nil {
		slog.Error("Failed to write invoice PDF file", "path", filename, "error", err)
		return "", err
	}

	slog.Info("Invoice PDF generated successfully", "path", filename, "size_bytes", n)
	return filename, nil
}
