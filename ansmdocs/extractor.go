// Package ansmdocs implements the ANSM document pipeline: a polite HTTP
// fetcher for base-donnees-publique.medicaments.gouv.fr "extrait" pages
// (global rate limiting, one retry honoring Retry-After, singleflight
// deduplication), a goquery-based extractor that turns the page's
// #tabpanel-rcp-panel / #tabpanel-notice-panel into the final sectioned JSON
// document, and a bluemonday sanitizer enforcing the strict content
// whitelist. The produced Document bytes are ready to be cached via
// docstore.Put.
package ansmdocs

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/unicode/norm"

	"github.com/giygas/medicaments-api/logging"
)

// Document types supported by this package. Values deliberately match the
// docstore DocType* constants and the /v1/medicaments/{cis}/{type} URL
// segments so no translation layer is needed downstream.
const (
	TypeRCP    = "rcp"
	TypeNotice = "notice"
)

// SourceAttribution is the source credit every produced document must carry
// (Etalab 2.0 licence obligation on ANSM data reuse).
const SourceAttribution = "ANSM - Base de données publique des médicaments"

// ErrNotAvailable reports that the requested document does not exist on the
// ANSM page for the given CIS: the corresponding tab panel is absent (or
// upstream answered with a definitive 404/410). The caller must record a
// tombstone and never refetch this (cis, docType) pair.
var ErrNotAvailable = errors.New("ansmdocs: document not available on the ANSM page")

// Document is the final sectioned JSON schema served by the API and cached
// (gzipped) by docstore. CIS, Type, Titre, MiseAJour and Source form the
// envelope; Sections carries the body.
type Document struct {
	CIS       string    `json:"cis"`
	Type      string    `json:"type"`
	Titre     string    `json:"titre"`
	MiseAJour string    `json:"miseAJour"`
	Source    string    `json:"source"`
	Sections  []Section `json:"sections"`
}

// Section is one rubrique of a document. Contenu holds sanitized HTML
// restricted to the whitelist enforced by Sanitize.
type Section struct {
	ID      string `json:"id"`
	Titre   string `json:"titre"`
	Contenu string `json:"contenu"`
}

// Extract parses the raw HTML of an ANSM medicament "extrait" page and builds
// the final sectioned Document for the requested docType.
//
// Absence semantics: ANSM pages render one tab panel per existing document,
// so when the requested panel is missing while the page is otherwise valid
// (its <h3> medicament title is present), Extract returns ErrNotAvailable —
// the signal for the caller to write a tombstone. When the page carries
// neither the panel nor a recognizable title it is considered malformed or
// structurally changed, and a wrapped error is returned instead, so a
// permanent tombstone is never recorded from a broken page.
func Extract(cis, docType string, pageHTML []byte) (*Document, error) {
	switch docType {
	case TypeRCP, TypeNotice:
	default:
		return nil, fmt.Errorf("ansmdocs: unsupported docType %q (want %q or %q)", docType, TypeRCP, TypeNotice)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("ansmdocs: failed to parse page HTML: %w", err)
	}

	panelID := "#tabpanel-" + docType + "-panel"
	panel := doc.Find(panelID).First()
	titre := pageTitle(doc)

	if panel.Length() == 0 {
		if titre == "" {
			return nil, fmt.Errorf("ansmdocs: page has no <h3> medicament title and no %s panel, refusing to infer document absence from a malformed page", panelID)
		}
		return nil, ErrNotAvailable
	}
	if titre == "" {
		return nil, errors.New("ansmdocs: page title <h3> not found")
	}

	miseAJour := parseDateNotif(panel.Find("p.DateNotif").First().Text())
	if miseAJour == "" {
		return nil, fmt.Errorf("ansmdocs: update date not found in %s (expected p.DateNotif \"ANSM - Mis à jour le : JJ/MM/AAAA\")", panelID)
	}

	sections := buildSections(panel, docType)
	if len(sections) == 0 {
		return nil, fmt.Errorf("ansmdocs: no sections found in %s", panelID)
	}

	return &Document{
		CIS:       cis,
		Type:      docType,
		Titre:     titre,
		MiseAJour: miseAJour,
		Source:    SourceAttribution,
		Sections:  sections,
	}, nil
}

