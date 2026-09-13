package exporter

import (
	"context"
	"errors"
	"html"
	"time"

	"fmt"
	"marktyp/internal/document"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phpdave11/gofpdf"
)

var (
	inlineMathPattern  = regexp.MustCompile(`\$([^\n$]+)\$`)
	linkPattern        = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	orderedItemPattern = regexp.MustCompile(`^\d+\.\s+`)
	browserExecutable  = findHeadlessBrowserExecutable
)

type ExportHTMLRequest = document.ExportHTMLRequest
type ExportPDFRequest = document.ExportPDFRequest

const AppName = document.AppName

type Exporter struct{}

func NewExporter() *Exporter {
	return &Exporter{}
}

func (e *Exporter) ExportHTMLToPath(request ExportHTMLRequest, filePath string) error {
	if strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing html content")
	}

	html := wrapHTMLDocument(request.Title, request.HTML)
	return os.WriteFile(filePath, []byte(html), 0o644)
}

func (e *Exporter) ExportPDFToPath(request ExportPDFRequest, filePath string) error {
	if strings.TrimSpace(request.Markdown) == "" && strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing document content")
	}

	html := strings.TrimSpace(request.HTML)
	if html != "" {
		if err := exportPDFViaHeadlessBrowser(request.Title, html, filePath); err == nil {
			return nil
		} else if strings.TrimSpace(request.Markdown) == "" {
			return err
		}
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddPage()
	writeMarkdownToPDF(pdf, request.Title, request.Markdown)

	return pdf.OutputFileAndClose(filePath)
}

