package sso

import (
	"crypto/x509"
	"encoding/pem"
	"strings"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

func parseX509Certs(pems ...string) []*x509.Certificate {
	var out []*x509.Certificate
	for _, raw := range pems {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.Contains(raw, "BEGIN") {
			raw = "-----BEGIN CERTIFICATE-----\n" + raw + "\n-----END CERTIFICATE-----"
		}
		rest := []byte(raw)
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				continue
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}
			out = append(out, cert)
		}
	}
	return out
}

func idpCerts(p IdentityProvider) []*x509.Certificate {
	return parseX509Certs(cfgString(p.Config, "idp_x509_cert"), cfgString(p.Config, "idp_x509_cert_standby"))
}

func hasSignature(el *etree.Element) bool {
	if el == nil {
		return false
	}
	for _, c := range el.ChildElements() {
		if local(c.Tag) == "Signature" {
			return true
		}
	}
	return false
}

func findSAML(el *etree.Element, name string) *etree.Element {
	if el == nil {
		return nil
	}
	if local(el.Tag) == name {
		return el
	}
	for _, c := range el.ChildElements() {
		if found := findSAML(c, name); found != nil {
			return found
		}
	}
	return nil
}

// VerifySAMLXML checks an enveloped XML signature on the Response or Assertion
// against the IdP certificate(s). Unsigned documents are rejected.
func VerifySAMLXML(xmlBytes []byte, certs []*x509.Certificate) error {
	_, err := verifiedSAML(xmlBytes, certs)
	return err
}

// verifiedSAML is the part of the document the signature vouches for: the Response
// or the Assertion that carries a valid signature. Only that part may be read.
// Whatever else the document holds was put there by whoever sent it, and an
// unsigned element beside a signed one would otherwise decide who logs in.
func verifiedSAML(xmlBytes []byte, certs []*x509.Certificate) (*etree.Element, error) {
	if len(certs) == 0 {
		return nil, ssoErr("saml_invalid", "The SAML identity provider has no certificate.", 401)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil || doc.Root() == nil {
		return nil, ssoErr("saml_invalid", "The SAML response is invalid.", 401)
	}
	store := &dsig.MemoryX509CertificateStore{Roots: certs}
	ctx := dsig.NewDefaultValidationContext(store)
	root := doc.Root()
	targets := []*etree.Element{root, findSAML(root, "Assertion")}
	var last error
	for _, el := range targets {
		if !hasSignature(el) {
			continue
		}
		signed, err := ctx.Validate(el)
		if err == nil {
			return signed, nil
		}
		last = err
	}
	if last != nil {
		return nil, ssoErr("saml_invalid", "The SAML signature is invalid.", 401)
	}
	return nil, ssoErr("saml_invalid", "The SAML assertion is unsigned.", 401)
}