// dateNotifRe matches the dd/mm/yyyy date inside a p.DateNotif text such as
// "ANSM - Mis à jour le : 07/11/2025".
var dateNotifRe = regexp.MustCompile(`(\d{2})/(\d{2})/(\d{4})`)

// parseDateNotif normalizes a p.DateNotif text into YYYY-MM-DD, returning ""
// when no plausible date is present.
func parseDateNotif(text string) string {
	m := dateNotifRe.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	if day < 1 || day > 31 || month < 1 || month > 12 {
		return ""
	}
	return m[3] + "-" + m[2] + "-" + m[1]
}

// pageTitle returns the medicament name from the page-level <h3> that sits
// next to the document tabs, with whitespace collapsed.
func pageTitle(doc *goquery.Document) string {
	h := doc.Find("main h3")
	if h.Length() == 0 {
		h = doc.Find("h3")
	}
	return collapseSpaces(h.First().Text())
}

// Selectors describing real ANSM markup (verified against
// base-donnees-publique.medicaments.gouv.fr, DSFR + embedded legacy annexe):
//
//   - section headings are <p> elements carrying Amm*Titre level 1/2 classes
//     (or AmmNomRubrique on older pages); level 3/4 classes are subheadings
//     of the current section and are demoted to inline <h4>;
//   - wrapperSelector lists structural elements that may sit between the tab
//     panel and the document flow and are transparently descended into when
//     they contain headings (tables never match, so they stay content);
//   - furnitureSelector removes DSFR chrome (sidemenu summary, tooltips,
//     back-to-top links, print buttons), the embedded legacy <head> and the
//     DateNotif metadata line: none of it belongs to document content.
const (
	sectionHeadingSelector = "p.AmmAnnexeTitre1, p.AmmAnnexeTitre2, p.AmmNoticeTitre1, p.AmmNoticeTitre2, p.AmmNomRubrique"
	inlineHeadingSelector  = "p.AmmAnnexeTitre3, p.AmmAnnexeTitre4, p.AmmNoticeTitre3, p.AmmNoticeTitre4"
	wrapperSelector        = "div, body, html, section, main, article, aside, center"
	furnitureSelector      = "script, style, noscript, head, title, meta, link, nav, .fr-sidemenu, .fr-tooltip, .fr-no-print, .print-button, p.DateNotif"
)

// buildSections walks the tab panel, splits its content on section headings
// and returns the finished sections with IDs assigned per docType and
// contenu sanitized.
func buildSections(panel *goquery.Selection, docType string) []Section {
	root := panel.Clone()
	root.Find(furnitureSelector).Remove()
	root.Find(`a[href="#top"]`).Remove()
	// Level 3/4 rubriques are subheadings of the current section: demote
	// them to inline <h4> so they stay inside their section's contenu
	// (h4 is part of the sanitizer whitelist).
	root.Find(inlineHeadingSelector).Each(func(_ int, s *goquery.Selection) {
		s.ReplaceWithHtml("<h4>" + html.EscapeString(collapseSpaces(s.Text())) + "</h4>")
	})

	b := &sectionBuilder{}
	b.walk(root)

	ids := newIDAssigner(docType)
	out := make([]Section, 0, len(b.raw))
	for _, r := range b.raw {
		out = append(out, Section{
			ID:      ids.assign(r.title),
			Titre:   r.title,
			Contenu: normalizeContenu(Sanitize(strings.Join(r.fragments, ""))),
		})
	}
	return dropEmptyParents(out)
}

// rawSection accumulates the raw HTML fragments emitted between two headings.
type rawSection struct {
	title     string
	fragments []string
}

