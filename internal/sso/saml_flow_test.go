package sso

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

const (
	emailClaim = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
	nameClaim  = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
	groupClaim = "http://schemas.xmlsoap.org/claims/Group"
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

type samlEnv struct {
	ctx  context.Context
	svc  Service
	p    *IdentityProvider
	key  *rsa.PrivateKey
	cert *x509.Certificate
}

func newSAMLEnv(t *testing.T) *samlEnv {
	t.Helper()
	key, cert, pemCert := selfSigned(t)
	d := dbtest.Server(t)
	svc := Service{
		DB: d, Users: auth.Users{DB: d}, Settings: settings.Store{DB: d},
		Cfg: appconfig.Config{}.WithAppURL("https://app.test"),
	}
	ctx := context.Background()
	p, err := svc.Create(ctx, IdPWrite{
		Name: "Corp", Slug: "corp", Preset: "custom_saml", Enabled: ptr(true),
		Fields: map[string]any{
			"idp_entity_id": "https://idp.example/saml",
			"idp_sso_url":   "https://idp.example/sso?x=1",
			"idp_x509_cert": pemCert,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &samlEnv{ctx: ctx, svc: svc, p: p, key: key, cert: cert}
}

// start runs the redirect step: it returns the RelayState the IdP would echo back
// and the id of the request it was asked to answer.
func (e *samlEnv) start(t *testing.T) (relayState, requestID string) {
	t.Helper()
	loc, err := e.svc.AuthorizeURL(e.ctx, *e.p, "/home")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(loc)
	relayState = u.Query().Get("RelayState")
	req, err := inflate(u.Query().Get("SAMLRequest"))
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(req, ` ID="`)
	if !ok {
		t.Fatalf("no request id in %s", req)
	}
	requestID, _, _ = strings.Cut(after, `"`)
	return relayState, requestID
}

func TestSAMLRedirectAsksTheIdPForALogin(t *testing.T) {
	e := newSAMLEnv(t)
	loc, err := e.svc.AuthorizeURL(e.ctx, *e.p, "//evil.example")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(loc)
	if u.Host != "idp.example" || u.Path != "/sso" || u.Query().Get("x") != "1" {
		t.Errorf("the IdP URL and its own query survive: %s", loc)
	}
	req, err := inflate(u.Query().Get("SAMLRequest"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`AssertionConsumerServiceURL="https://app.test/api/auth/sso/corp/acs"`,
		"<saml:Issuer>https://app.test/api/auth/sso/corp/metadata</saml:Issuer>",
		`Destination="https://idp.example/sso?x=1"`,
	} {
		if !strings.Contains(req, want) {
			t.Errorf("request lacks %s:\n%s", want, req)
		}
	}
	st, _ := e.svc.ConsumeState(e.ctx, u.Query().Get("RelayState"))
	if st == nil || st.SAMLRequestID == "" || !strings.Contains(req, `ID="`+st.SAMLRequestID+`"`) {
		t.Fatalf("state %+v", st)
	}
	if st.RedirectAfter != "" {
		t.Errorf("an off-site redirect target is dropped: %q", st.RedirectAfter)
	}
}

func TestSAMLRedirectNeedsAnIdPURL(t *testing.T) {
	e := newSAMLEnv(t)
	p := *e.p
	p.Config = map[string]any{}
	if _, err := e.svc.AuthorizeURL(e.ctx, p, ""); ssoCode(err) != "missing_field" {
		t.Errorf("error %q", ssoCode(err))
	}
}

func TestSAMLLoginReadsTheSignedAssertion(t *testing.T) {
	e := newSAMLEnv(t)
	state, reqID := e.start(t)
	resp := samlResponse(t, e.key, e.cert, "ann@corp.test", reqID, map[string][]string{
		emailClaim: {"ann@corp.test"}, nameClaim: {"Ann"}, groupClaim: {"staff", "admins"},
	})
	id, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, resp), state)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "ann@corp.test" || id.Email != "ann@corp.test" || !id.EmailVerified || id.Name != "Ann" ||
		len(id.Groups) != 2 {
		t.Errorf("identity %+v", id)
	}
	if _, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, resp), state); ssoCode(err) != "invalid_state" {
		t.Errorf("a response replayed on the same state: %q", ssoCode(err))
	}
}

func TestSAMLLoginRefusesForeignResponses(t *testing.T) {
	e := newSAMLEnv(t)
	otherKey, otherCert, _ := selfSigned(t)

	tests := []struct {
		name string
		resp func(reqID string) string
	}{
		{"signed by another key", func(reqID string) string {
			return encode(t, samlResponse(t, otherKey, otherCert, "ann@corp.test", reqID, nil))
		}},
		{"an answer to another request", func(string) string {
			return encode(t, samlResponse(t, e.key, e.cert, "ann@corp.test", "_someone_elses", nil))
		}},
		{"not base64", func(string) string { return "!!!" }},
		{"empty", func(string) string { return "" }},
		{"not XML", func(string) string { return base64.StdEncoding.EncodeToString([]byte("hello")) }},
	}
	for _, tc := range tests {
		state, reqID := e.start(t)
		if _, err := e.svc.HandleACS(e.ctx, *e.p, tc.resp(reqID), state); ssoCode(err) != "saml_invalid" {
			t.Errorf("%s: %q, want saml_invalid", tc.name, ssoCode(err))
		}
	}

	state, reqID := e.start(t)
	other := *e.p
	other.ID += 50
	if _, err := e.svc.HandleACS(e.ctx, other, encode(t, samlResponse(t, e.key, e.cert, "a", reqID, nil)), state); ssoCode(
		err,
	) != "invalid_state" {
		t.Errorf("another provider's state: %q", ssoCode(err))
	}
}

func TestSAMLCertificateRotation(t *testing.T) {
	e := newSAMLEnv(t)
	newKey, newCert, newPEM := selfSigned(t)
	p := *e.p
	p.Config = map[string]any{
		"idp_entity_id":         p.Config["idp_entity_id"],
		"idp_x509_cert":         p.Config["idp_x509_cert"],
		"idp_x509_cert_standby": newPEM,
	}
	state, reqID := e.start(t)
	if _, err := e.svc.HandleACS(e.ctx, p, encode(t, samlResponse(t, newKey, newCert, "ann@corp.test", reqID, nil)), state); err != nil {
		t.Errorf("the standby certificate must verify during a rotation: %v", err)
	}
	p.Config = map[string]any{}
	state, reqID = e.start(t)
	if _, err := e.svc.HandleACS(e.ctx, p, encode(t, samlResponse(t, e.key, e.cert, "ann@corp.test", reqID, nil)), state); ssoCode(
		err,
	) != "saml_invalid" {
		t.Errorf("without a certificate nothing verifies: %q", ssoCode(err))
	}
}

// The signature covers one assertion. Anything added next to it is not vouched for,
// so it must never decide who logs in.
func TestSAMLUnsignedAssertionBesideSignedOneIsIgnored(t *testing.T) {
	e := newSAMLEnv(t)
	state, reqID := e.start(t)
	resp := samlResponse(t, e.key, e.cert, "ann@corp.test", reqID, map[string][]string{emailClaim: {"ann@corp.test"}})

	forged := etree.NewElement("Assertion")
	forged.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:assertion")
	forged.CreateElement("NameID").SetText("root@corp.test")
	a := forged.CreateElement("AttributeStatement").CreateElement("Attribute")
	a.CreateAttr("Name", emailClaim)
	a.CreateElement("AttributeValue").SetText("root@corp.test")
	resp.AddChild(forged)

	id, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, resp), state)
	if err == nil && (id.Subject == "root@corp.test" || id.Email == "root@corp.test") {
		t.Fatalf("signature wrapping: the unsigned assertion decided the identity: %+v", id)
	}
}

