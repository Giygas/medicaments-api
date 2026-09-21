package ansmdocs

import (
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

// docPolicy is the single content policy, built once: bluemonday policies
// are not safe to build concurrently but are safe to use concurrently once
// created, which matches this package's usage.
var docPolicy = sync.OnceValue(func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	// Whitelist exactly the elements a document body may use (spec
	// decision 5): text emphasis, lists, tables and the sub-heading levels
	// the extractor emits. Every other element (div, span, a, img, …) is
	// stripped; disallowed elements keep their text children while script
	// and style content is dropped entirely by bluemonday.
	p.AllowElements(
		"p", "strong", "em", "b", "i", "u",
		"ul", "ol", "li",
		"table", "thead", "tbody", "tr", "td", "th",
		"sub", "sup", "h3", "h4", "br",
	)
	// The only attributes allowed anywhere, and only on table cells.
	p.AllowAttrs("colspan", "rowspan").OnElements("td", "th")
	return p
})

// Sanitize strips everything from an HTML fragment except the whitelisted
// elements and the colspan/rowspan cell attributes. Classes, styles, ids
// and every other attribute are removed.
func Sanitize(fragment string) string {
	return docPolicy().Sanitize(fragment)
}