// sectionBuilder splits a subtree into raw sections.
type sectionBuilder struct {
	raw []rawSection
}

// walk visits the children of sel in document order: headings open a new
// section, structural wrappers containing headings are descended into, and
// everything else is appended to the current section. Content encountered
// before the first heading (page furniture leftovers) is discarded.
func (b *sectionBuilder) walk(sel *goquery.Selection) {
	sel.Children().Each(func(_ int, child *goquery.Selection) {
		if child.Is(sectionHeadingSelector) {
			if title := headingTitle(child); title != "" {
				b.raw = append(b.raw, rawSection{title: title})
			}
			return
		}
		if child.Is(wrapperSelector) && child.Find(sectionHeadingSelector).Length() > 0 {
			b.walk(child)
			return
		}
		if len(b.raw) == 0 {
			return
		}
		outer, err := goquery.OuterHtml(child)
		if err != nil {
			logging.Debug("Skipping unrenderable node in ANSM document", "error", err)
			return
		}
		b.raw[len(b.raw)-1].fragments = append(b.raw[len(b.raw)-1].fragments, outer)
	})
}

// headingTitle extracts the human-readable title of a heading element,
// dropping the DSFR back-to-top links and tooltips ANSM injects into every
// rubrique heading.
func headingTitle(s *goquery.Selection) string {
	clone := s.Clone()
	clone.Find(`a[href="#top"], .fr-tooltip`).Remove()
	return collapseSpaces(clone.Text())
}

// dropEmptyParents removes structural parent rubriques (4, 5, 6 …) that carry
// no content of their own and are immediately followed by their first child
// rubrique (4.1, 5.1 …): the numbered children are the real sections.
func dropEmptyParents(sections []Section) []Section {
	out := make([]Section, 0, len(sections))
	for i, s := range sections {
		if strings.TrimSpace(s.Contenu) == "" && i+1 < len(sections) &&
			strings.HasPrefix(sections[i+1].ID, s.ID+".") {
			continue
		}
		out = append(out, s)
	}
	return out
}

// idAssigner hands out stable, unique section IDs. RCP rubriques follow the
// canonical RCP numbering (1-12, 4.1-4.9, 5.1-5.3, 6.1-6.6): the heading's
// own leading number when present, otherwise the canonical rubrique matched
// by title. Notices have a different, mostly unnumbered rubric structure, so
// their IDs are slugified titles (spec decision 2). Unmatched RCP headings
// fall back to slugified titles, and collisions are suffixed -2, -3 …
type idAssigner struct {
	rcp  bool
	used map[string]int
}

// newIDAssigner returns an assigner for the given docType.
func newIDAssigner(docType string) *idAssigner {
	return &idAssigner{rcp: docType == TypeRCP, used: make(map[string]int)}
}

// assign returns the ID for a section title, deduplicating repeats.
func (a *idAssigner) assign(title string) string {
	base := slugify(title)
	if a.rcp {
		base = rcpSectionID(title)
	}
	if base == "" {
		base = "section"
	}
	n := a.used[base]
	a.used[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n+1)
}

// leadingRubriqueNumberRe matches the "4.2." prefix of rubrique titles such
// as "4.2. Posologie et mode d'administration" or "1. DENOMINATION…".
var leadingRubriqueNumberRe = regexp.MustCompile(`^\(?\s*(\d{1,2}(?:\.\d{1,2}){0,2})[.)]?\s+`)

// rcpSectionID derives the canonical RCP rubrique number for a heading
// title: explicit leading number first, then canonical table matching, then
// a slugified-title fallback.
func rcpSectionID(title string) string {
	if m := leadingRubriqueNumberRe.FindStringSubmatch(title); m != nil {
		return strings.TrimSuffix(m[1], ".")
	}
	if id, ok := matchCanonicalRubric(title); ok {
		return id
	}
	return slugify(title)
}