// InResponseTo on the Response is outside the signature, so whoever replays a
// captured response can rewrite it. Only the signed one counts.
func TestSAMLReplayWithARewrittenRequestIdIsRefused(t *testing.T) {
	e := newSAMLEnv(t)
	_, oldReq := e.start(t)
	captured := samlResponse(t, e.key, e.cert, "ann@corp.test", oldReq, nil)

	state, newReq := e.start(t)
	captured.RemoveAttr("InResponseTo")
	captured.CreateAttr("InResponseTo", newReq)
	if _, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, captured), state); ssoCode(err) != "saml_invalid" {
		t.Errorf("a response to an old request was accepted for a new one: %q", ssoCode(err))
	}
}

// Only logins we started are accepted, so a captured response that names no
// request cannot be replayed against a fresh one.
func TestSAMLResponseMustAnswerOurRequest(t *testing.T) {
	e := newSAMLEnv(t)
	state, _ := e.start(t)
	if _, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, samlResponse(t, e.key, e.cert, "ann@corp.test", "", nil)), state); ssoCode(
		err,
	) != "saml_invalid" {
		t.Errorf("a response without InResponseTo: %q", ssoCode(err))
	}
}

func TestSAMLAssertionConditionsAreEnforced(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name   string
		mutate func(*samlOpts)
	}{
		{"another issuer", func(o *samlOpts) { o.issuer = "https://evil.example/saml" }},
		{"another audience", func(o *samlOpts) { o.audience = "https://other-app.test/metadata" }},
		{"no audience", func(o *samlOpts) { o.audience = "" }},
		{"expired", func(o *samlOpts) { o.notOnOrAfter = now.Add(-10 * time.Minute).Format(time.RFC3339) }},
		{"not valid yet", func(o *samlOpts) { o.notBefore = now.Add(10 * time.Minute).Format(time.RFC3339) }},
		{"no validity window", func(o *samlOpts) { o.notOnOrAfter = "" }},
		{"unparsable time", func(o *samlOpts) { o.notOnOrAfter = "tomorrow" }},
	}
	for _, tc := range tests {
		e := newSAMLEnv(t)
		state, reqID := e.start(t)
		o := validSAML()
		tc.mutate(&o)
		resp := samlResponseWith(t, e.key, e.cert, "ann@corp.test", reqID, nil, o)
		if _, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, resp), state); ssoCode(err) != "saml_invalid" {
			t.Errorf("%s: %q, want saml_invalid", tc.name, ssoCode(err))
		}
	}
}

