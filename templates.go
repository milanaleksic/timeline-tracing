package timelineTracing

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templateDatadogTraceLink.html
//go:embed template.html
//go:embed openWithPerfetto.html
var embeddedData embed.FS

type templateName string

const (
	templateDatadogHtml templateName = "templateDatadogTraceLink.html"
	templateHtml        templateName = "template.html"
)

func loadTemplate(templateName templateName) (*template.Template, error) {
	t, err := template.ParseFS(embeddedData, string(templateName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the template file %q: %w", templateName, err)
	}
	return t, nil
}
