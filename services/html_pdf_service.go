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
	// rod.StreamReader implements io.Reader
	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:      toPtr(8.27),  // A4 Width in inches
		PaperHeight:     toPtr(11.69), // A4 Height
		MarginTop:       toPtr(0.0),
		MarginBottom:    toPtr(0.0),
		MarginLeft:      toPtr(0.0),
		MarginRight:     toPtr(0.0), // We handle margins in CSS
		PrintBackground: true,       // Important for CSS backgrounds
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
