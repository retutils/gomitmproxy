package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/retutils/gomitmproxy/internal/helper"
	log "github.com/sirupsen/logrus"
)

func loadConfigFromFile(filename string) (*Config, error) {
	var config Config
	if err := helper.NewStructFromFile(filename, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func loadConfigFromCli() *Config {
	config := new(Config)
	fs := flag.NewFlagSet("go-mitmproxy", flag.ExitOnError)
	defineFlags(fs, config)
	fs.Parse(os.Args[1:])
	return config
}

func defineFlags(fs *flag.FlagSet, config *Config) {
	fs.BoolVar(&config.version, "version", config.version, "show go-mitmproxy version")
	fs.StringVar(&config.Addr, "addr", config.Addr, "proxy listen addr")
	if config.Addr == "" {
		config.Addr = ":9080"
	}
	fs.StringVar(&config.WebAddr, "web_addr", config.WebAddr, "web interface listen addr")
	if config.WebAddr == "" {
		config.WebAddr = ":9081"
	}
	fs.BoolVar(&config.SslInsecure, "ssl_insecure", config.SslInsecure, "not verify upstream server SSL/TLS certificates.")
	fs.Var((*arrayValue)(&config.IgnoreHosts), "ignore_hosts", "a list of ignore hosts")
	fs.Var((*arrayValue)(&config.AllowHosts), "allow_hosts", "a list of allow hosts")
	fs.StringVar(&config.CertPath, "cert_path", config.CertPath, "path of generate cert files")
	fs.IntVar(&config.Debug, "debug", config.Debug, "debug mode: 1 - print debug log, 2 - show debug from")
	fs.StringVar(&config.Dump, "dump", config.Dump, "dump filename")
	fs.IntVar(&config.DumpLevel, "dump_level", config.DumpLevel, "dump level: 0 - header, 1 - header + body")
	fs.StringVar(&config.Upstream, "upstream", config.Upstream, "upstream proxy")
	fs.BoolVar(&config.UpstreamCert, "upstream_cert", config.UpstreamCert, "connect to upstream server to look up certificate details")
	fs.StringVar(&config.MapRemote, "map_remote", config.MapRemote, "map remote config filename")
	fs.StringVar(&config.MapLocal, "map_local", config.MapLocal, "map local config filename")
	fs.StringVar(&config.LogFile, "log_file", config.LogFile, "log file path")
	fs.StringVar(&config.filename, "f", config.filename, "read config from the filename")

	fs.StringVar(&config.ProxyAuth, "proxyauth", config.ProxyAuth, `enable proxy authentication. Format: "username:pass", "user1:pass1|user2:pass2","any" to accept any user/pass combination`)
	fs.StringVar(&config.TlsFingerprint, "tls_fingerprint", config.TlsFingerprint, "TLS fingerprint to emulate (chrome, firefox, ios, android, edge, 360, qq, random)")
	fs.StringVar(&config.FingerprintSave, "fingerprint_save", config.FingerprintSave, "Save client fingerprint to file with specified name")
	fs.BoolVar(&config.FingerprintList, "fingerprint_list", config.FingerprintList, "List saved client fingerprints")
	fs.BoolVar(&config.ScanPII, "scan_pii", config.ScanPII, "Enable PII and confidential information scanning")
	fs.BoolVar(&config.ScanTech, "scan_tech", config.ScanTech, "Enable technology and framework scanning (Wappalyzer)")
	fs.StringVar(&config.StorageDir, "storage_dir", config.StorageDir, "Directory to store captured flows (DuckDB + Bleve)")
	fs.StringVar(&config.Search, "search", config.Search, "Search query for stored flows (requires -storage_dir)")
	fs.Var((*arrayValue)(&config.DnsResolvers), "dns_resolvers", "a list of DNS resolvers")
	fs.IntVar(&config.DnsRetries, "dns_retries", config.DnsRetries, "number of DNS resolution retries")
	if config.DnsRetries == 0 {
		config.DnsRetries = 2
	}
}

func mergeConfigs(fileConfig, cliConfig *Config) *Config {
	config := new(Config)
	*config = *fileConfig
	if cliConfig.Addr != "" {
		config.Addr = cliConfig.Addr
	}
	if cliConfig.WebAddr != "" {
		config.WebAddr = cliConfig.WebAddr
	}
	if cliConfig.SslInsecure {
		config.SslInsecure = cliConfig.SslInsecure
	}
	if len(cliConfig.IgnoreHosts) > 0 {
		config.IgnoreHosts = cliConfig.IgnoreHosts
	}
	if len(cliConfig.AllowHosts) > 0 {
		config.AllowHosts = cliConfig.AllowHosts
	}
	if cliConfig.CertPath != "" {
		config.CertPath = cliConfig.CertPath
	}
	if cliConfig.Debug != 0 {
		config.Debug = cliConfig.Debug
	}
	if cliConfig.Dump != "" {
		config.Dump = cliConfig.Dump
	}
	if cliConfig.DumpLevel != 0 {
		config.DumpLevel = cliConfig.DumpLevel
	}
	if cliConfig.Upstream != "" {
		config.Upstream = cliConfig.Upstream
	}
	if !cliConfig.UpstreamCert {
		config.UpstreamCert = cliConfig.UpstreamCert
	}
	if cliConfig.MapRemote != "" {
		config.MapRemote = cliConfig.MapRemote
	}
	if cliConfig.MapLocal != "" {
		config.MapLocal = cliConfig.MapLocal
	}
	if cliConfig.LogFile != "" {
		config.LogFile = cliConfig.LogFile
	}
	if cliConfig.TlsFingerprint != "" {
		config.TlsFingerprint = cliConfig.TlsFingerprint
	}
	if cliConfig.FingerprintSave != "" {
		config.FingerprintSave = cliConfig.FingerprintSave
	}
	if cliConfig.FingerprintList {
		config.FingerprintList = cliConfig.FingerprintList
	}
	if cliConfig.StorageDir != "" {
		config.StorageDir = cliConfig.StorageDir
	}
	if len(cliConfig.DnsResolvers) > 0 {
		config.DnsResolvers = cliConfig.DnsResolvers
	}
	if cliConfig.DnsRetries != 0 {
		config.DnsRetries = cliConfig.DnsRetries
	}
	if cliConfig.ScanPII {
		config.ScanPII = cliConfig.ScanPII
	}
	if cliConfig.ScanTech {
		config.ScanTech = cliConfig.ScanTech
	}
	return config
}

func loadConfig() *Config {
	// 1. Initial pass to find the config file
	filename := ""
	for i, arg := range os.Args {
		if arg == "-f" && i+1 < len(os.Args) {
			filename = os.Args[i+1]
			break
		}
	}

	config := new(Config)
	if filename != "" {
		fileConfig, err := loadConfigFromFile(filename)
		if err != nil {
			log.Warnf("read config from %v error %v", filename, err)
		} else {
			config = fileConfig
			log.Infof("Loaded config from file %v: %+v", filename, config)
		}
	}

	// 2. Final pass with CLI overrides
	finalFs := flag.NewFlagSet("go-mitmproxy", flag.ExitOnError)
	defineFlags(finalFs, config)
	finalFs.Parse(os.Args[1:])

	return config
}

// arrayValue 实现了 flag.Value 接口
type arrayValue []string

func (a *arrayValue) String() string {
	return fmt.Sprint(*a)
}

func (a *arrayValue) Set(value string) error {
	*a = append(*a, value)
	return nil
}
