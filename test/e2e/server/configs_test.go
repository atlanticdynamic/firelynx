//go:build e2e
// +build e2e

package server

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
)

//go:embed testdata/*.tmpl
var Templates embed.FS

func init() {
	logger := slog.Default()

	fs.WalkDir(Templates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			logger.Debug("Example template loaded", "path", path)
		}
		return nil
	})
}

// ProcessTemplate applies values to a template file and returns the result
func ProcessTemplate(templatePath string, data TemplateData) ([]byte, error) {
	templateContent, err := Templates.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("config").Parse(string(templateContent))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
