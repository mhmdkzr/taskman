// Package components contains the server-rendered web components.
package components

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// Home renders the application landing page.
func Home() templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Loop</title>
</head>
<body>
  <main>
    <h1>Loop</h1>
    <p>SQLite-backed agent runtime.</p>
    <button data-on:click="@post('/')">Refresh</button>
  </main>
</body>
</html>`)
		if err != nil {
			return fmt.Errorf("render home component: %w", err)
		}
		return nil
	})
}
