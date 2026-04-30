// Package pdf implements service.PDFRenderer via headless Chrome
// (chromedp). The four html/template files are embedded at compile
// time so a binary built without the source tree still renders.
//
// Why headless Chrome rather than a Go-native PDF lib (gofpdf,
// jung-kurt/gofpdf, signintech/gopdf): consistent CSS support, easy
// updates to the template, and a future-friendly path to e-signature /
// QR codes via standard HTML.
//
// The renderer is safe for concurrent use — each Render spawns its own
// chromedp context so multiple goroutines never share Chrome state.
package pdf

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"marketplace-backend/internal/domain/invoicing"
)

//go:embed templates/*.html.tmpl
var templatesFS embed.FS

// renderTimeout caps the total chromedp run. PDF rendering of a small
// invoice usually finishes in <2s; anything past 30s is a stuck Chrome
// instance and we fail hard rather than block the request.
const renderTimeout = 30 * time.Second

// Renderer is the PDF adapter. Templates are parsed once at New() and
// reused for every render — html/template Templates are safe for
// concurrent execution.
type Renderer struct {
	invoiceFR    *template.Template
	invoiceEN    *template.Template
	creditNoteFR *template.Template
	creditNoteEN *template.Template
}

// New parses every embedded template upfront. Failing here means the
// embed went wrong (missing template, syntax error) — log loudly and
// abort startup rather than discover the failure on the first invoice.
func New() (*Renderer, error) {
	funcs := template.FuncMap{
		"upper": strings.ToUpper,
	}

	parse := func(name string) (*template.Template, error) {
		raw, err := templatesFS.ReadFile("templates/" + name)
		if err != nil {
			return nil, fmt.Errorf("pdf: read template %s: %w", name, err)
		}
		t, err := template.New(name).Funcs(funcs).Parse(string(raw))
		if err != nil {
			return nil, fmt.Errorf("pdf: parse template %s: %w", name, err)
		}
		return t, nil
	}

	invFR, err := parse("invoice.fr.html.tmpl")
	if err != nil {
		return nil, err
	}
	invEN, err := parse("invoice.en.html.tmpl")
	if err != nil {
		return nil, err
	}
	cnFR, err := parse("credit_note.fr.html.tmpl")
	if err != nil {
		return nil, err
	}
	cnEN, err := parse("credit_note.en.html.tmpl")
	if err != nil {
		return nil, err
	}

	return &Renderer{
		invoiceFR:    invFR,
		invoiceEN:    invEN,
		creditNoteFR: cnFR,
		creditNoteEN: cnEN,
	}, nil
}

// invoiceView is the view-model passed to invoice templates. Cents are
// pre-formatted into display strings here so the template stays purely
// presentational.
type invoiceView struct {
	Number             string
	IssuedAt           string
	ServicePeriodStart string
	ServicePeriodEnd   string
	Issuer             invoicing.IssuerInfo
	Recipient          invoicing.RecipientInfo
	Items              []itemView
	AmountExcl         string
	VATAmount          string
	VATRate            string
	AmountIncl         string
	Mentions           []string
}

// creditNoteView is the avoir view-model. Fewer line-items than an
// invoice — the credit note has no per-item table in V1, only totals.
type creditNoteView struct {
	Number                string
	IssuedAt              string
	OriginalInvoiceNumber string
	Reason                string
	Issuer                invoicing.IssuerInfo
	Recipient             invoicing.RecipientInfo
	AmountExcl            string
	VATAmount             string
	VATRate               string
	AmountIncl            string
	Mentions              []string
}

// itemView formats one invoice line for display.
type itemView struct {
	Description string
	Quantity    string
	UnitPrice   string
	Amount      string
}

// pickInvoiceTemplate returns the right invoice template for the
// language. Anything other than "fr" falls back to English to satisfy
// the port contract that language drift never blocks invoicing.
func (r *Renderer) pickInvoiceTemplate(language string) *template.Template {
	if strings.EqualFold(strings.TrimSpace(language), "fr") {
		return r.invoiceFR
	}
	return r.invoiceEN
}

// pickCreditNoteTemplate is the avoir twin of pickInvoiceTemplate.
func (r *Renderer) pickCreditNoteTemplate(language string) *template.Template {
	if strings.EqualFold(strings.TrimSpace(language), "fr") {
		return r.creditNoteFR
	}
	return r.creditNoteEN
}

// RenderInvoice produces the PDF bytes for a finalized invoice.
func (r *Renderer) RenderInvoice(ctx context.Context, inv *invoicing.Invoice, language string) ([]byte, error) {
	if inv == nil {
		return nil, fmt.Errorf("pdf: invoice is nil")
	}
	view := buildInvoiceView(inv, language)

	var buf bytes.Buffer
	tmpl := r.pickInvoiceTemplate(language)
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("pdf: execute invoice template: %w", err)
	}

	return r.htmlToPDF(ctx, buf.String())
}

// RenderCreditNote produces the PDF bytes for a finalized credit note.
// The credit note doesn't carry the original invoice number on its
// own — the caller passes the parent invoice's number to display via
// the back-reference banner.
func (r *Renderer) RenderCreditNote(ctx context.Context, cn *invoicing.CreditNote, language string) ([]byte, error) {
	if cn == nil {
		return nil, fmt.Errorf("pdf: credit note is nil")
	}
	view := buildCreditNoteView(cn, language)

	var buf bytes.Buffer
	tmpl := r.pickCreditNoteTemplate(language)
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("pdf: execute credit note template: %w", err)
	}

	return r.htmlToPDF(ctx, buf.String())
}

