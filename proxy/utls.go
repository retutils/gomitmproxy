package proxy

import (
	"crypto/tls"
	"net"
	"strings"

	"github.com/retutils/gomitmproxy/internal/helper"
	utls "github.com/refraction-networking/utls"
)

// NewUtlsConn creates and configures a utls.UConn based on the proxy options and client hello info.
// It handles standard fingerprints, "client" mirroring, and saved profiles.
func NewUtlsConn(conn net.Conn, opts *Options, clientHello *tls.ClientHelloInfo) (*utls.UConn, error) {
	uConfig := &utls.Config{
		InsecureSkipVerify: opts.SslInsecure,
		KeyLogWriter:       helper.GetTlsKeyLogWriter(),
		ServerName:         clientHello.ServerName,
		NextProtos:         clientHello.SupportedProtos,
		CipherSuites:       clientHello.CipherSuites,
	}

	if len(clientHello.SupportedVersions) > 0 {
		min := clientHello.SupportedVersions[0]
		max := clientHello.SupportedVersions[0]
		for _, v := range clientHello.SupportedVersions {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		uConfig.MinVersion = min
		uConfig.MaxVersion = max
	}

	fpName := opts.TlsFingerprint
	id, isStandard := getClientHelloID(fpName)
	var spec *utls.ClientHelloSpec

	if !isStandard {
		if strings.ToLower(fpName) == "client" {
			uConfig.CipherSuites = nil // Clear defaults to use client's
			id = utls.HelloCustom
			spec = mirroredClientHelloSpec(clientHello)
		} else {
			// Check for saved profile
			fp, err := LoadFingerprint(fpName)
			if err == nil && fp != nil {
				uConfig.CipherSuites = nil
				id = utls.HelloCustom
				spec = fp.ToSpec()
				ensureSNI(spec, clientHello.ServerName)
			} else {
				// Fallback to Chrome if profile not found
				id = utls.HelloChrome_Auto
			}
		}
	}

	uConn := utls.UClient(conn, uConfig, id)

	if id == utls.HelloCustom && spec != nil {
		if err := uConn.ApplyPreset(spec); err != nil {
			return nil, err
		}
	}

	return uConn, nil
}

func getClientHelloID(name string) (utls.ClientHelloID, bool) {
	switch strings.ToLower(name) {
	case "chrome":
		return utls.HelloChrome_Auto, true
	case "firefox":
		return utls.HelloFirefox_Auto, true
	case "ios":
		return utls.HelloIOS_Auto, true
	case "android":
		return utls.HelloAndroid_11_OkHttp, true
	case "edge":
		return utls.HelloEdge_Auto, true
	case "safari":
		return utls.HelloSafari_Auto, true
	case "360":
		return utls.Hello360_Auto, true
	case "qq":
		return utls.HelloQQ_Auto, true
	case "random":
		return utls.HelloRandomized, true
	default:
		return utls.HelloCustom, false
	}
}

func ensureSNI(spec *utls.ClientHelloSpec, serverName string) {
	hasSNI := false
	for _, ext := range spec.Extensions {
		if _, ok := ext.(*utls.SNIExtension); ok {
			hasSNI = true
			break
		}
	}
	if !hasSNI {
		spec.Extensions = append(spec.Extensions, &utls.SNIExtension{ServerName: serverName})
	}
}

func mirroredClientHelloSpec(info *tls.ClientHelloInfo) *utls.ClientHelloSpec {
	spec := &utls.ClientHelloSpec{}

	spec.CipherSuites = make([]uint16, len(info.CipherSuites))
	copy(spec.CipherSuites, info.CipherSuites)
	spec.CompressionMethods = []uint8{0}

	extensions := []utls.TLSExtension{}
	if info.ServerName != "" {
		extensions = append(extensions, &utls.SNIExtension{ServerName: info.ServerName})
	}

	if len(info.SupportedCurves) > 0 {
		curves := make([]utls.CurveID, len(info.SupportedCurves))
		for i, c := range info.SupportedCurves {
			curves[i] = utls.CurveID(c)
		}
		extensions = append(extensions, &utls.SupportedCurvesExtension{Curves: curves})
		
		keyShares := []utls.KeyShare{}
		for _, curve := range info.SupportedCurves {
			keyShares = append(keyShares, utls.KeyShare{Group: utls.CurveID(curve)}) // Empty Data triggers generation
		}
		extensions = append(extensions, &utls.KeyShareExtension{KeyShares: keyShares})
	}

	if len(info.SupportedPoints) > 0 {
		extensions = append(extensions, &utls.SupportedPointsExtension{SupportedPoints: info.SupportedPoints})
	}

	if len(info.SignatureSchemes) > 0 {
		algos := make([]utls.SignatureScheme, len(info.SignatureSchemes))
		for i, s := range info.SignatureSchemes {
			algos[i] = utls.SignatureScheme(s)
		}
		extensions = append(extensions, &utls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: algos})
	}

	if len(info.SupportedProtos) > 0 {
		extensions = append(extensions, &utls.ALPNExtension{AlpnProtocols: info.SupportedProtos})
	}

	if len(info.SupportedVersions) > 0 {
		versions := make([]uint16, len(info.SupportedVersions))
		copy(versions, info.SupportedVersions)
		extensions = append(extensions, &utls.SupportedVersionsExtension{Versions: versions})
	}

	spec.Extensions = extensions
	return spec
}

// Helper to convert utls state to standard tls state
func UtlsStateToTlsState(state utls.ConnectionState) *tls.ConnectionState {
	return &tls.ConnectionState{
		Version:                     state.Version,
		HandshakeComplete:           state.HandshakeComplete,
		DidResume:                   state.DidResume,
		CipherSuite:                 state.CipherSuite,
		NegotiatedProtocol:          state.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  state.NegotiatedProtocolIsMutual,
		ServerName:                  state.ServerName,
		PeerCertificates:            state.PeerCertificates,
		VerifiedChains:              state.VerifiedChains,
		SignedCertificateTimestamps: state.SignedCertificateTimestamps,
		OCSPResponse:                state.OCSPResponse,
		TLSUnique:                   state.TLSUnique,
	}
}