func wrapHTMLDocument(title string, body string) string {
	documentTitle := strings.TrimSpace(title)
	if documentTitle == "" {
		documentTitle = AppName + " document"
	}
	katexCSS := loadKatexCSSForExport()

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    @page {
      size: A4;
      margin: 14mm 14mm 16mm;
    }
    :root {
      color: #1d1d1b;
      font-family: Georgia, "Times New Roman", serif;
      line-height: 1.55;
    }
    body {
      margin: 0;
      padding: 0;
      background: #ffffff;
      color: #1d1d1b;
    }
    main {
      max-width: none;
      margin: 0;
      background: transparent;
      border: 0;
      border-radius: 0;
      padding: 0;
    }
    h1, h2, h3 {
      line-height: 1.1;
      margin: 0 0 0.55em;
    }
    p {
      margin: 0 0 0.85em;
    }
    img, svg {
      max-width: 100%%;
      page-break-inside: avoid;
      break-inside: avoid;
    }
    blockquote {
      margin-left: 0;
      padding-left: 16px;
      border-left: 4px solid rgba(181, 87, 47, 0.35);
      color: #6f665f;
    }
    pre, code, .katex, .katex * {
      font-family: "SFMono-Regular", Consolas, monospace;
    }
    .katex {
      position: relative;
      display: inline-block;
      font-family: "Times New Roman", Georgia, serif;
      text-rendering: auto;
    }
    .katex .katex-mathml {
      position: absolute !important;
      width: 1px !important;
      height: 1px !important;
      padding: 0 !important;
      margin: -1px !important;
      overflow: hidden !important;
      clip: rect(0, 0, 0, 0) !important;
      white-space: nowrap !important;
      border: 0 !important;
    }
    .katex-display {
      display: block;
      margin: 0.8em 0 1em;
      text-align: left;
    }
    pre {
      background: #f5f5f2;
      border-radius: 10px;
      padding: 12px;
      overflow: hidden;
    }
    marktyp-mermaid, marktyp-math-inline, marktyp-math-block {
      display: inline-block;
      max-width: 100%%;
    }
    marktyp-math-block {
      display: block;
      margin: 0.8em 0 1em;
    }
    %s
  </style>
</head>
<body>
  <main>
    %s
  </main>
</body>
</html>`, html.EscapeString(documentTitle), katexCSS, body)
}

func loadKatexCSSForExport() string {
	candidates := []string{
		"frontend/node_modules/katex/dist/katex.min.css",
		"katex.min.css",
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil && len(data) > 0 {
			return string(data)
		}
	}

	return ""
}

func exportPDFViaHeadlessBrowser(title string, bodyHTML string, outputPath string) error {
	browserPath, ok := browserExecutable()
	if !ok {
		return errors.New("no compatible browser found for headless pdf")
	}

	tempDir, err := os.MkdirTemp("", AppName+"-pdf-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "export.html")
	if err := os.WriteFile(inputPath, []byte(wrapHTMLDocument(title, bodyHTML)), 0o644); err != nil {
		return err
	}

	fileURL := (&url.URL{Scheme: "file", Path: inputPath}).String()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx,
		browserPath,
		"--headless=new",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--no-sandbox",
		"--allow-file-access-from-files",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=7000",
		"--no-pdf-header-footer",
		"--print-to-pdf="+outputPath,
		"--print-to-pdf-no-header",
		fileURL,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("headless export failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func findHeadlessBrowserExecutable() (string, bool) {
	candidates := []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"microsoft-edge",
		"microsoft-edge-stable",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/microsoft-edge",
		"/usr/bin/microsoft-edge-stable",
	}

	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, true
		}
	}

	return "", false
}

func writeMarkdownToPDF(pdf *gofpdf.Fpdf, title string, markdown string) {
	documentTitle := strings.TrimSpace(title)
	if documentTitle == "" {
		documentTitle = AppName + " document"
	}

	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(0, 10, documentTitle, "", 1, "", false, 0, "")
	pdf.Ln(4)

	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	inCodeBlock := false
	inMermaidBlock := false
	inMathBlock := false
	mathLines := make([]string, 0, 6)

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, " ")
		trimmed := strings.TrimSpace(line)

		if inMathBlock {
			if trimmed == "$$" {
				pdf.SetFont("Helvetica", "I", 11)
				pdf.SetTextColor(47, 53, 66)
				pdf.MultiCell(0, 6, strings.Join(mathLines, "\n"), "", "L", false)
				pdf.SetTextColor(29, 29, 27)
				pdf.Ln(2)
				inMathBlock = false
				mathLines = mathLines[:0]
				continue
			}
			mathLines = append(mathLines, line)
			continue
		}

		if trimmed == "$$" {
			inMathBlock = true
			mathLines = mathLines[:0]
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			language := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			if !inCodeBlock && strings.EqualFold(language, "mermaid") {
				inMermaidBlock = true
			}
			inCodeBlock = !inCodeBlock
			if !inCodeBlock {
				pdf.Ln(2)
				inMermaidBlock = false
			}
			pdf.Ln(2)
			continue
		}

		if inCodeBlock {
			pdf.SetFont("Courier", "", 10)
			if inMermaidBlock {
				pdf.SetTextColor(69, 86, 115)
			} else {
				pdf.SetTextColor(29, 29, 27)
			}
			pdf.MultiCell(0, 5, line, "", "L", false)
			pdf.SetTextColor(29, 29, 27)
			continue
		}

		if trimmed == "" {
			pdf.Ln(3)
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "# "):
			pdf.SetFont("Helvetica", "B", 18)
			pdf.MultiCell(0, 9, renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))), "", "L", false)
			pdf.Ln(1)
		case strings.HasPrefix(trimmed, "## "):
			pdf.SetFont("Helvetica", "B", 15)
			pdf.MultiCell(0, 8, renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))), "", "L", false)
			pdf.Ln(1)
		case strings.HasPrefix(trimmed, "### "):
			pdf.SetFont("Helvetica", "B", 13)
			pdf.MultiCell(0, 7, renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))), "", "L", false)
		case strings.HasPrefix(trimmed, "> "):
			pdf.SetFont("Helvetica", "I", 11)
			pdf.SetTextColor(111, 102, 95)
			pdf.MultiCell(0, 6, renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "> "))), "", "L", false)
			pdf.SetTextColor(29, 29, 27)
		case strings.HasPrefix(trimmed, "- [ ] "):
			pdf.SetFont("Helvetica", "", 11)
			pdf.MultiCell(0, 6, "[ ] "+renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ] "))), "", "L", false)
		case strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] "):
			pdf.SetFont("Helvetica", "", 11)
			done := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x] "), "- [X] "))
			pdf.MultiCell(0, 6, "[x] "+renderInlineMarkdown(done), "", "L", false)
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ "):
			pdf.SetFont("Helvetica", "", 11)
			item := strings.TrimSpace(trimmed[2:])
			pdf.MultiCell(0, 6, "- "+renderInlineMarkdown(item), "", "L", false)
		case orderedItemPattern.MatchString(trimmed):
			pdf.SetFont("Helvetica", "", 11)
			pdf.MultiCell(0, 6, renderInlineMarkdown(trimmed), "", "L", false)
		case strings.HasPrefix(trimmed, "$"):
			pdf.SetFont("Courier", "", 10)
			pdf.MultiCell(0, 6, renderInlineMarkdown(trimmed), "", "L", false)
		case strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "***"):
			y := pdf.GetY() + 1
			pageW, _ := pdf.GetPageSize()
			marginLeft, _, marginRight, _ := pdf.GetMargins()
			pdf.Line(marginLeft, y, pageW-marginRight, y)
			pdf.Ln(4)
		default:
			pdf.SetFont("Helvetica", "", 11)
			pdf.MultiCell(0, 6, renderInlineMarkdown(line), "", "L", false)
		}
	}
}

func renderInlineMarkdown(line string) string {
	out := strings.TrimSpace(line)
	out = strings.ReplaceAll(out, `\`, "")
	out = strings.ReplaceAll(out, "**", "")
	out = strings.ReplaceAll(out, "__", "")
	out = strings.ReplaceAll(out, "*", "")
	out = strings.ReplaceAll(out, "_", "")
	out = strings.ReplaceAll(out, "`", "")
	out = strings.ReplaceAll(out, "~~", "")
	out = strings.ReplaceAll(out, "<br>", " ")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&amp;", "&")

	out = linkPattern.ReplaceAllString(out, `$1 ($2)`)
	out = inlineMathPattern.ReplaceAllString(out, `[$1]`)

	return out
}
