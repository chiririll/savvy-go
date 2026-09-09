package domain

import (
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

func (s SSO) SamlMetadata(p IdentityProvider) string {
	entity := strings.TrimRight(s.AppURL, "/") + "/api/auth/sso/" + p.Slug + "/metadata"
	acs := s.ACSURL(p)
	return fmt.Sprintf(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s" index="0" isDefault="true"/>
  </SPSSODescriptor>
</EntityDescriptor>
`, xmlEscape(entity), xmlEscape(acs))
}

func (s SSO) samlRedirect(ctx context.Context, p IdentityProvider, redirectAfter string) (string, error) {
	ssoURL := cfgString(p.Config, "idp_sso_url")
	if ssoURL == "" {
		return "", ssoErr("missing_field", "Missing required field: IdP SSO URL.", 422)
	}
	reqID := "_" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), " ", "")
	state, err := s.SaveState(ctx, IdPState{
		ProviderID: p.ID, SAMLRequestID: reqID, RedirectAfter: sanitizeRedirect(redirectAfter),
	})
	if err != nil {
		return "", err
	}
	entity := strings.TrimRight(s.AppURL, "/") + "/api/auth/sso/" + p.Slug + "/metadata"
	instant := time.Now().UTC().Format(time.RFC3339)
	doc := fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="%s" Version="2.0" IssueInstant="%s" Destination="%s" AssertionConsumerServiceURL="%s" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"><saml:Issuer>%s</saml:Issuer></samlp:AuthnRequest>`,
		xmlEscape(reqID), xmlEscape(instant), xmlEscape(ssoURL), xmlEscape(s.ACSURL(p)), xmlEscape(entity))
	encoded, err := deflateB64(doc)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(ssoURL)
	if err != nil {
		return "", ssoErr("idp_error", "Invalid IdP SSO URL.", 502)
	}
	q := u.Query()
	q.Set("SAMLRequest", encoded)
	q.Set("RelayState", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s SSO) HandleACS(ctx context.Context, p IdentityProvider, samlResponse, relayState string) (NormalizedIdentity, error) {
	st, err := s.ConsumeState(ctx, relayState)
	if err != nil || st == nil || st.ProviderID != p.ID {
		return NormalizedIdentity{}, ssoErr("invalid_state", "The SSO state is invalid or expired.", 401)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(samlResponse))
	if err != nil || len(raw) == 0 {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML response is invalid.", 401)
	}
	if err := VerifySAMLXML(raw, idpCerts(p)); err != nil {
		return NormalizedIdentity{}, err
	}
	nameID, inResponseTo, attrs, ok := parseSAMLResponse(string(raw))
	if !ok {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML assertion is invalid.", 401)
	}
	if st.SAMLRequestID != "" && inResponseTo != "" && inResponseTo != st.SAMLRequestID {
		return NormalizedIdentity{}, ssoErr("saml_invalid", "The SAML assertion is invalid.", 401)
	}
	return samlIdentity(p, nameID, attrs), nil
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

func parseSAMLResponse(raw string) (nameID, inResponseTo string, attrs map[string][]string, ok bool) {
	if !strings.Contains(raw, "Assertion") && !strings.Contains(raw, "NameID") {
		return "", "", nil, false
	}
	dec := xml.NewDecoder(strings.NewReader(raw))
	attrs = map[string][]string{}
	var attrName string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch local(t.Name.Local) {
			case "Response":
				for _, a := range t.Attr {
					if local(a.Name.Local) == "InResponseTo" {
						inResponseTo = a.Value
					}
				}
			case "NameID":
				var v string
				if dec.DecodeElement(&v, &t) == nil {
					nameID = strings.TrimSpace(v)
				}
			case "Attribute":
				attrName = ""
				for _, a := range t.Attr {
					if local(a.Name.Local) == "Name" {
						attrName = a.Value
					}
				}
			case "AttributeValue":
				var v string
				if dec.DecodeElement(&v, &t) == nil && attrName != "" {
					attrs[attrName] = append(attrs[attrName], strings.TrimSpace(v))
				}
			}
		}
	}
	if nameID == "" && len(attrs) == 0 {
		return "", "", nil, false
	}
	return nameID, inResponseTo, attrs, true
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