// htmlToPDF spawns a chromedp context, navigates to a data URL of the
// rendered HTML, and prints to PDF honoring the @page rules. The data
// URL keeps everything in-memory — no temp files, no race window.
func (r *Renderer) htmlToPDF(ctx context.Context, html string) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	allocCtx, cancelAlloc := chromedp.NewContext(runCtx)
	defer cancelAlloc()

	var pdfBytes []byte
	// base64-encode the HTML to dodge URL-fragment / reserved-char
	// problems in data: URLs (e.g. # and & in inline CSS would otherwise
	// truncate the document and produce a blank PDF).
	dataURL := "data:text/html;charset=utf-8;base64," + base64.StdEncoding.EncodeToString([]byte(html))

	err := chromedp.Run(allocCtx,
		chromedp.Navigate(dataURL),
		chromedp.ActionFunc(func(c context.Context) error {
			out, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(c)
			if err != nil {
				return err
			}
			pdfBytes = out
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("pdf: chromedp render: %w", err)
	}
	return pdfBytes, nil
}

// buildInvoiceView formats every cents amount + date for the template.
// Centralised here so the template remains presentation-only.
func buildInvoiceView(inv *invoicing.Invoice, language string) invoiceView {
	items := make([]itemView, 0, len(inv.Items))
	for _, it := range inv.Items {
		items = append(items, itemView{
			Description: it.Description,
			Quantity:    formatQuantity(it.Quantity),
			UnitPrice:   formatAmount(it.UnitPriceCents, language),
			Amount:      formatAmount(it.AmountCents, language),
		})
	}
	return invoiceView{
		Number:             inv.Number,
		IssuedAt:           formatDate(inv.IssuedAt, language),
		ServicePeriodStart: formatDate(inv.ServicePeriodStart, language),
		ServicePeriodEnd:   formatDate(inv.ServicePeriodEnd, language),
		Issuer:             inv.IssuerSnapshot,
		Recipient:          inv.RecipientSnapshot,
		Items:              items,
		AmountExcl:         formatAmount(inv.AmountExclTaxCents, language),
		VATAmount:          formatAmount(inv.VATAmountCents, language),
		VATRate:            formatRate(inv.VATRate),
		AmountIncl:         formatAmount(inv.AmountInclTaxCents, language),
		Mentions:           append([]string(nil), inv.MentionsRendered...),
	}
}

// buildCreditNoteView is the avoir twin of buildInvoiceView. Note we
// do NOT have the parent invoice number on the credit note entity
// itself (only the FK) — the caller is expected to fetch and inject
// it via a future hook. For V1 the template renders the OriginalInvoiceID
// fallback so PDFs still generate end-to-end.
func buildCreditNoteView(cn *invoicing.CreditNote, language string) creditNoteView {
	parentLabel := cn.OriginalInvoiceID.String()
	return creditNoteView{
		Number:                cn.Number,
		IssuedAt:              formatDate(cn.IssuedAt, language),
		OriginalInvoiceNumber: parentLabel,
		Reason:                cn.Reason,
		Issuer:                cn.IssuerSnapshot,
		Recipient:             cn.RecipientSnapshot,
		AmountExcl:            formatAmount(cn.AmountExclTaxCents, language),
		VATAmount:             formatAmount(cn.VATAmountCents, language),
		VATRate:               formatRate(cn.VATRate),
		AmountIncl:            formatAmount(cn.AmountInclTaxCents, language),
		Mentions:              append([]string(nil), cn.MentionsRendered...),
	}
}

// formatAmount turns cents into "1 234,56 €" (FR) or "EUR 1,234.56" (EN).
func formatAmount(cents int64, language string) string {
	euros := cents / 100
	cs := cents % 100
	if cs < 0 {
		cs = -cs
	}
	if isFR(language) {
		return fmt.Sprintf("%s,%02d €", groupThousands(euros, ' '), cs)
	}
	return fmt.Sprintf("EUR %s.%02d", groupThousands(euros, ','), cs)
}

// formatQuantity drops the decimal when the quantity is a whole
// number (most invoice items are 1-unit, no need for "1.000").
func formatQuantity(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}

// formatDate renders a date as "25/04/2026" (FR) or "April 25, 2026" (EN).
func formatDate(t time.Time, language string) string {
	if t.IsZero() {
		return ""
	}
	if isFR(language) {
		return t.Format("02/01/2006")
	}
	return t.Format("January 2, 2006")
}

// formatRate prints a percentage like "0" or "20" (no decimals when
// the rate is a whole number, two when it isn't).
func formatRate(rate float64) string {
	if rate == float64(int64(rate)) {
		return fmt.Sprintf("%d", int64(rate))
	}
	return fmt.Sprintf("%.2f", rate)
}

// isFR centralises the FR-vs-EN decision so the rest of the package
// agrees on the fallback.
func isFR(language string) bool {
	return strings.EqualFold(strings.TrimSpace(language), "fr")
}

// groupThousands inserts a thousands separator. Avoids golang.org/x/text.
func groupThousands(n int64, sep rune) string {
	if n < 0 {
		return "-" + groupThousands(-n, sep)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	pre := len(s) % 3
	if pre > 0 {
		out = append(out, s[:pre]...)
		if len(s) > pre {
			// `sep` is a callsite-controlled separator rune (',' or
			// ' ').  The caller passes ASCII only; byte() is safe.
			out = append(out, byte(sep)) // #nosec G115 -- ASCII separator only
		}
	}
	for i := pre; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, byte(sep)) // #nosec G115 -- ASCII separator only
		}
	}
	return string(out)
}
