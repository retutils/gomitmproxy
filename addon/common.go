package addon

import (
	"fmt"

	"github.com/retutils/gomitmproxy/proxy"
	"github.com/samber/lo"
	"github.com/tidwall/match"
)

// MapFrom defines the source criteria for mapping
type MapFrom struct {
	Protocol string   `json:"protocol"`
	Host     string   `json:"host"`
	Method   []string `json:"method"`
	Path     string   `json:"path"`
}

// Match checks if the request matches the criteria
func (mf *MapFrom) Match(req *proxy.Request) bool {
	if mf.Protocol != "" && mf.Protocol != req.URL.Scheme {
		return false
	}
	if mf.Host != "" && mf.Host != req.URL.Host {
		return false
	}
	if len(mf.Method) > 0 && !lo.Contains(mf.Method, req.Method) {
		return false
	}
	if mf.Path != "" && !match.Match(req.URL.Path, mf.Path) {
		return false
	}
	return true
}

// Validate checks if the fields are valid
func (mf *MapFrom) Validate() error {
	if mf.Protocol != "" && mf.Protocol != "http" && mf.Protocol != "https" {
		return fmt.Errorf("invalid protocol %v", mf.Protocol)
	}
	return nil
}
