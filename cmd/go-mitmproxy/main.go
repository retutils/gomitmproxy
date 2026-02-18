package main

import (
	"fmt"
	rawLog "log"
	"net/http"
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
	version bool // show go-mitmproxy version

	Addr         string   // proxy listen addr
	WebAddr      string   // web interface listen addr
	SslInsecure  bool     // not verify upstream server SSL/TLS certificates.
	IgnoreHosts  []string // a list of ignore hosts
	AllowHosts   []string // a list of allow hosts
	CertPath     string   // path of generate cert files
	Debug        int      // debug mode: 1 - print debug log, 2 - show debug from
	Dump         string   // dump filename
	DumpLevel    int      // dump level: 0 - header, 1 - header + body
	Upstream     string   // upstream proxy
	UpstreamCert bool     // Connect to upstream server to look up certificate details. Default: True
	MapRemote    string   // map remote config filename
	MapLocal     string   // map local config filename
	LogFile      string   // log file path

	filename string // read config from the filename

	ProxyAuth       string // Require proxy authentication
	TlsFingerprint  string // TLS fingerprint to emulate (chrome, firefox, ios, or random)
	FingerprintSave string // Save decoding client hello to file
	FingerprintList bool   // List saved fingerprints
	StorageDir      string // Directory to store captured flows (DuckDB + Bleve)
	Search          string // Search query for stored flows
	ScanPII         bool   // Enable PII scanning (regex + AC)
}

func main() {
	config := loadConfig()

	if config.FingerprintList {
		names, err := proxy.ListFingerprints()
		if err != nil {
			log.Fatal(err)
		}
		if len(names) == 0 {
			fmt.Println("No saved fingerprints found.")
		} else {
			fmt.Println("Saved fingerprints:")
			for _, name := range names {
				fmt.Printf(" - %s\n", name)
			}
		}
		os.Exit(0)
	}

	if config.Search != "" {
		if config.StorageDir == "" {
			fmt.Println("-storage_dir is required for search")
			os.Exit(1)
		}

		// Initialize service in read-only mode if possible, but NewService opens both
		// For CLI search, we just need to init, search, print, exit
		svc, err := storage.NewService(config.StorageDir)
		if err != nil {
			log.Fatalf("Failed to open storage: %v", err)
		}
		defer svc.Close()

		results, err := svc.Search(config.Search)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
		} else {
			fmt.Printf("Found %d results:\n", len(results))
			for _, entry := range results {
				fmt.Printf("[%s] %s %s (Status: %d)\n", entry.ID, entry.Method, entry.URL, entry.StatusCode)
				// Maybe print snippets or headers if verbose?
				// For now just ID and URL is proof of concept
			}
		}
		os.Exit(0)
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
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	if config.version {
		fmt.Println("go-mitmproxy: " + p.Version)
		os.Exit(0)
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
			log.Fatal(err)
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

	if config.StorageDir != "" {
		storageAddon, err := addon.NewStorageAddon(config.StorageDir)
		if err != nil {
			log.Fatalf("failed to init storage: %v", err)
		}
		p.AddAddon(storageAddon)
		defer storageAddon.Close()
		log.Infof("Flow storage enabled in: %s", config.StorageDir)
	}

	log.Fatal(p.Start())
}
