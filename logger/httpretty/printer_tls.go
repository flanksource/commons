package httpretty

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/flanksource/commons/logger/httpretty/internal/color"
)

func findPeerCertificate(hostname string, state *tls.ConnectionState) (cert *x509.Certificate) {
	if chains := state.VerifiedChains; chains != nil && chains[0] != nil && chains[0][0] != nil {
		return chains[0][0]
	}
	if hostname == "" && len(state.PeerCertificates) > 0 {
		return state.PeerCertificates[0]
	}
	for _, cert := range state.PeerCertificates {
		if err := cert.VerifyHostname(hostname); err == nil {
			return cert
		}
	}
	return nil
}

func (p *printer) printTLSInfo(state *tls.ConnectionState, skipVerifyChains bool) {
	if state == nil {
		return
	}
	protocol := tlsProtocolVersions[state.Version]
	if protocol == "" {
		protocol = fmt.Sprintf("%#v", state.Version)
	}
	cipher := tlsCiphers[state.CipherSuite]
	if cipher == "" {
		cipher = fmt.Sprintf("%#v", state.CipherSuite)
	}
	p.printf("* %s (%s)", p.format(color.FgBlue, protocol), p.format(color.FgBlue, cipher))
	if !skipVerifyChains && state.VerifiedChains == nil {
		p.print(" (insecure=true)")
	}
	if state.NegotiatedProtocol != "" {
		p.printf(" ALPN:%v", p.format(color.FgBlue, state.NegotiatedProtocol))
	}
}

func (p *printer) printOutgoingClientTLS(config *tls.Config) {
	if config == nil || len(config.Certificates) == 0 {
		return
	}
	if cert := config.Certificates[0].Leaf; cert != nil {
		p.printClientCertificateSimple("", cert)
	} else {
		p.print(`* Client certificate: unparsed certificate found, skipping`)
	}
}

func (p *printer) printClientCertificateSimple(hostname string, cert *x509.Certificate) {
	cn := cert.Subject.CommonName
	if cn == "" {
		cn = "unknown"
	}

	additionalSANs := len(cert.DNSNames) - 1
	if additionalSANs < 0 {
		additionalSANs = 0
	}

	issuerCN := cert.Issuer.CommonName
	if issuerCN == "" {
		issuerCN = "unknown"
	}

	now := time.Now()
	expiryDuration := cert.NotAfter.Sub(now)
	var expiryStr string
	if expiryDuration < 0 {
		expiryStr = "expired"
	} else if expiryDuration < 24*time.Hour {
		expiryStr = fmt.Sprintf("%.0fh", expiryDuration.Hours())
	} else {
		expiryStr = fmt.Sprintf("%.0fd", expiryDuration.Hours()/24)
	}

	valid := true
	if hostname != "" {
		if err := cert.VerifyHostname(hostname); err != nil {
			valid = false
		}
	}
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		valid = false
	}

	msg := fmt.Sprintf(" %s", cn)
	if additionalSANs > 0 {
		msg += fmt.Sprintf(" [+%d more]", additionalSANs)
	}
	msg += fmt.Sprintf(" (issued by: %s, expires in %s)", issuerCN, expiryStr)

	if valid {
		p.println(p.format(color.FgGreen, msg))
	} else {
		p.println(p.format(color.FgRed, msg))
	}
}

func (p *printer) printIncomingClientTLS(state *tls.ConnectionState) {
	if state == nil || len(state.PeerCertificates) == 0 {
		return
	}
	if cert := findPeerCertificate("", state); cert != nil {
		p.printCertificate("", cert, state)
	} else {
		p.println(p.format(color.FgRed, "** No valid certificate was found"))
	}
}

func (p *printer) printTLSServer(host string, state *tls.ConnectionState) {
	if state == nil {
		return
	}
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}
	if cert := findPeerCertificate(hostname, state); cert != nil {
		p.printCertificate(hostname, cert, state)
	} else {
		p.print(p.format(color.FgRed, "** No valid certificate was found"))
	}
}

func (p *printer) printCertificate(hostname string, cert *x509.Certificate, state *tls.ConnectionState) {
	cn := cert.Subject.CommonName
	if cn == "" {
		cn = "unknown"
	}

	additionalSANs := len(cert.DNSNames) - 1
	if additionalSANs < 0 {
		additionalSANs = 0
	}

	issuerCN := cert.Issuer.CommonName
	if issuerCN == "" {
		issuerCN = "unknown"
	}

	now := time.Now()
	expiryDuration := cert.NotAfter.Sub(now)
	var expiryStr string
	if expiryDuration < 0 {
		expiryStr = "expired"
	} else if expiryDuration < 24*time.Hour {
		expiryStr = fmt.Sprintf("%.0fh", expiryDuration.Hours())
	} else {
		expiryStr = fmt.Sprintf("%.0fd", expiryDuration.Hours()/24)
	}

	valid := true
	if hostname != "" {
		if err := cert.VerifyHostname(hostname); err != nil {
			valid = false
		}
	}
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		valid = false
	}

	msg := fmt.Sprintf(" %s", cn)
	if additionalSANs > 0 {
		msg += fmt.Sprintf(" [+%d more]", additionalSANs)
	}
	msg += fmt.Sprintf(" (issued by: %s, expires in %s)", issuerCN, expiryStr)

	if valid {
		p.println(p.format(color.FgGreen, msg))
	} else {
		p.println(p.format(color.FgRed, msg))
	}
}
