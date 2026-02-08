package proxy

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"

	utls "github.com/refraction-networking/utls"
)

type Fingerprint struct {
	Name             string   `json:"name"`
	CipherSuites     []uint16 `json:"cipher_suites"`
	SupportedVersions []uint16 `json:"supported_versions"`
	SupportedCurves  []uint16 `json:"supported_curves"`
	SupportedPoints  []uint8  `json:"supported_points"`
	SignatureSchemes []uint16 `json:"signature_schemes"`
	ALPNProtocols    []string `json:"alpn_protocols"`
}

// Convert tls.ClientHelloInfo to Fingerprint struct
func NewFingerprintFromClientHello(name string, info *tls.ClientHelloInfo) *Fingerprint {
	fp := &Fingerprint{
		Name:             name,
		CipherSuites:     make([]uint16, len(info.CipherSuites)),
		SupportedVersions: make([]uint16, len(info.SupportedVersions)),
		SupportedCurves:  make([]uint16, len(info.SupportedCurves)),
		SupportedPoints:  make([]uint8, len(info.SupportedPoints)),
		SignatureSchemes: make([]uint16, len(info.SignatureSchemes)),
		ALPNProtocols:    make([]string, len(info.SupportedProtos)),
	}
	copy(fp.CipherSuites, info.CipherSuites)
	copy(fp.SupportedVersions, info.SupportedVersions)
	for i, c := range info.SupportedCurves {
		fp.SupportedCurves[i] = uint16(c)
	}
	copy(fp.SupportedPoints, info.SupportedPoints)
	for i, s := range info.SignatureSchemes {
		fp.SignatureSchemes[i] = uint16(s)
	}
	copy(fp.ALPNProtocols, info.SupportedProtos)
	return fp
}

// Convert Fingerprint to utls.ClientHelloSpec
func (fp *Fingerprint) ToSpec() *utls.ClientHelloSpec {
	spec := &utls.ClientHelloSpec{}

	spec.CipherSuites = make([]uint16, len(fp.CipherSuites))
	copy(spec.CipherSuites, fp.CipherSuites)
	
	spec.CompressionMethods = []uint8{0}

	extensions := []utls.TLSExtension{}
	
	// SNI will be added dynamically per request based on target host, not from fingerprint

	if len(fp.SupportedCurves) > 0 {
		curves := make([]utls.CurveID, len(fp.SupportedCurves))
		for i, c := range fp.SupportedCurves {
			curves[i] = utls.CurveID(c)
		}
		extensions = append(extensions, &utls.SupportedCurvesExtension{Curves: curves})
		
		// KeyShare
		keyShares := []utls.KeyShare{}
		for _, curve := range fp.SupportedCurves {
			keyShares = append(keyShares, utls.KeyShare{Group: utls.CurveID(curve)})
		}
		extensions = append(extensions, &utls.KeyShareExtension{KeyShares: keyShares})
	}

	if len(fp.SupportedPoints) > 0 {
		extensions = append(extensions, &utls.SupportedPointsExtension{SupportedPoints: fp.SupportedPoints})
	}

	if len(fp.SignatureSchemes) > 0 {
		algos := make([]utls.SignatureScheme, len(fp.SignatureSchemes))
		for i, s := range fp.SignatureSchemes {
			algos[i] = utls.SignatureScheme(s)
		}
		extensions = append(extensions, &utls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: algos})
	}

	if len(fp.ALPNProtocols) > 0 {
		extensions = append(extensions, &utls.ALPNExtension{AlpnProtocols: fp.ALPNProtocols})
	}

	if len(fp.SupportedVersions) > 0 {
		versions := make([]uint16, len(fp.SupportedVersions))
		copy(versions, fp.SupportedVersions)
		extensions = append(extensions, &utls.SupportedVersionsExtension{Versions: versions})
	}

	spec.Extensions = extensions
	return spec
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func SaveFingerprint(name string, fp *Fingerprint) error {
	var path string
	if filepath.IsAbs(name) || filepath.Dir(name) != "." {
		path = name
		if filepath.Ext(path) == "" {
			path += ".json"
		}
	} else {
		dir := GetFingerprintDir()
		if err := ensureDir(dir); err != nil {
			return err
		}
		path = filepath.Join(dir, name+".json")
	}
	
	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadFingerprint(name string) (*Fingerprint, error) {
	// Try direct path first
	if _, err := os.Stat(name); err == nil {
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var fp Fingerprint
		if err := json.Unmarshal(data, &fp); err != nil {
			return nil, err
		}
		return &fp, nil
	}

	// Try default directory
	dir := GetFingerprintDir()
	path := filepath.Join(dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Try without extension
		path = filepath.Join(dir, name)
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	
	var fp Fingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return nil, err
	}
	return &fp, nil
}

func ListFingerprints() ([]string, error) {
	dir := GetFingerprintDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name()[:len(entry.Name())-5])
		}
	}
	return names, nil
}

func GetFingerprintDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "fingerprints"
	}
	return filepath.Join(home, ".mitmproxy", "fingerprints")
}