// rcpCanonicalRubrics is the canonical RCP table of contents.
var rcpCanonicalRubrics = []struct{ id, title string }{
	{"1", "Dénomination du médicament"},
	{"2", "Composition qualitative et quantitative"},
	{"3", "Forme pharmaceutique"},
	{"4", "Données cliniques"},
	{"4.1", "Indications thérapeutiques"},
	{"4.2", "Posologie et mode d'administration"},
	{"4.3", "Contre-indications"},
	{"4.4", "Mises en garde spéciales et précautions d'emploi"},
	{"4.5", "Interactions avec d'autres médicaments et autres formes d'interactions"},
	{"4.6", "Fertilité, grossesse et allaitement"},
	{"4.7", "Effets sur l'aptitude à conduire des véhicules et à utiliser des machines"},
	{"4.8", "Effets indésirables"},
	{"4.9", "Surdosage"},
	{"5", "Propriétés pharmacologiques"},
	{"5.1", "Propriétés pharmacodynamiques"},
	{"5.2", "Propriétés pharmacocinétiques"},
	{"5.3", "Données de sécurité préclinique"},
	{"6", "Données pharmaceutiques"},
	{"6.1", "Liste des excipients"},
	{"6.2", "Incompatibilités"},
	{"6.3", "Durée de conservation"},
	{"6.4", "Précautions particulières de conservation"},
	{"6.5", "Nature et contenu de l'emballage extérieur"},
	{"6.6", "Précautions particulières d'élimination et de manipulation"},
	{"7", "Titulaire de l'autorisation de mise sur le marché"},
	{"8", "Numéro(s) d'autorisation de mise sur le marché"},
	{"9", "Date de première autorisation/de renouvellement de l'autorisation"},
	{"10", "Date de mise à jour du texte"},
	{"11", "Dosimétrie"},
	{"12", "Instructions pour la préparation des radiopharmaceutiques"},
}

// rcpCanonicalIndex is the normalized lookup form of rcpCanonicalRubrics.
var rcpCanonicalIndex = func() map[string]string {
	m := make(map[string]string, len(rcpCanonicalRubrics))
	for _, r := range rcpCanonicalRubrics {
		m[normalizeTitle(r.title)] = r.id
	}
	return m
}()

// matchCanonicalRubric resolves a rubrique title to its canonical RCP number,
// first by exact normalized equality, then by containment (longest canonical
// title wins) to absorb small wording variations.
func matchCanonicalRubric(title string) (string, bool) {
	n := normalizeTitle(title)
	if id, ok := rcpCanonicalIndex[n]; ok {
		return id, true
	}
	if len(n) < 4 {
		return "", false
	}
	bestID, bestLen := "", 0
	for norm, id := range rcpCanonicalIndex {
		if len(norm) <= bestLen {
			continue
		}
		if strings.Contains(n, norm) || strings.Contains(norm, n) {
			bestID, bestLen = id, len(norm)
		}
	}
	if bestID != "" {
		return bestID, true
	}
	return "", false
}

// normalizeTitle prepares a rubrique title for canonical matching: accents
// folded, uppercased, punctuation turned into spaces, whitespace collapsed.
func normalizeTitle(s string) string {
	mapped := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, foldAccents(strings.ToUpper(s)))
	return strings.Join(strings.Fields(mapped), " ")
}

// foldAccents replaces accented runes with their base ASCII form by dropping
// the combining marks of the NFD decomposition (é → e).
func foldAccents(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(s))
}

// slugify turns a heading title into a stable URL-friendly ID: accents
// folded, lowercased, every non-alphanumeric run collapsed to a single "-"
// (leading and trailing separators dropped).
func slugify(s string) string {
	var b strings.Builder
	lastHyphen := true
	for _, r := range strings.ToLower(foldAccents(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// collapseSpaces trims and squeezes internal whitespace runs (headings and
// titles carry newlines and repeated spaces in the source markup).
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
