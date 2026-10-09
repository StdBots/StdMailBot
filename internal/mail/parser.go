package mail

import (
	"html"
	"regexp"
	"strings"
)

var (
	otpRegex      = regexp.MustCompile(`\b\d{4,8}\b`)
	linkRegex     = regexp.MustCompile(`https?://[^\s<>"'\)]+`)
	scriptRegex   = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	tagRegex      = regexp.MustCompile(`<[^>]+>`)
	newlineRegex  = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</tr>|</div>`)
	spaceRegex    = regexp.MustCompile(`[ \t]+`)
	multilineRgx  = regexp.MustCompile(`\n{3,}`)
)

// ExtractOTP finds verification codes (4 to 8 digits), ignoring year numbers (19xx, 20xx)
func ExtractOTP(text string) string {
	matches := otpRegex.FindAllString(text, -1)
	for _, m := range matches {
		if !strings.HasPrefix(m, "19") && !strings.HasPrefix(m, "20") {
			return m
		}
	}
	return ""
}

// ExtractLinks finds and separates magic/verification links from ordinary links
func ExtractLinks(text string) (magicLinks []string, normalLinks []string) {
	text = html.UnescapeString(text)
	rawMatches := linkRegex.FindAllString(text, -1)

	seen := make(map[string]bool)
	keywords := []string{"magic", "login", "verify", "confirm", "token", "activate", "auth", "click", "reset"}

	for _, raw := range rawMatches {
		clean := strings.Trim(raw, ".,;:\"')>")
		if !strings.HasPrefix(clean, "http") || seen[clean] {
			continue
		}
		seen[clean] = true

		lower := strings.ToLower(clean)
		isMagic := false
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				isMagic = true
				break
			}
		}

		if isMagic {
			magicLinks = append(magicLinks, clean)
		} else {
			normalLinks = append(normalLinks, clean)
		}
	}

	return magicLinks, normalLinks
}

// CleanHTML converts raw email HTML into clean readable text
func CleanHTML(rawHTML string) string {
	if rawHTML == "" {
		return ""
	}

	text := html.UnescapeString(rawHTML)
	text = scriptRegex.ReplaceAllString(text, "")
	text = newlineRegex.ReplaceAllString(text, "\n")
	text = tagRegex.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\r", "")
	text = spaceRegex.ReplaceAllString(text, " ")
	text = multilineRgx.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}
