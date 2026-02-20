package main

import (
	"fmt"
	rawLog "log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/retutils/gomitmproxy/addon"
	"github.com/retutils/gomitmproxy/internal/helper"
	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
	"github.com/retutils/gomitmproxy/web"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	version bool `json:"version"` // show go-mitmproxy version

	Addr         string   `json:"addr"`          // proxy listen addr
	WebAddr      string   `json:"web_addr"`      // web interface listen addr
	SslInsecure  bool     `json:"ssl_insecure"`  // not verify upstream server SSL/TLS certificates.
	IgnoreHosts  []string `json:"ignore_hosts"`  // a list of ignore hosts
	AllowHosts   []string `json:"allow_hosts"`   // a list of allow hosts
	CertPath     string   `json:"cert_path"`     // path of generate cert files
	Debug        int      `json:"debug"`         // debug mode: 1 - print debug log, 2 - show debug from
	Dump         string   `json:"dump"`          // dump filename
	DumpLevel    int      `json:"dump_level"`    // dump level: 0 - header, 1 - header + body
	Upstream     string   `json:"upstream"`      // upstream proxy
	UpstreamCert bool     `json:"upstream_cert"` // Connect to upstream server to look up certificate details. Default: True
	MapRemote    string   `json:"map_remote"`    // map remote config filename
	MapLocal     string   `json:"map_local"`     // map local config filename
	LogFile      string   `json:"log_file"`      // log file path

	filename string `json:"-"` // read config from the filename

	ProxyAuth       string   `json:"proxyauth"`        // Require proxy authentication
	TlsFingerprint  string   `json:"tls_fingerprint"`  // TLS fingerprint to emulate (chrome, firefox, ios, or random)
	FingerprintSave string   `json:"fingerprint_save"` // Save decoding client hello to file
	FingerprintList bool     `json:"fingerprint_list"` // List saved fingerprints
	StorageDir      string   `json:"storage_dir"`      // Directory to store captured flows (DuckDB + Bleve)
	Search          string   `json:"search"`           // Search query for stored flows
	ScanPII         bool     `json:"scan_pii"`         // Enable PII scanning (regex + AC)
	ScanTech        bool     `json:"scan_tech"`        // Enable technology scanning (Wappalyzer)
	DnsResolvers    []string `json:"dns_resolvers"`
	DnsRetries      int      `json:"dns_retries"`
}

func main() {
	config := loadConfig()
	if err := Run(config); err != nil {
		log.Fatal(err)
	}
}

func Run(config *Config) error {
	if config.FingerprintList {
		names, err := proxy.ListFingerprints()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("No saved fingerprints found.")
		} else {
			fmt.Println("Saved fingerprints:")
			for _, name := range names {
				fmt.Printf(" - %s\n", name)
			}
		}
		return nil
	}

	if config.Search != "" {
		if config.StorageDir == "" {
			return fmt.Errorf("-storage_dir is required for search")
		}

		// Initialize service in read-only mode if possible, but NewService opens both
		// For CLI search, we just need to init, search, print, exit
		svc, err := storage.NewService(config.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to open storage: %w", err)
		}
		defer svc.Close()

		results, err := svc.Search(config.Search)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
		} else {
			fmt.Printf("Found %d results:\n", len(results))
			hostTechs := make(map[string][]storage.HostTechnology)
			for _, entry := range results {
				u, _ := url.Parse(entry.URL)
				hostname := u.Hostname()
				if _, ok := hostTechs[hostname]; !ok {
					techs, _ := svc.GetHostTechnologies(hostname)
					hostTechs[hostname] = techs
				}

				techStr := ""
				if len(hostTechs[hostname]) > 0 {
					var names []string
					for _, t := range hostTechs[hostname] {
						names = append(names, t.TechName)
					}
					techStr = fmt.Sprintf(" [Tech: %s]", strings.Join(names, ", "))
				}

				fmt.Printf("[%s] %s %s (Status: %d)%s\n", entry.ID, entry.Method, entry.URL, entry.StatusCode, techStr)
			}
		}
		return nil
	}

	if config.Debug > 0 {
		rawLog.SetFlags(rawLog.LstdFlags | rawLog.Lshortfile)
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if config.Debug == 2 {
		log.SetReportCaller(true)
	}
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	opts := &proxy.Options{
		Debug:             config.Debug,
		Addr:              config.Addr,
		StreamLargeBodies: 1024 * 1024 * 5,
		SslInsecure:       config.SslInsecure,
		CaRootPath:        config.CertPath,
		Upstream:          config.Upstream,
		LogFilePath:       config.LogFile,
		TlsFingerprint:    config.TlsFingerprint,
		FingerprintSave:   config.FingerprintSave,
		DnsResolvers:      config.DnsResolvers,
		DnsRetries:        config.DnsRetries,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		return err
	}

	if config.version {
		fmt.Println("go-mitmproxy: " + p.Version)
		return nil
	}

	log.Infof("go-mitmproxy version %v\n", p.Version)

	if len(config.IgnoreHosts) > 0 {
		p.SetShouldInterceptRule(func(req *http.Request) bool {
			return !helper.MatchHost(req.Host, config.IgnoreHosts)
		})
	}
	if len(config.AllowHosts) > 0 {
		p.SetShouldInterceptRule(func(req *http.Request) bool {
			return helper.MatchHost(req.Host, config.AllowHosts)
		})
	}

	if !config.UpstreamCert {
		p.AddAddon(proxy.NewUpstreamCertAddon(false))
		log.Infoln("UpstreamCert config false")
	}

	if config.ProxyAuth != "" && strings.ToLower(config.ProxyAuth) != "any" {
		log.Infoln("Enable entry authentication")
		auth, err := NewDefaultBasicAuth(config.ProxyAuth)
		if err != nil {
			return err
		}
		p.SetAuthProxy(auth.EntryAuth)
	}

	if config.LogFile != "" {
		// Use instance logger with file output
		p.AddAddon(proxy.NewInstanceLogAddonWithFile(config.Addr, "", config.LogFile))
		log.Infof("Logging to file: %s", config.LogFile)
	} else {
		// Use default logger
		p.AddAddon(&proxy.LogAddon{})
	}
	p.AddAddon(web.NewWebAddon(config.WebAddr))

	if config.MapRemote != "" {
		mapRemote, err := addon.NewMapRemoteFromFile(config.MapRemote)
		if err != nil {
			log.Warnf("load map remote error: %v", err)
		} else {
			p.AddAddon(mapRemote)
		}
	}

	if config.MapLocal != "" {
		mapLocal, err := addon.NewMapLocalFromFile(config.MapLocal)
		if err != nil {
			log.Warnf("load map local error: %v", err)
		} else {
			p.AddAddon(mapLocal)
		}
	}

	if config.Dump != "" {
		dumper := addon.NewDumperWithFilename(config.Dump, config.DumpLevel)
		p.AddAddon(dumper)
	}

	if config.ScanPII {
		p.AddAddon(addon.NewPIIAddon())
		log.Infoln("PII scanning enabled")
	}

	var storageSvc *storage.Service
	if config.StorageDir != "" {
		storageAddon, err := addon.NewStorageAddon(config.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to init storage: %w", err)
		}
		p.AddAddon(storageAddon)
		storageSvc = storageAddon.Service
		defer storageAddon.Close()
		log.Infof("Flow storage enabled in: %s", config.StorageDir)
	}

	if config.ScanTech {
		p.AddAddon(addon.NewWappalyzerAddon(storageSvc))
		log.Infoln("Technology scanning enabled")
	}

	return p.Start()
}
