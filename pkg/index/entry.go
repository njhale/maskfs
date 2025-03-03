package index

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"time"
)

// GetEntry fetches file metadata and returns an Entry
func GetEntry(fsys fs.FS, path string) (*Entry, error) {
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return nil, err
	}

	linkPath, err := url.JoinPath("/files", url.PathEscape(path))
	if err != nil {
		return nil, err
	}

	return &Entry{
		Name:     info.Name(),
		Size:     info.Size(),
		Mode:     info.Mode(),
		ModTime:  info.ModTime().Format(time.RFC3339),
		IsDir:    info.IsDir(),
		FSPath:   path,
		LinkPath: linkPath,
	}, nil
}

// Entry represents file or directory metadata specifically for directory listing pages
type Entry struct {
	Name     string
	Size     int64
	Mode     fs.FileMode
	ModTime  string
	IsDir    bool
	FSPath   string // File path relative to the filesystem's root directory (leading slash omitted)
	LinkPath string // URL-encoded path for HTML links
}

// Mask masks entries from an index.
type Mask interface {
	// Masked returns true if the entry should be masked.
	Masked(entry *Entry) bool
}

// Entries is a collection of Entry objects
type Entries []*Entry

// GetEntries returns a new index of entries from the given path.
// If a mask is provided, it will be used to filter the entries.
func GetEntries(fsys fs.FS, path string, mask Mask) (Entries, error) {
	children, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}

	var masked Entries
	for _, child := range children {
		entry, err := GetEntry(fsys, filepath.Join(path, child.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to get entry: %w", err)
		}

		if entry == nil || (mask != nil && mask.Masked(entry)) {
			// The entry is not valid or the entry is masked, skip it
			continue
		}

		masked = append(masked, entry)
	}

	return masked, nil
}

func (e Entries) WriteHTML(w io.Writer, directory *Entry, entries Entries) error {
	if directory == nil {
		return errors.New("invalid directory referenced")
	}
	for _, entry := range entries {
		if entry == nil {
			return errors.New("invalid entry referenced")
		}
	}

	data := struct {
		Directory *Entry
		Entries   Entries
	}{
		Directory: directory,
		Entries:   entries,
	}

	tmpl, err := template.New("directory").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Directory listing for /{{.Directory.FSPath}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { text-align: left; padding: 12px; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; }
        tr:hover { background-color: #f5f5f5; }
        a { color: #0366d6; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Directory listing for /{{.Directory.FSPath}}</h1>
        <table>
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Size</th>
                    <th>Mode</th>
                    <th>Modified</th>
                </tr>
            </thead>
            <tbody>
                {{if ne .Directory.LinkPath "/"}}
                <tr>
                    <td><a href="{{.Directory.LinkPath}}/..">..</a></td>
                    <td>-</td>
                    <td>-</td>
                    <td>-</td>
                </tr>
                {{end}}
                {{range .Entries}}
                <tr>
                    <td><a href="{{.LinkPath}}">{{.Name}}</a></td>
                    <td>{{if .IsDir}}-{{else}}{{.Size}}{{end}}</td>
                    <td>{{.Mode}}</td>
                    <td>{{.ModTime}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>`
