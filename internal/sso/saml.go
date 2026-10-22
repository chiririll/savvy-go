package sso

import (
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/beevik/etree"
)

func (s Service) SamlMetadata(ctx context.Context, p IdentityProvider) string {
	entity := s.samlEntityID(p)
	acs := s.ACSURL(ctx, p)
	return fmt.Sprintf(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true"
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="%s" index="0" isDefault="true"/>
  </SPSSODescriptor>
</EntityDescriptor>
`, xmlEscape(entity), xmlEscape(acs))
}

func (s Service) samlRedirect(ctx context.Context, p IdentityProvider, redirectAfter string) (string, error) {
	ssoURL := cfgString(p.Config, "idp_sso_url")
	if ssoURL == "" {
		return "", ssoErr("missing_field", "Missing required field: IdP Service URL.", 422)
	}
	reqID := "_" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), " ", "")
	state, err := s.SaveState(ctx, IdPState{
		ProviderID: p.ID, SAMLRequestID: reqID, RedirectAfter: sanitizeRedirect(redirectAfter),
	})
	if err != nil {
		return "", err
	}
	entity := s.samlEntityID(p)
	instant := time.Now().UTC().Format(time.RFC3339)
	doc := fmt.Sprintf(`<samlp:AuthnRequest`+
		` xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"`+
		` xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"`+
		` ID="%s" Version="2.0" IssueInstant="%s" Destination="%s" AssertionConsumerServiceURL="%s"`+
		` ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">`+
		`<saml:Issuer>%s</saml:Issuer></samlp:AuthnRequest>`,
		xmlEscape(reqID), xmlEscape(instant), xmlEscape(ssoURL), xmlEscape(s.ACSURL(ctx, p)), xmlEscape(entity))
	encoded, err := deflateB64(doc)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(ssoURL)
	if err != nil {
		return "", ssoErr("idp_error", "Invalid IdP Service URL.", 502)
	}
	q := u.Query()
	q.Set("SAMLRequest", encoded)
	q.Set("RelayState", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s Service) HandleACS(
	ctx context.Context,
	p IdentityProvider,
	samlResponse, relayState string,
) (NormalizedIdentity, error) {
	st, err := s.ConsumeState(ctx, relayState)
	if err != nil || st == nil || st.ProviderID != p.ID {
		return NormalizedIdentity{}, ssoErr("invalid_state", "The Service state is invalid or expired.", 401)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(samlResponse))
	if err != nil || len(raw) == 0 {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML response is invalid.", 401)
	}
	signed, err := verifiedSAML(raw, idpCerts(p))
	if err != nil {
		return NormalizedIdentity{}, err
	}
	// Read only what the signature covers.
	vouched := etree.NewDocument()
	vouched.SetRoot(signed.Copy())
	signedXML, err := vouched.WriteToString()
	if err != nil {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML response is invalid.", 401)
	}
	nameID, inResponseTo, attrs, ok := parseSAMLResponse(signedXML)
	if !ok {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML assertion is invalid.", 401)
	}
	// Only logins we started are accepted, so the signed assertion must answer exactly this request.
	if st.SAMLRequestID == "" || inResponseTo != st.SAMLRequestID {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML assertion is invalid.", 401)
	}
	if !samlConditionsHold(signedXML, cfgString(p.Config, "idp_entity_id"), s.samlEntityID(p), time.Now()) {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML assertion is invalid.", 401)
	}
	return samlIdentity(p, nameID, attrs), nil
}

func (s Service) samlEntityID(p IdentityProvider) string {
	return s.Cfg.AppURL() + "/api/auth/sso/" + p.Slug + "/metadata"
}

// samlClockSkew is how far the IdP's clock may differ from ours.
const samlClockSkew = 2 * time.Minute

// samlWithin reports whether now falls inside the NotBefore/NotOnOrAfter
// window, allowing for clock skew. A missing end only passes if needEnd is false.
func samlWithin(now time.Time, notBefore, notOnOrAfter string, needEnd bool) bool {
	if notBefore != "" {
		t, err := time.Parse(time.RFC3339, notBefore)
		if err != nil || now.Add(samlClockSkew).Before(t) {
			return false
		}
	}
	if notOnOrAfter == "" {
		return !needEnd
	}
	t, err := time.Parse(time.RFC3339, notOnOrAfter)
	return err == nil && now.Add(-samlClockSkew).Before(t)
}

// samlAttrOK is the value of attribute name of t and whether t has it. Where
// an attribute repeats, the last one wins.
func samlAttrOK(t xml.StartElement, name string) (value string, found bool) {
	for _, a := range t.Attr {
		if local(a.Name.Local) == name {
			value, found = a.Value, true
		}
	}
	return value, found
}

// samlAttr is the first value of attribute name of t, or "". The conditions
// check relies on the first one winning.
func samlAttr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if local(a.Name.Local) == name {
			return a.Value
		}
	}
	return ""
}

// samlScan collects what samlConditionsHold needs while walking the assertion.
type samlScan struct {
	idpEntity    string
	now          time.Time
	restrictions [][]string
	issuers      int
	conditions   bool
}

