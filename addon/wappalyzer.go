package addon

import (
	"strings"
	"sync"

	"github.com/projectdiscovery/wappalyzergo"
	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
	log "github.com/sirupsen/logrus"
)

type WappalyzerAddon struct {
	proxy.BaseAddon
	wappalyzer *wappalyzer.Wappalyze
	storage    *storage.Service
	
	// Cache for host-level results to avoid redundant database writes
	hostCache   map[string]map[string]bool // hostname -> techName -> true
	hostCacheMu sync.RWMutex
}

func NewWappalyzerAddon(svc *storage.Service) *WappalyzerAddon {
	w, err := wappalyzer.New()
	if err != nil {
		log.Errorf("Failed to initialize wappalyzer: %v", err)
		return nil
	}

	return &WappalyzerAddon{
		wappalyzer: w,
		storage:    svc,
		hostCache:  make(map[string]map[string]bool),
	}
}

func (a *WappalyzerAddon) Response(f *proxy.Flow) {
	if f.Response == nil || f.Response.Body == nil || a.storage == nil {
		return
	}

	// 1. Resource Capping: Skip body analysis for responses exceeding 1MB
	if len(f.Response.Body) > 1024*1024 {
		return
	}

	// 2. Content Type Check: Only analyze text-based content
	contentType := f.Response.Header.Get("Content-Type")
	if !isTextContent(contentType) {
		return
	}

	// Perform analysis asynchronously
	go a.analyzeFlow(f)
}

func (a *WappalyzerAddon) analyzeFlow(f *proxy.Flow) {
	hostname := f.Request.URL.Hostname()
	
	body, err := f.Response.DecodedBody()
	if err != nil {
		return
	}

	// 3. Strategic Content Sampling
	sampledBody := a.sampleBody(body, f.Response.Header.Get("Content-Type"))

	// 4. Analysis
	// wappalyzergo FingerprintWithCats takes http.Header and []byte
	techs := a.wappalyzer.FingerprintWithCats(f.Response.Header, sampledBody)
	
	if len(techs) == 0 {
		return
	}

	// 5. Host-Aggregated Persistence with Caching
	var newTechs []storage.HostTechnology
	
	a.hostCacheMu.Lock()
	if _, ok := a.hostCache[hostname]; !ok {
		a.hostCache[hostname] = make(map[string]bool)
	}
	
	for techName, info := range techs {
		if !a.hostCache[hostname][techName] {
			a.hostCache[hostname][techName] = true
			
			var catNames []string
			mapping := wappalyzer.GetCategoriesMapping()
			for _, catID := range info.Cats {
				if cat, ok := mapping[catID]; ok {
					catNames = append(catNames, cat.Name)
				}
			}
			categories := strings.Join(catNames, ", ")

			newTechs = append(newTechs, storage.HostTechnology{
				Hostname:   hostname,
				TechName:   techName,
				Categories: categories,
				Version:    "", // version detection is limited in standard Analyze
			})
		}
	}
	a.hostCacheMu.Unlock()

	if len(newTechs) > 0 {
		if err := a.storage.SaveHostTechnologies(hostname, newTechs); err != nil {
			log.Errorf("WappalyzerAddon: failed to save host techs for %s: %v", hostname, err)
		}
		log.Infof("[TECH DETECTED] %s: %v", hostname, techs)
	}
}

func (a *WappalyzerAddon) sampleBody(body []byte, contentType string) []byte {
	if !strings.Contains(strings.ToLower(contentType), "html") {
		return body
	}

	if len(body) <= 65536 {
		return body
	}

	// Keep head (first 32KB) and footer (last 32KB)
	res := make([]byte, 65536)
	copy(res, body[:32768])
	copy(res[32768:], body[len(body)-32768:])
	return res
}
