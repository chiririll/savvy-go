package sso

import (
	"bytes"
	"compress/flate"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"io"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

// samlResponse is a Response around an Assertion for nameID, signed by key.
func samlResponse(
	t *testing.T,
	key *rsa.PrivateKey,
	cert *x509.Certificate,
	nameID, inResponseTo string,
	attrs map[string][]string,
) *etree.Element {
	t.Helper()
	return samlResponseWith(t, key, cert, nameID, inResponseTo, attrs, validSAML())
}

// samlOpts are the parts of an assertion the service must hold the IdP to.
type samlOpts struct {
	issuer       string
	audience     string // empty: no AudienceRestriction at all
	notBefore    string // empty: omitted
	notOnOrAfter string // empty: no Conditions window at all
}

func validSAML() samlOpts {
	now := time.Now().UTC()
	return samlOpts{
		issuer:       "https://idp.example/saml",
		audience:     "https://app.test/api/auth/sso/corp/metadata",
		notBefore:    now.Add(-time.Minute).Format(time.RFC3339),
		notOnOrAfter: now.Add(5 * time.Minute).Format(time.RFC3339),
	}
}

func samlResponseWith(
	t *testing.T,
	key *rsa.PrivateKey,
	cert *x509.Certificate,
	nameID, inResponseTo string,
	attrs map[string][]string,
	o samlOpts,
) *etree.Element {
	t.Helper()
	assertion := etree.NewElement("Assertion")
	assertion.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:assertion")
	assertion.CreateAttr("ID", "_assert1")
	assertion.CreateAttr("Version", "2.0")
	assertion.CreateAttr("IssueInstant", time.Now().UTC().Format(time.RFC3339))
	assertion.CreateElement("Issuer").SetText(o.issuer)
	if o.notOnOrAfter != "" || o.audience != "" {
		cond := assertion.CreateElement("Conditions")
		if o.notBefore != "" {
			cond.CreateAttr("NotBefore", o.notBefore)
		}
		if o.notOnOrAfter != "" {
			cond.CreateAttr("NotOnOrAfter", o.notOnOrAfter)
		}
		if o.audience != "" {
			cond.CreateElement("AudienceRestriction").CreateElement("Audience").SetText(o.audience)
		}
	}
	subject := assertion.CreateElement("Subject")
	subject.CreateElement("NameID").SetText(nameID)
	if inResponseTo != "" {
		// Where a real IdP says which request this answers: inside the signed part.
		subject.CreateElement("SubjectConfirmation").
			CreateElement("SubjectConfirmationData").
			CreateAttr("InResponseTo", inResponseTo)
	}
	if len(attrs) > 0 {
		stmt := assertion.CreateElement("AttributeStatement")
		for name, values := range attrs {
			a := stmt.CreateElement("Attribute")
			a.CreateAttr("Name", name)
			for _, v := range values {
				a.CreateElement("AttributeValue").SetText(v)
			}
		}
	}
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
	if inResponseTo != "" {
		resp.CreateAttr("InResponseTo", inResponseTo)
	}
	resp.AddChild(signed)
	return resp
}

func encode(t *testing.T, resp *etree.Element) string {
	t.Helper()
	doc := etree.NewDocument()
	doc.SetRoot(resp)
	s, err := doc.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func inflate(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	r := flate.NewReader(bytes.NewReader(raw))
	defer r.Close()
	out, err := io.ReadAll(r)
	return string(out), err
}