// visit checks one start element. It returns false when the assertion must be rejected.
func (sc *samlScan) visit(dec *xml.Decoder, t xml.StartElement) bool {
	switch local(t.Name.Local) {
	case "Issuer":
		var v string
		if dec.DecodeElement(&v, &t) != nil || strings.TrimSpace(v) != sc.idpEntity {
			return false
		}
		sc.issuers++
	case "Conditions":
		sc.conditions = true
		return samlWithin(sc.now, samlAttr(t, "NotBefore"), samlAttr(t, "NotOnOrAfter"), true)
	case "SubjectConfirmationData":
		return samlWithin(sc.now, samlAttr(t, "NotBefore"), samlAttr(t, "NotOnOrAfter"), false)
	case "AudienceRestriction":
		sc.restrictions = append(sc.restrictions, nil)
	case "Audience":
		var v string
		if dec.DecodeElement(&v, &t) != nil || len(sc.restrictions) == 0 {
			return false
		}
		last := len(sc.restrictions) - 1
		sc.restrictions[last] = append(sc.restrictions[last], strings.TrimSpace(v))
	}
	return true
}

// samlConditionsHold checks the signed assertion came from the configured IdP,
// is meant for us, and is valid right now. Anything missing counts as failing.
func samlConditionsHold(raw, idpEntity, audience string, now time.Time) bool {
	if idpEntity == "" {
		return false
	}
	sc := &samlScan{idpEntity: idpEntity, now: now}
	dec := xml.NewDecoder(strings.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if t, ok := tok.(xml.StartElement); ok && !sc.visit(dec, t) {
			return false
		}
	}
	if sc.issuers == 0 || !sc.conditions || len(sc.restrictions) == 0 {
		return false
	}
	for _, r := range sc.restrictions {
		if !slices.Contains(r, audience) {
			return false
		}
	}
	return true
}

func samlIdentity(p IdentityProvider, nameID string, attrs map[string][]string) NormalizedIdentity {
	maps := p.ClaimMappings
	if maps == nil {
		if preset, ok := PresetByKey(p.Preset); ok {
			maps = preset.DefaultClaims
		}
	}
	first := func(key string) string {
		raw := maps[key]
		attr, _ := raw.(string)
		if attr == "" {
			return ""
		}
		if vals := attrs[attr]; len(vals) > 0 {
			return vals[0]
		}
		return ""
	}
	subject := first("subject")
	if subject == "" {
		subject = nameID
	}
	id := NormalizedIdentity{
		Subject: subject,
		Email:   first("email"),
		Name:    first("name"),
		Raw:     map[string]any{"nameid": nameID},
	}
	if id.Email != "" {
		id.EmailVerified = true
	}
	if g, _ := maps["groups"].(string); g != "" {
		id.Groups = attrs[g]
	}
	for k, v := range attrs {
		if len(v) == 1 {
			id.Raw[k] = v[0]
		} else {
			id.Raw[k] = v
		}
	}
	return id
}

// samlParse collects the name ID, request ID and attributes of a response.
type samlParse struct {
	nameID, inResponseTo string
	attrs                map[string][]string
	attrName             string
}

// visit reads one start element.
func (sp *samlParse) visit(dec *xml.Decoder, t xml.StartElement) {
	switch local(t.Name.Local) {
	case "Response":
		if v, ok := samlAttrOK(t, "InResponseTo"); ok {
			sp.inResponseTo = v
		}
	case "SubjectConfirmationData":
		if v, ok := samlAttrOK(t, "InResponseTo"); ok && sp.inResponseTo == "" {
			sp.inResponseTo = v
		}
	case "NameID":
		var v string
		if dec.DecodeElement(&v, &t) == nil {
			sp.nameID = strings.TrimSpace(v)
		}
	case "Attribute":
		sp.attrName, _ = samlAttrOK(t, "Name")
	case "AttributeValue":
		var v string
		if dec.DecodeElement(&v, &t) == nil && sp.attrName != "" {
			sp.attrs[sp.attrName] = append(sp.attrs[sp.attrName], strings.TrimSpace(v))
		}
	}
}

func parseSAMLResponse(raw string) (nameID, inResponseTo string, attrs map[string][]string, ok bool) {
	if !strings.Contains(raw, "Assertion") && !strings.Contains(raw, "NameID") {
		return "", "", nil, false
	}
	dec := xml.NewDecoder(strings.NewReader(raw))
	sp := &samlParse{attrs: map[string][]string{}}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if t, isStart := tok.(xml.StartElement); isStart {
			sp.visit(dec, t)
		}
	}
	if sp.nameID == "" && len(sp.attrs) == 0 {
		return "", "", nil, false
	}
	return sp.nameID, sp.inResponseTo, sp.attrs, true
}

func local(n string) string {
	if i := strings.IndexByte(n, ':'); i >= 0 {
		return n[i+1:]
	}
	return n
}

func deflateB64(s string) (string, error) {
	var b strings.Builder
	w, err := flate.NewWriter(&b, flate.DefaultCompression)
	if err != nil {
		return "", err
	}
	if _, err := io.WriteString(w, s); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(b.String())), nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
