// Package sanitize provides functions for sanitizing text.
package sanitize

import (
	"bytes"
	"path"
	"regexp"
	"strings"

	parser "golang.org/x/net/html"
)

var (
	ignoreTags = []string{"title", "script", "style", "iframe", "frame", "frameset", "noframes", "noembed", "embed", "applet", "object", "base"}

	defaultTags = []string{"h1", "h2", "h3", "h4", "h5", "h6", "div", "span", "hr", "p", "br", "b", "i", "strong", "em", "ol", "ul", "li", "a", "img", "pre", "code", "blockquote", "article", "section"}

	defaultAttributes = []string{"id", "class", "src", "href", "title", "alt", "name", "rel"}
)

// HTMLAllowing sanitizes html, allowing some tags.
// Arrays of allowed tags and allowed attributes may optionally be passed as the second and third arguments.

// HTML strips html tags, replace common entities, and escapes <>&;'" in the result.
// Note the returned text may contain entities as it is escaped by HTMLEscapeString, and most entities are not translated.

// We are very restrictive as this is intended for ascii url slugs
var illegalPath = regexp.MustCompile(`[^[:alnum:]\~\-\./]`)

// Remove all other unrecognised characters apart from
var illegalName = regexp.MustCompile(`[^[:alnum:]-.]`)

// Name makes a string safe to use in a file name by first finding the path basename, then replacing non-ascii characters.
func Name(s string) string {
	// Start with lowercase string
	fileName := s
	fileName = baseNameSeparators.ReplaceAllString(fileName, "-")

	fileName = path.Clean(path.Base(fileName))

	// Remove illegal characters for names, replacing some common separators with -
	fileName = cleanString(fileName, illegalName)

	// NB this may be of length 0, caller must check
	return fileName
}

// Replace these separators with -
var baseNameSeparators = regexp.MustCompile(`[./]`)

// A very limited list of transliterations to catch common european names translated to urls.
// This set could be expanded with at least caps and many more characters.
var transliterations = map[rune]string{
	'À': "A",
	'Á': "A",
	'Â': "A",
	'Ã': "A",
	'Ä': "A",
	'Å': "AA",
	'Æ': "AE",
	'Ç': "C",
	'È': "E",
	'É': "E",
	'Ê': "E",
	'Ë': "E",
	'Ì': "I",
	'Í': "I",
	'Î': "I",
	'Ï': "I",
	'Ð': "D",
	'Ł': "L",
	'Ñ': "N",
	'Ò': "O",
	'Ó': "O",
	'Ô': "O",
	'Õ': "O",
	'Ö': "OE",
	'Ø': "OE",
	'Œ': "OE",
	'Ù': "U",
	'Ú': "U",
	'Ü': "UE",
	'Û': "U",
	'Ý': "Y",
	'Þ': "TH",
	'ẞ': "SS",
	'à': "a",
	'á': "a",
	'â': "a",
	'ã': "a",
	'ä': "ae",
	'å': "aa",
	'æ': "ae",
	'ç': "c",
	'è': "e",
	'é': "e",
	'ê': "e",
	'ë': "e",
	'ì': "i",
	'í': "i",
	'î': "i",
	'ï': "i",
	'ð': "d",
	'ł': "l",
	'ñ': "n",
	'ń': "n",
	'ò': "o",
	'ó': "o",
	'ô': "o",
	'õ': "o",
	'ō': "o",
	'ö': "oe",
	'ø': "oe",
	'œ': "oe",
	'ś': "s",
	'ù': "u",
	'ú': "u",
	'û': "u",
	'ū': "u",
	'ü': "ue",
	'ý': "y",
	'ÿ': "y",
	'ż': "z",
	'þ': "th",
	'ß': "ss",
}

// Accents replaces a set of accented characters with ascii equivalents.
func Accents(s string) string {
	// Replace some common accent characters
	b := bytes.NewBufferString("")
	for _, c := range s {
		// Check transliterations first
		if val, ok := transliterations[c]; ok {
			b.WriteString(val)
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

var (
	// If the attribute contains data: or javascript: anywhere, ignore it
	// we don't allow this in attributes as it is so frequently used for xss
	// NB we allow spaces in the value, and lowercase.
	illegalAttr = regexp.MustCompile(`(d\s*a\s*t\s*a|j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*)\s*:`)

	// We are far more restrictive with href attributes.
	legalHrefAttr = regexp.MustCompile(`\A[/#][^/\\]?|mailto:|http://|https://`)
)

// cleanAttributes returns an array of attributes after removing malicious ones.
func cleanAttributes(a []parser.Attribute, allowed []string) []parser.Attribute {
	if len(a) == 0 {
		return a
	}

	var cleaned []parser.Attribute
	for _, attr := range a {
		if includes(allowed, attr.Key) {

			val := strings.ToLower(attr.Val)

			// Check for illegal attribute values
			if illegalAttr.FindString(val) != "" {
				attr.Val = ""
			}

			// Check for legal href values - / mailto:// http:// or https://
			if attr.Key == "href" {
				if legalHrefAttr.FindString(val) == "" {
					attr.Val = ""
				}
			}

			// If we still have an attribute, append it to the array
			if attr.Val != "" {
				cleaned = append(cleaned, attr)
			}
		}
	}
	return cleaned
}

// A list of characters we consider separators in normal strings and replace with our canonical separator - rather than removing.
var (
	separators = regexp.MustCompile(`[!&_="#|+?:]`)

	dashes = regexp.MustCompile(`[\-]+`)
)

// cleanString replaces separators with - and removes characters listed in the regexp provided from string.
// Accents, spaces, and all characters not in A-Za-z0-9 are replaced.
func cleanString(s string, r *regexp.Regexp) string {

	// Remove any trailing space to avoid ending on -
	s = strings.Trim(s, " ")

	// Flatten accents first so that if we remove non-ascii we still get a legible name
	s = Accents(s)

	// Replace certain joining characters with a dash
	s = separators.ReplaceAllString(s, "-")

	// Remove all other unrecognised characters - NB we do allow any printable characters
	// s = r.ReplaceAllString(s, "")

	// Remove any multiple dashes caused by replacements above
	s = dashes.ReplaceAllString(s, "-")

	return s
}

// includes checks for inclusion of a string in a []string.
func includes(a []string, s string) bool {
	for _, as := range a {
		if as == s {
			return true
		}
	}
	return false
}