func TestSAMLClockSkewIsTolerated(t *testing.T) {
	e := newSAMLEnv(t)
	state, reqID := e.start(t)
	o := validSAML()
	o.notBefore = time.Now().UTC().Add(30 * time.Second).Format(time.RFC3339)
	if _, err := e.svc.HandleACS(e.ctx, *e.p, encode(t, samlResponseWith(t, e.key, e.cert, "ann@corp.test", reqID, nil, o)), state); err != nil {
		t.Errorf("a few seconds of clock drift between us and the IdP: %v", err)
	}
}

func TestSAMLIdentityMapping(t *testing.T) {
	p := IdentityProvider{Preset: "custom_saml"}
	id := samlIdentity(p, "name-id", map[string][]string{emailClaim: {"a@b.c", "second"}, "single": {"x"}})
	if id.Subject != "name-id" || id.Email != "a@b.c" || !id.EmailVerified {
		t.Errorf("identity %+v", id)
	}
	if id.Raw["single"] != "x" || id.Raw["nameid"] != "name-id" {
		t.Errorf("raw %v", id.Raw)
	}
	if multi, _ := id.Raw[emailClaim].([]string); len(multi) != 2 {
		t.Errorf("a multi-valued attribute stays a list: %v", id.Raw[emailClaim])
	}
	bare := samlIdentity(p, "only-name-id", nil)
	if bare.Subject != "only-name-id" || bare.Email != "" || bare.EmailVerified {
		t.Errorf("a response without attributes %+v", bare)
	}
	mapped := samlIdentity(
		IdentityProvider{ClaimMappings: map[string]any{"subject": "uid", "email": "mail"}},
		"nid",
		map[string][]string{"uid": {"u-7"}, "mail": {"m@x.y"}},
	)
	if mapped.Subject != "u-7" || mapped.Email != "m@x.y" {
		t.Errorf("mapped %+v", mapped)
	}
}

func TestParseSAMLResponseHandlesPrefixesAndJunk(t *testing.T) {
	xml := `<samlp:Response xmlns:samlp="p" xmlns:saml="a" InResponseTo="_r1"><saml:Assertion><saml:Subject><saml:NameID> bob </saml:NameID></saml:Subject>` +
		`<saml:AttributeStatement><saml:Attribute Name="g"><saml:AttributeValue> a </saml:AttributeValue><saml:AttributeValue>b</saml:AttributeValue></saml:Attribute></saml:AttributeStatement></saml:Assertion></samlp:Response>`
	nameID, irt, attrs, ok := parseSAMLResponse(xml)
	if !ok || nameID != "bob" || irt != "_r1" || len(attrs["g"]) != 2 || attrs["g"][0] != "a" {
		t.Errorf("parsed %q %q %v %v", nameID, irt, attrs, ok)
	}
	for _, junk := range []string{"", "plain text", "<a><b/></a>"} {
		if _, _, _, ok := parseSAMLResponse(junk); ok {
			t.Errorf("%q parsed", junk)
		}
	}
}

func TestSAMLMetadata(t *testing.T) {
	e := newSAMLEnv(t)
	md := e.svc.SamlMetadata(e.ctx, *e.p)
	for _, want := range []string{
		`entityID="https://app.test/api/auth/sso/corp/metadata"`,
		`Location="https://app.test/api/auth/sso/corp/acs"`,
		`WantAssertionsSigned="true"`,
	} {
		if !strings.Contains(md, want) {
			t.Errorf("metadata lacks %s", want)
		}
	}
	if got := xmlEscape(`a&b<c>"d"`); got != "a&amp;b&lt;c&gt;&quot;d&quot;" {
		t.Errorf("xmlEscape %q", got)
	}
}
