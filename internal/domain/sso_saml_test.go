package domain

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

func TestVerifySAMLXMLSignedAndTampered(t *testing.T) {
	key, cert, pemCert := selfSigned(t)
	xml := signedAssertion(t, key, cert, "jane@test.com")
	certs := parseX509Certs(pemCert)
	if err := VerifySAMLXML([]byte(xml), certs); err != nil {
		t.Fatalf("valid signature: %v", err)
	}
	tampered := strings.Replace(xml, "jane@test.com", "evil@test.com", 1)
	if err := VerifySAMLXML([]byte(tampered), certs); err == nil {
		t.Fatal("tampered assertion accepted")
	}
	unsigned := `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"><saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"><saml:NameID>x</saml:NameID></saml:Assertion></samlp:Response>`
	if err := VerifySAMLXML([]byte(unsigned), certs); err == nil {
		t.Fatal("unsigned accepted")
	}
	nameID, _, _, ok := parseSAMLResponse(xml)
	if !ok || nameID != "jane@test.com" {
		t.Fatalf("parse %q %v", nameID, ok)
	}
}

func selfSigned(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "idp.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pemCert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return key, cert, pemCert
}

func signedAssertion(t *testing.T, key *rsa.PrivateKey, cert *x509.Certificate, nameID string) string {
	t.Helper()
	assertion := etree.NewElement("Assertion")
	assertion.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:assertion")
	assertion.CreateAttr("ID", "_assert1")
	assertion.CreateAttr("Version", "2.0")
	assertion.CreateAttr("IssueInstant", time.Now().UTC().Format(time.RFC3339))
	issuer := assertion.CreateElement("Issuer")
	issuer.SetText("https://idp.example/saml")
	nid := assertion.CreateElement("NameID")
	nid.SetText(nameID)
	ctx, err := dsig.NewSigningContext(key, [][]byte{cert.Raw})
	if err != nil {
		t.Fatal(err)
	}
	signed, err := ctx.SignEnveloped(assertion)
	if err != nil {
		t.Fatal(err)
	}
	resp := etree.NewElement("Response")
	resp.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:protocol")
	resp.CreateAttr("ID", "_resp1")
	resp.AddChild(signed)
	doc := etree.NewDocument()
	doc.SetRoot(resp)
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
