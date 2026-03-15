package services

import (
	"context"
	"din-invoice/models"
	"din-invoice/views"
	"encoding/base64"
	"fmt"
	"io"
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
	u := launcher.New().NoSandbox(true).Leakless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	if err := page.SetDocumentContent(htmlContent); err != nil {
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

	pdfStream, err := page.PDF(opts)
	if err != nil {
		return fmt.Errorf("failed to generate pdf: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, pdfStream)
	return err
}

// GenerateEuerPDFHTML renders the EÜR overview as a PDF.
func GenerateEuerPDFHTML(stats *models.EuerStats, settings *models.AppSettings) (string, error) {
	htmlComponent := views.EuerPDF(stats, *settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	filename := filepath.Join(outDir, "euer_uebersicht.pdf")

	footer := pageFooterHTML(settings.SenderName)
	if err := renderHTMLToPDFWithFooter(htmlBuilder.String(), filename, footer); err != nil {
		return "", err
	}
	return filename, nil
}

// GenerateInventoryPDFHTML renders the inventory list as a PDF.
func GenerateInventoryPDFHTML(products []models.Product, settings *models.AppSettings) (string, error) {
	htmlComponent := views.InventarPDF(products, *settings)

	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
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

// GenerateInvoicePDFHTML renders the HTML view and converts it to PDF using Rod.
func GenerateInvoicePDFHTML(inv *models.Invoice, settings *models.AppSettings) (string, error) {
	// 1. Prepare Logo (Base64)
	// Relative file URLs are tricky with rod.SetDocumentContent. Base64 is safer.
	if settings.LogoPath != "" {
		// Clean the path (remove file:// prefix if we added it previously, though now we prefer Base64)
		cleanPath := strings.TrimPrefix(settings.LogoPath, "file://")

		// If it's a relative path, resolve it relative to CWD
		if !filepath.IsAbs(cleanPath) {
			abs, err := filepath.Abs(cleanPath)
			if err == nil {
				cleanPath = abs
			}
		}

		data, err := os.ReadFile(cleanPath)
		if err == nil {
			// Detect MIME type
			mimeType := http.DetectContentType(data)
			base64Data := base64.StdEncoding.EncodeToString(data)
			// Update settings object (passed by value mostly, but it's a pointer here? No, *AppSettings)
			// We modify the copy or the actual settings struct for this render only.
			// Ideally we don't mutate global state, but here we modify the struct passed to the view.
			settings.LogoPath = fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
		} else {
			// Log error? or ignore
			fmt.Printf("Failed to read logo file for PDF: %v\n", err)
		}
	}

	htmlComponent := views.InvoicePDF(inv, settings)

	// Create a buffer to render into
	var htmlBuilder strings.Builder
	if err := htmlComponent.Render(context.Background(), &htmlBuilder); err != nil {
		return "", fmt.Errorf("failed to render html: %w", err)
	}

	htmlContent := htmlBuilder.String()

	// 2. Setup Rod (Browser)
	// Use NoSandbox for container environments.
	// Disable Leakless to avoid anti-virus false positives on Windows.
	u := launcher.New().NoSandbox(true).Leakless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()

	// 3. Load Content
	// We use SetDocumentContent to load HTML directly
	if err := page.SetDocumentContent(htmlContent); err != nil {
		return "", fmt.Errorf("failed to set page content: %w", err)
	}

	page.MustWaitLoad()

	// 4. Generate PDF
	// A4 measurements
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
		return "", fmt.Errorf("failed to generate pdf: %w", err)
	}

	// 5. Save to File
	outDir := settings.PDFOutputPath
	if outDir == "" {
		outDir = "./invoices/"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	// Create file
	filename := filepath.Join(outDir, fmt.Sprintf("rechnung_%s.pdf", inv.InvoiceNumber))
	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Copy stream to file
	_, err = io.Copy(f, pdfStream)
	if err != nil {
		return "", err
	}

	return filename, nil
}
