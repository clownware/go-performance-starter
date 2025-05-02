package webutil

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// FormErrors maps form field names to validation error messages.
type FormErrors map[string]string

// TODO: Implement template caching for production (Phase 10)

// Base directory for templates
const templateBaseDir = "web/templates"

// Default template functions
var functions = template.FuncMap{
	"default": func(def, val interface{}) interface{} {
		// If val is the zero value, return def; otherwise return val
		switch v := val.(type) {
		case string:
			if v == "" {
				return def
			}
		case nil:
			return def
		}
		return val
	},
	"rawHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"currentYear": func() int {
		return time.Now().Year()
	},
	// Add other global template functions here
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	// Arithmetic helper for templates
	"add": func(a, b int) int { return a + b },
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict helper: key/value pairs required")
		}
		m := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict helper: keys must be strings")
			}
			m[key] = values[i+1]
		}
		return m, nil
	},
}

// RenderTemplate renders the specified template(s) with the provided data.
// It handles parsing layouts, pages, and partials.
func RenderTemplate(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]interface{}) {
	// Add default data
	if data == nil {
		data = make(map[string]interface{})
	}
	// Add current year for footer, etc.
	data["CurrentYear"] = time.Now().Year()
	// Add CSRF token if available (will be added in Phase 5)
	// data["CSRFToken"] = nosurf.Token(r)

	// Get the base template path
	baseTemplate := filepath.Join(templateBaseDir, "layouts", "base.html")
	pageTemplate := filepath.Join(templateBaseDir, page)

	// Find all required files (base, page, partials, components)
	files, err := filepath.Glob(filepath.Join(templateBaseDir, "layouts", "*.html"))
	if err != nil {
		log.Printf("ERROR: Glob layouts failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	partials, err := filepath.Glob(filepath.Join(templateBaseDir, "partials", "*.html"))
	if err != nil {
		log.Printf("ERROR: Glob partials failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	components, err := filepath.Glob(filepath.Join(templateBaseDir, "components", "*.html"))
	if err != nil {
		log.Printf("ERROR: Glob components failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	files = append(files, pageTemplate)
	files = append(files, partials...)
	files = append(files, components...)

	// Parse the templates
	ts, err := template.New(filepath.Base(baseTemplate)).Funcs(functions).ParseFiles(files...)
	if err != nil {
		log.Printf("ERROR: Parsing template files failed: %v", err)
		log.Printf("Files attempted: %v", files)
		// Show error in browser for dev/debug
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<pre style='color: red; background: #fff0f0; padding: 1em; border: 1px solid #f00;'>Template Parse Error:\n" + err.Error() + "\nFiles attempted:\n" + template.HTMLEscapeString(strings.Join(files, "\n")) + "</pre>"))
		return
	} 

	// Use a buffer to capture the output and handle execution errors
	buf := new(bytes.Buffer)
	err = ts.ExecuteTemplate(buf, filepath.Base(baseTemplate), data)
	if err != nil {
		log.Printf("ERROR: Executing template failed: %v", err)
		// Show error in browser for dev/debug
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<pre style='color: red; background: #fff0f0; padding: 1em; border: 1px solid #f00;'>Template Execution Error:\n" + err.Error() + "</pre>"))
		return
	}

	// Write the status code and the rendered template
	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Printf("ERROR: Writing template output failed: %v", err)
		// Don't try to write another header here, connection might be broken
	}
}

// Helper to easily render with errors
func RenderTemplateWithErrors(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]interface{}, errors FormErrors) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Errors"] = errors // Add the errors map under the key "Errors"
	RenderTemplate(w, r, status, page, data)
}
