# Authentication & API Integration Features

This document describes the advanced authentication and API integration features in PMP Mock HTTP Server.

## Table of Contents

- [OpenAPI/Swagger Import](#openapiswagger-import)
- [OAuth2 Flow Simulation](#oauth2-flow-simulation)
- [SAML/SSO Mocking](#samlsso-mocking)

---

## OpenAPI/Swagger Import

Auto-generate mock configurations from OpenAPI 3.x and Swagger 2.0 specifications.

### Features

- ✅ **OpenAPI 3.x Support**: Parse and convert OpenAPI 3.0+ specs
- ✅ **Swagger 2.0 Support**: Parse and convert Swagger 2.0 specs
- ✅ **Automatic Mock Generation**: Create mocks for all endpoints
- ✅ **Example Extraction**: Use examples from spec or generate from schemas
- ✅ **Priority Management**: Auto-assign priorities for proper matching
- ✅ **Multiple Formats**: Support JSON and YAML input
- ✅ **URL Fetching**: Import specs directly from URLs

### Usage

#### Command Line Tool

```bash
# Build the import tool
go build -o pmp-import ./cmd/import

# Import from file
./pmp-import --input api-spec.yaml --output mocks/api.yaml

# Import from URL
./pmp-import --input https://api.example.com/openapi.json --output mocks/api.yaml

# Generate examples from schemas
./pmp-import --input api-spec.yaml --output mocks/api.yaml --generate-examples
```

#### Programmatic Usage

```go
import "github.com/comfortablynumb/pmp-mock-http/internal/openapi"

// Create parser
parser := openapi.NewParser(true) // true = generate examples

// Parse spec
mockSpec, err := parser.ParseFile("api-spec.yaml")
if err != nil {
    log.Fatal(err)
}

// Save mocks
err = openapi.SaveMocks(mockSpec, "mocks/generated.yaml")
```

### Example

**Input OpenAPI Spec:**
```yaml
openapi: 3.0.0
info:
  title: Pet Store API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: Success
          content:
            application/json:
              example:
                - id: 1
                  name: "Fluffy"
```

**Generated Mock:**
```yaml
mocks:
  - name: "get /pets"
    priority: 100
    request:
      uri: "/pets"
      method: "GET"
    response:
      status_code: 200
      headers:
        Content-Type: "application/json"
      body: |
        [{"id":1,"name":"Fluffy"}]
```

### Supported Features

| Feature | OpenAPI 3.x | Swagger 2.0 |
|---------|-------------|-------------|
| Paths & Operations | ✅ | ✅ |
| Request Bodies | ✅ | ✅ |
| Response Examples | ✅ | ✅ |
| Schema Examples | ✅ | ✅ |
| Parameters | ✅ | ✅ |
| Multiple Methods | ✅ | ✅ |
| Base Path | ✅ | ✅ |

---

## OAuth2 Flow Simulation

Complete OAuth2 and OpenID Connect server simulation for testing authentication flows.

### Features

- ✅ **Authorization Code Flow**: Full OAuth2 authorization code grant
- ✅ **Client Credentials Flow**: Machine-to-machine authentication
- ✅ **Implicit Flow**: Legacy browser-based flow
- ✅ **Password Grant**: Resource owner password credentials
- ✅ **Refresh Tokens**: Token refresh support
- ✅ **PKCE Support**: Proof Key for Code Exchange
- ✅ **JWT Tokens**: RS256-signed JSON Web Tokens
- ✅ **OpenID Connect**: ID tokens and userinfo endpoint
- ✅ **JWKS Endpoint**: Public key discovery
- ✅ **Discovery**: OpenID Connect configuration endpoint

### Quick Start

#### Using Built-in OAuth2 Provider

```go
import "github.com/comfortablynumb/pmp-mock-http/internal/oauth"

// Create OAuth2 provider
provider, err := oauth.NewOAuth2Provider("http://localhost:8083")
if err != nil {
    log.Fatal(err)
}

// Register custom client
provider.RegisterClient(&oauth.Client{
    ClientID:     "my-app",
    ClientSecret: "my-secret",
    RedirectURIs: []string{"http://localhost:3000/callback"},
    Scopes:       []string{"openid", "profile", "email"},
})

// Set up endpoints
http.HandleFunc("/oauth/authorize", provider.HandleAuthorize)
http.HandleFunc("/oauth/token", provider.HandleToken)
http.HandleFunc("/oauth/userinfo", provider.HandleUserInfo)
http.HandleFunc("/.well-known/jwks.json", provider.HandleJWKS)
```

#### Using Mock Configuration

See `examples/oauth/oauth-server.yaml` for a complete mock configuration.

### OAuth2 Flows

#### 1. Authorization Code Flow

**Step 1: Authorization Request**
```http
GET /oauth/authorize?
    response_type=code&
    client_id=my-app&
    redirect_uri=http://localhost:3000/callback&
    scope=openid%20profile%20email&
    state=xyz
```

**Step 2: Token Exchange**
```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=AUTH_CODE&
redirect_uri=http://localhost:3000/callback&
client_id=my-app&
client_secret=my-secret
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "tGzv3JOkF0XG5Qx2TlKWIA",
  "scope": "openid profile email",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### 2. Client Credentials Flow

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&
client_id=my-app&
client_secret=my-secret&
scope=api:read api:write
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api:read api:write"
}
```

#### 3. Refresh Token Flow

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token&
refresh_token=tGzv3JOkF0XG5Qx2TlKWIA&
client_id=my-app&
client_secret=my-secret
```

#### 4. PKCE (Proof Key for Code Exchange)

**Step 1: Authorization with PKCE**
```http
GET /oauth/authorize?
    response_type=code&
    client_id=my-app&
    redirect_uri=http://localhost:3000/callback&
    code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
    code_challenge_method=S256
```

**Step 2: Token Exchange with Verifier**
```http
POST /oauth/token

grant_type=authorization_code&
code=AUTH_CODE&
code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
```

### OpenID Connect

#### Discovery Endpoint

```http
GET /.well-known/openid-configuration
```

**Response:**
```json
{
  "issuer": "http://localhost:8083",
  "authorization_endpoint": "http://localhost:8083/oauth/authorize",
  "token_endpoint": "http://localhost:8083/oauth/token",
  "userinfo_endpoint": "http://localhost:8083/oauth/userinfo",
  "jwks_uri": "http://localhost:8083/.well-known/jwks.json",
  "response_types_supported": ["code", "token", "id_token"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"]
}
```

#### UserInfo Endpoint

```http
GET /oauth/userinfo
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response:**
```json
{
  "sub": "user-123",
  "name": "John Doe",
  "given_name": "John",
  "family_name": "Doe",
  "email": "john@example.com",
  "email_verified": true
}
```

### JWT Token Structure

**Access Token Claims:**
```json
{
  "iss": "http://localhost:8083",
  "sub": "user-123",
  "aud": "my-app",
  "exp": 1234567890,
  "iat": 1234564290,
  "scope": "openid profile email",
  "client_id": "my-app"
}
```

**ID Token Claims:**
```json
{
  "iss": "http://localhost:8083",
  "sub": "user-123",
  "aud": "my-app",
  "exp": 1234567890,
  "iat": 1234564290,
  "name": "John Doe",
  "email": "john@example.com",
  "email_verified": true
}
```

---

## SAML/SSO Mocking

Complete SAML 2.0 Identity Provider (IdP) simulation for testing enterprise SSO.

### Features

- ✅ **SAML 2.0 Support**: Full SAML 2.0 protocol
- ✅ **SP-Initiated Flow**: Service Provider initiated SSO
- ✅ **IdP-Initiated Flow**: Identity Provider initiated SSO
- ✅ **HTTP-POST Binding**: Auto-submit forms
- ✅ **HTTP-Redirect Binding**: URL-based flow
- ✅ **Assertions**: Signed SAML assertions
- ✅ **Attributes**: Custom user attributes
- ✅ **Metadata**: IdP metadata endpoint
- ✅ **Sessions**: Session management
- ✅ **Self-Signed Certificates**: Auto-generated certificates

### Quick Start

#### Using Built-in SAML Provider

```go
import "github.com/comfortablynumb/pmp-mock-http/internal/saml"

// Create SAML provider
provider, err := saml.NewSAMLProvider("http://localhost:8083/saml")
if err != nil {
    log.Fatal(err)
}

// Set up endpoints
http.HandleFunc("/saml/sso", provider.HandleSSO)
http.HandleFunc("/saml/metadata", provider.HandleMetadata)
```

#### Using Mock Configuration

See `examples/saml/saml-idp.yaml` for a complete mock configuration.

### SAML Flows

#### 1. SP-Initiated SSO Flow

**Step 1: SP Redirects to IdP**
```http
GET /saml/sso?
    SAMLRequest=<base64_encoded_request>&
    RelayState=target_url
```

**Step 2: IdP Authenticates User**

The IdP displays a login form (or auto-authenticates in mock mode).

**Step 3: IdP Posts SAML Response to SP**

The IdP generates an HTML form that auto-submits:

```html
<form method="post" action="http://sp.example.com/saml/acs">
  <input type="hidden" name="SAMLResponse" value="<base64_saml_response>"/>
  <input type="hidden" name="RelayState" value="target_url"/>
</form>
```

**Step 4: SP Validates and Creates Session**

#### 2. IdP-Initiated SSO Flow

**Step 1: User Accesses IdP**
```http
GET /saml/sso?acs=http://sp.example.com/saml/acs
```

**Step 2: IdP Posts to SP**

Same as SP-initiated flow, but without initial SAML request.

### SAML Assertion Structure

```xml
<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                ID="_123456"
                Version="2.0"
                IssueInstant="2024-01-15T10:00:00Z">
  <saml:Issuer>http://localhost:8083/saml</saml:Issuer>
  <saml:Subject>
    <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
      user@example.com
    </saml:NameID>
    <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml:SubjectConfirmationData
          NotOnOrAfter="2024-01-15T11:00:00Z"
          Recipient="http://sp.example.com/saml/acs"/>
    </saml:SubjectConfirmation>
  </saml:Subject>
  <saml:Conditions NotBefore="2024-01-15T10:00:00Z"
                   NotOnOrAfter="2024-01-15T11:00:00Z">
    <saml:AudienceRestriction>
      <saml:Audience>http://sp.example.com</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>
  <saml:AttributeStatement>
    <saml:Attribute Name="email">
      <saml:AttributeValue>user@example.com</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="firstName">
      <saml:AttributeValue>John</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="lastName">
      <saml:AttributeValue>Doe</saml:AttributeValue>
    </saml:Attribute>
  </saml:AttributeStatement>
  <saml:AuthnStatement AuthnInstant="2024-01-15T10:00:00Z"
                       SessionIndex="session-123">
    <saml:AuthnContext>
      <saml:AuthnContextClassRef>
        urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
      </saml:AuthnContextClassRef>
    </saml:AuthnContext>
  </saml:AuthnStatement>
</saml:Assertion>
```

### Metadata Endpoint

```http
GET /saml/metadata
```

**Response:**
```xml
<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="http://localhost:8083/saml">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIIDXTCCAkWgAwIBAgIJAKL0UG+mRK...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                         Location="http://localhost:8083/saml/sso"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="http://localhost:8083/saml/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>
```

### Testing SAML Integration

#### Using a Test SP

Many SAML service providers offer test/sandbox environments:

1. **Okta**: https://developer.okta.com/
2. **Auth0**: https://auth0.com/docs/authenticate/protocols/saml
3. **OneLogin**: https://developers.onelogin.com/

#### Configuration Tips

1. **Entity ID**: Use `http://localhost:8083/saml`
2. **SSO URL**: Use `http://localhost:8083/saml/sso`
3. **Certificate**: Use the certificate from `/saml/metadata`
4. **Attributes**: email, firstName, lastName, role

---

## Examples

Complete working examples are available in the `examples/` directory:

- `examples/oauth/oauth-server.yaml` - OAuth2/OpenID Connect server
- `examples/saml/saml-idp.yaml` - SAML 2.0 Identity Provider
- `examples/openapi/` - OpenAPI import examples

---

## Best Practices

### OAuth2

1. **Token Expiry**: Use realistic expiry times (1 hour for access, 30 days for refresh)
2. **Scopes**: Define clear, granular scopes
3. **PKCE**: Always use PKCE for mobile/SPA applications
4. **State Parameter**: Validate state to prevent CSRF
5. **Token Storage**: Store tokens securely (httpOnly cookies, secure storage)

### SAML

1. **Certificate Rotation**: Plan for certificate expiration
2. **Clock Skew**: Allow for time differences between systems
3. **Attribute Mapping**: Document required attributes
4. **Session Duration**: Set appropriate session timeouts
5. **Logout**: Implement SLO (Single Logout) if needed

### OpenAPI Import

1. **Examples**: Provide examples in your OpenAPI spec
2. **Descriptions**: Use clear operation descriptions
3. **Schemas**: Define comprehensive schemas
4. **Versioning**: Include API version in the spec
5. **Validation**: Validate generated mocks before use

---

## Troubleshooting

### OAuth2 Issues

**Problem**: Invalid token error
- **Solution**: Check token expiry, verify JWT signature

**Problem**: Invalid redirect URI
- **Solution**: Ensure redirect URI exactly matches registered URI

**Problem**: Invalid client
- **Solution**: Verify client ID and secret

### SAML Issues

**Problem**: Signature validation fails
- **Solution**: Ensure clock synchronization, check certificate

**Problem**: Assertion expired
- **Solution**: Check NotOnOrAfter timestamps

**Problem**: Audience restriction
- **Solution**: Verify SP entity ID matches audience

### OpenAPI Import Issues

**Problem**: No mocks generated
- **Solution**: Check spec format (OpenAPI 3.x or Swagger 2.0)

**Problem**: Missing examples
- **Solution**: Use `--generate-examples` flag

**Problem**: Incorrect paths
- **Solution**: Check basePath in Swagger 2.0 specs

---

## Contributing

Contributions are welcome! Please see the main project README for contribution guidelines.

## License

See main project LICENSE file.
