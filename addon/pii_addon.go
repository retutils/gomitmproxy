package addon

import (
	"regexp"
	"strings"

	ahocorasick "github.com/petar-dambovaliev/aho-corasick"
	"github.com/retutils/gomitmproxy/proxy"
	log "github.com/sirupsen/logrus"
)

type PIIAddon struct {
	proxy.BaseAddon
	ac       ahocorasick.AhoCorasick
	patterns map[string]*regexp.Regexp
	keywords []string
}

func NewPIIAddon() *PIIAddon {
	addon := &PIIAddon{
		patterns: make(map[string]*regexp.Regexp),
	}

	// 1. Initialize Regex Patterns
	addon.patterns["Email"] = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	addon.patterns["IPv4"] = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	addon.patterns["SSN"] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`) // Simple US SSN
	// Simplified Credit Card (Luhn check not strictly applied here, just pattern)
	addon.patterns["CreditCard"] = regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})\b`)

	// 2. Initialize Aho-Corasick for Keywords
	builder := ahocorasick.NewAhoCorasickBuilder(ahocorasick.Opts{
		AsciiCaseInsensitive: true,
		MatchOnlyWholeWords:  false,
	})

	keywords := []string{
		"password",
		"secret",
		"passwd",
		"api_key",
		"apikey",
		"access_token",
		"auth_token",
		"private_key",
		"client_secret",
	}
	addon.keywords = keywords

	addon.ac = builder.Build(keywords)

	return addon
}

type PIIFinding struct {
	Source  string `json:"source"`
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
}

func (a *PIIAddon) Response(f *proxy.Flow) {
	if f.Response == nil || f.Response.Body == nil {
		return
	}

	body, err := f.Response.DecodedBody()
	if err != nil {
		return
	}

	// We convert to string once. Note: Body can be binary, might want to limit scan or check Content-Type.
	// For safety, let's look at Content-Type.
	contentType := f.Response.Header.Get("Content-Type")
	if !isTextContent(contentType) {
		return
	}

	bodyStr := string(body)
	var findings []PIIFinding

	// 1. Regex Scan
	for name, re := range a.patterns {
		if re.MatchString(bodyStr) {
			findings = append(findings, PIIFinding{
				Source:  "response_body",
				Type:    name,
				Snippet: "", // TODO: Extract snippet
			})
		}
	}

	// 2. Aho-Corasick Scan
	matches := a.ac.FindAll(bodyStr)
	if len(matches) > 0 {
		findings = append(findings, PIIFinding{
			Source:  "response_body",
			Type:    "SensitiveKeywords",
			Snippet: "",
		})
	}

	if len(findings) > 0 {
		warnMsg := ""
		for _, f := range findings {
			warnMsg += f.Type + ", "
		}
		warnMsg = strings.TrimSuffix(warnMsg, ", ")

		log.Warnf("[PII DETECTED] %s %s detected: %s", f.Request.Method, f.Request.URL.String(), warnMsg)

		f.Metadata["pii"] = true
	}
}

func isTextContent(contentType string) bool {
	if contentType == "" {
		return false // Assume binary if unknown? Or inspect? Let's be conservative.
	}
	lower := strings.ToLower(contentType)
	return strings.Contains(lower, "text") ||
		strings.Contains(lower, "json") ||
		strings.Contains(lower, "xml") ||
		strings.Contains(lower, "javascript") ||
		strings.Contains(lower, "html")
}
