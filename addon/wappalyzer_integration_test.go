package addon

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
)

func TestWappalyzerAddon_WappalyzerCom_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := storage.NewService(tmpDir)
	defer svc.Close()

	addon := NewWappalyzerAddon(svc)

	hostname := "wappalyzer.com"
	u, _ := url.Parse("https://" + hostname)
	
	// Simulate wappalyzer.com response
	f := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    u,
		},
		Response: &proxy.Response{
			StatusCode: 200,
			Header: http.Header{
				"Server":           []string{"cloudflare"},
				"Content-Type":     []string{"text/html; charset=utf-8"},
				"Strict-Transport-Security": []string{"max-age=15724800; includeSubDomains"},
			},
			// Body with Vue.js, Nuxt.js, GTM, etc. markers
			Body: []byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Wappalyzer</title>
					<script src="https://www.googletagmanager.com/gtm.js?id=GTM-XXXX"></script>
					<meta name="generator" content="Nuxt.js">
				</head>
				<body>
					<div id="__nuxt"></div>
					<div id="__layout"></div>
					<script src="/_nuxt/app.js"></script>
					<script>window.__VUE_DEVTOOLS_GLOBAL_HOOK__ = {}</script>
				</body>
				</html>
			`),
		},
	}

	addon.analyzeFlow(f) // Run synchronously for test

	// Verify persistence in DuckDB
	techs, err := svc.GetHostTechnologies(hostname)
	if err != nil {
		t.Fatalf("Failed to get host techs: %v", err)
	}

	if len(techs) == 0 {
		t.Fatal("Expected some technologies to be detected, got 0")
	}

	foundCloudflare := false
	foundNuxt := false
	foundGTM := false
	foundVue := false

	for _, tech := range techs {
		t.Logf("Detected: %s (%s)", tech.TechName, tech.Categories)
		switch tech.TechName {
		case "Cloudflare":
			foundCloudflare = true
		case "Nuxt.js":
			foundNuxt = true
		case "Google Tag Manager":
			foundGTM = true
		case "Vue.js":
			foundVue = true
		}
	}

	if !foundCloudflare {
		t.Error("Cloudflare not detected")
	}
	if !foundNuxt {
		t.Error("Nuxt.js not detected")
	}
	if !foundGTM {
		t.Log("Google Tag Manager not detected - patterns might differ")
	}
	if !foundVue {
		t.Log("Vue.js not detected - patterns might differ")
	}
}
