package saml

import (
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SAMLProvider manages SAML SSO flows
type SAMLProvider struct {
	issuer          string
	cert            *x509.Certificate
	privateKey      *rsa.PrivateKey
	assertionExpiry time.Duration
	sessions        map[string]*SAMLSession
	mu              sync.RWMutex
}

// SAMLSession represents a SAML session
type SAMLSession struct {
	SessionID    string
	NameID       string
	Attributes   map[string]string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// SAMLResponse represents a SAML 2.0 Response
type SAMLResponse struct {
	XMLName      xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`
	ID           string   `xml:"ID,attr"`
	Version      string   `xml:"Version,attr"`
	IssueInstant string   `xml:"IssueInstant,attr"`
	Destination  string   `xml:"Destination,attr,omitempty"`
	Issuer       Issuer   `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Status       Status   `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
	Assertion    Assertion `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
}

// Issuer represents the SAML issuer
type Issuer struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Value   string   `xml:",chardata"`
}

// Status represents the SAML status
type Status struct {
	XMLName    xml.Name   `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
	StatusCode StatusCode `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`
}

// StatusCode represents the SAML status code
type StatusCode struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`
	Value   string   `xml:"Value,attr"`
}

// Assertion represents a SAML assertion
type Assertion struct {
	XMLName            xml.Name           `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                 string             `xml:"ID,attr"`
	Version            string             `xml:"Version,attr"`
	IssueInstant       string             `xml:"IssueInstant,attr"`
	Issuer             Issuer             `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Subject            Subject            `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`
	Conditions         Conditions         `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`
	AttributeStatement AttributeStatement `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`
	AuthnStatement     AuthnStatement     `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnStatement"`
}

// Subject represents the SAML subject
type Subject struct {
	XMLName             xml.Name            `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`
	NameID              NameID              `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	SubjectConfirmation SubjectConfirmation `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`
}

// NameID represents the SAML NameID
type NameID struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	Format  string   `xml:"Format,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// SubjectConfirmation represents subject confirmation
type SubjectConfirmation struct {
	XMLName                 xml.Name                `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`
	Method                  string                  `xml:"Method,attr"`
	SubjectConfirmationData SubjectConfirmationData `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`
}

// SubjectConfirmationData represents subject confirmation data
type SubjectConfirmationData struct {
	XMLName      xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`
	NotOnOrAfter string   `xml:"NotOnOrAfter,attr"`
	Recipient    string   `xml:"Recipient,attr"`
}

// Conditions represents SAML conditions
type Conditions struct {
	XMLName             xml.Name            `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`
	NotBefore           string              `xml:"NotBefore,attr"`
	NotOnOrAfter        string              `xml:"NotOnOrAfter,attr"`
	AudienceRestriction AudienceRestriction `xml:"urn:oasis:names:tc:SAML:2.0:assertion AudienceRestriction"`
}

// AudienceRestriction represents audience restriction
type AudienceRestriction struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AudienceRestriction"`
	Audience string   `xml:"urn:oasis:names:tc:SAML:2.0:assertion Audience"`
}

// AttributeStatement represents SAML attributes
type AttributeStatement struct {
	XMLName    xml.Name    `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`
	Attributes []Attribute `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`
}

// Attribute represents a SAML attribute
type Attribute struct {
	XMLName        xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`
	Name           string           `xml:"Name,attr"`
	NameFormat     string           `xml:"NameFormat,attr,omitempty"`
	AttributeValue []AttributeValue `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeValue"`
}

// AttributeValue represents an attribute value
type AttributeValue struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeValue"`
	Type    string   `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	Value   string   `xml:",chardata"`
}

// AuthnStatement represents authentication statement
type AuthnStatement struct {
	XMLName             xml.Name    `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnStatement"`
	AuthnInstant        string      `xml:"AuthnInstant,attr"`
	SessionIndex        string      `xml:"SessionIndex,attr"`
	AuthnContext        AuthnContext `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnContext"`
}

// AuthnContext represents authentication context
type AuthnContext struct {
	XMLName              xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnContext"`
	AuthnContextClassRef string   `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnContextClassRef"`
}

// NewSAMLProvider creates a new SAML provider
func NewSAMLProvider(issuer string) (*SAMLProvider, error) {
	// Generate self-signed certificate for SAML
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mock SAML IdP"},
			CommonName:   issuer,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &SAMLProvider{
		issuer:          issuer,
		cert:            cert,
		privateKey:      privateKey,
		assertionExpiry: time.Hour,
		sessions:        make(map[string]*SAMLSession),
	}, nil
}

// HandleSSO handles SP-initiated SSO
func (p *SAMLProvider) HandleSSO(w http.ResponseWriter, r *http.Request) {
	// Parse SAML request (if present)
	samlRequest := r.URL.Query().Get("SAMLRequest")
	relayState := r.URL.Query().Get("RelayState")

	// For mock purposes, auto-authenticate
	nameID := "user@example.com"

	// Create session
	sessionID := p.generateID()
	session := &SAMLSession{
		SessionID: sessionID,
		NameID:    nameID,
		Attributes: map[string]string{
			"email":      nameID,
			"firstName":  "Mock",
			"lastName":   "User",
			"role":       "user",
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 8),
	}

	p.mu.Lock()
	p.sessions[sessionID] = session
	p.mu.Unlock()

	// Get ACS URL from request or use default
	acsURL := r.URL.Query().Get("acs")
	if acsURL == "" {
		acsURL = "http://localhost:8080/saml/acs"
	}

	// Generate SAML response
	samlResponse := p.generateSAMLResponse(nameID, sessionID, acsURL, session.Attributes)

	// Encode response
	encoded, err := p.encodeSAMLResponse(samlResponse)
	if err != nil {
		log.Printf("SAML: Error encoding response: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Show form to POST to SP
	p.renderSAMLPostForm(w, acsURL, encoded, relayState, samlRequest)
}

// HandleMetadata handles the metadata endpoint
func (p *SAMLProvider) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	// Generate IdP metadata
	metadata := fmt.Sprintf(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="%s">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>%s</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                         Location="%s/saml/sso"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="%s/saml/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`, p.issuer, p.getCertificateString(), p.issuer, p.issuer)

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write([]byte(metadata)); err != nil {
		log.Printf("SAML: Error writing metadata: %v\n", err)
	}
}

// generateSAMLResponse generates a SAML response
func (p *SAMLProvider) generateSAMLResponse(nameID, sessionID, acsURL string, attributes map[string]string) *SAMLResponse {
	now := time.Now()
	notOnOrAfter := now.Add(p.assertionExpiry)

	// Build attributes
	var attrs []Attribute
	for name, value := range attributes {
		attrs = append(attrs, Attribute{
			Name:       name,
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
			AttributeValue: []AttributeValue{
				{
					Type:  "xs:string",
					Value: value,
				},
			},
		})
	}

	response := &SAMLResponse{
		ID:           p.generateID(),
		Version:      "2.0",
		IssueInstant: now.UTC().Format(time.RFC3339),
		Destination:  acsURL,
		Issuer: Issuer{
			Value: p.issuer,
		},
		Status: Status{
			StatusCode: StatusCode{
				Value: "urn:oasis:names:tc:SAML:2.0:status:Success",
			},
		},
		Assertion: Assertion{
			ID:           p.generateID(),
			Version:      "2.0",
			IssueInstant: now.UTC().Format(time.RFC3339),
			Issuer: Issuer{
				Value: p.issuer,
			},
			Subject: Subject{
				NameID: NameID{
					Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
					Value:  nameID,
				},
				SubjectConfirmation: SubjectConfirmation{
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: SubjectConfirmationData{
						NotOnOrAfter: notOnOrAfter.UTC().Format(time.RFC3339),
						Recipient:    acsURL,
					},
				},
			},
			Conditions: Conditions{
				NotBefore:    now.UTC().Format(time.RFC3339),
				NotOnOrAfter: notOnOrAfter.UTC().Format(time.RFC3339),
				AudienceRestriction: AudienceRestriction{
					Audience: acsURL,
				},
			},
			AttributeStatement: AttributeStatement{
				Attributes: attrs,
			},
			AuthnStatement: AuthnStatement{
				AuthnInstant: now.UTC().Format(time.RFC3339),
				SessionIndex: sessionID,
				AuthnContext: AuthnContext{
					AuthnContextClassRef: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
				},
			},
		},
	}

	return response
}

// encodeSAMLResponse encodes a SAML response for HTTP-POST binding
func (p *SAMLProvider) encodeSAMLResponse(response *SAMLResponse) (string, error) {
	// Marshal to XML
	xmlData, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal SAML response: %w", err)
	}

	// Base64 encode (HTTP-POST binding doesn't use deflate)
	encoded := base64.StdEncoding.EncodeToString(xmlData)
	return encoded, nil
}

// renderSAMLPostForm renders the auto-submit form for SAML HTTP-POST binding
func (p *SAMLProvider) renderSAMLPostForm(w http.ResponseWriter, acsURL, samlResponse, relayState, samlRequest string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>SAML SSO</title>
</head>
<body onload="document.forms[0].submit()">
    <form method="post" action="%s">
        <input type="hidden" name="SAMLResponse" value="%s"/>
        %s
        <noscript>
            <p>JavaScript is disabled. Please click the button below to continue.</p>
            <button type="submit">Continue</button>
        </noscript>
    </form>
</body>
</html>`, acsURL, samlResponse, func() string {
		if relayState != "" {
			return fmt.Sprintf(`<input type="hidden" name="RelayState" value="%s"/>`, relayState)
		}
		return ""
	}())

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("SAML: Error writing form: %v\n", err)
	}
}

// generateID generates a random SAML ID
func (p *SAMLProvider) generateID() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		log.Printf("SAML: Error generating ID: %v\n", err)
	}
	return "_" + fmt.Sprintf("%x", b)
}

// getCertificateString returns the certificate as a base64 string
func (p *SAMLProvider) getCertificateString() string {
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.cert.Raw,
	})
	// Remove PEM headers and newlines
	certStr := string(certPEM)
	certStr = strings.ReplaceAll(certStr, "-----BEGIN CERTIFICATE-----", "")
	certStr = strings.ReplaceAll(certStr, "-----END CERTIFICATE-----", "")
	certStr = strings.ReplaceAll(certStr, "\n", "")
	return certStr
}

// DecodeSAMLRequest decodes a SAML request (for SP-initiated flow)
func DecodeSAMLRequest(encoded string) ([]byte, error) {
	// URL decode
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, err
	}

	// Base64 decode
	compressed, err := base64.StdEncoding.DecodeString(decoded)
	if err != nil {
		return nil, err
	}

	// Deflate decompress (HTTP-Redirect binding uses deflate)
	reader := flate.NewReader(strings.NewReader(string(compressed)))
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("SAML: Error closing reader: %v\n", err)
		}
	}()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
