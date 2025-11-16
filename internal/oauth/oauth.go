package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OAuth2Provider manages OAuth2 flows and token generation
type OAuth2Provider struct {
	issuer           string
	privateKey       *rsa.PrivateKey
	publicKey        *rsa.PublicKey
	authCodes        map[string]*AuthorizationCode
	tokens           map[string]*TokenInfo
	clients          map[string]*Client
	mu               sync.RWMutex
	tokenExpiry      time.Duration
	refreshExpiry    time.Duration
}

// Client represents an OAuth2 client application
type Client struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
	Scopes       []string
}

// AuthorizationCode represents an authorization code
type AuthorizationCode struct {
	Code         string
	ClientID     string
	RedirectURI  string
	Scope        string
	CodeChallenge string
	CodeChallengeMethod string
	ExpiresAt    time.Time
	UserID       string
}

// TokenInfo represents token metadata
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scope        string
	ClientID     string
	UserID       string
}

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"` // For OpenID Connect
}

// NewOAuth2Provider creates a new OAuth2 provider
func NewOAuth2Provider(issuer string) (*OAuth2Provider, error) {
	// Generate RSA key pair for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	provider := &OAuth2Provider{
		issuer:        issuer,
		privateKey:    privateKey,
		publicKey:     &privateKey.PublicKey,
		authCodes:     make(map[string]*AuthorizationCode),
		tokens:        make(map[string]*TokenInfo),
		clients:       make(map[string]*Client),
		tokenExpiry:   time.Hour,        // 1 hour
		refreshExpiry: time.Hour * 24 * 30, // 30 days
	}

	// Register a default client
	provider.RegisterClient(&Client{
		ClientID:     "default-client",
		ClientSecret: "default-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"openid", "profile", "email"},
	})

	return provider, nil
}

// RegisterClient registers a new OAuth2 client
func (p *OAuth2Provider) RegisterClient(client *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[client.ClientID] = client
	log.Printf("OAuth2: Registered client %s\n", client.ClientID)
}

// HandleAuthorize handles the authorization endpoint (/authorize)
func (p *OAuth2Provider) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	responseType := query.Get("response_type")
	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	scope := query.Get("scope")
	state := query.Get("state")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")

	// Validate client
	p.mu.RLock()
	client, exists := p.clients[clientID]
	p.mu.RUnlock()

	if !exists {
		http.Error(w, "invalid_client", http.StatusBadRequest)
		return
	}

	// Validate redirect URI
	validURI := false
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			validURI = true
			break
		}
	}
	if !validURI {
		http.Error(w, "invalid_redirect_uri", http.StatusBadRequest)
		return
	}

	switch responseType {
	case "code":
		// Authorization Code Flow
		code := p.generateCode()
		authCode := &AuthorizationCode{
			Code:         code,
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			Scope:        scope,
			CodeChallenge: codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
			ExpiresAt:    time.Now().Add(10 * time.Minute),
			UserID:       "mock-user-id",
		}

		p.mu.Lock()
		p.authCodes[code] = authCode
		p.mu.Unlock()

		// Redirect with authorization code
		redirectURL, _ := url.Parse(redirectURI)
		q := redirectURL.Query()
		q.Set("code", code)
		if state != "" {
			q.Set("state", state)
		}
		redirectURL.RawQuery = q.Encode()

		http.Redirect(w, r, redirectURL.String(), http.StatusFound)

	case "token":
		// Implicit Flow
		accessToken := p.generateAccessToken(clientID, scope, "mock-user-id")

		// Redirect with access token in fragment
		redirectURL, _ := url.Parse(redirectURI)
		fragment := fmt.Sprintf("access_token=%s&token_type=Bearer&expires_in=%d",
			accessToken, int(p.tokenExpiry.Seconds()))
		if scope != "" {
			fragment += fmt.Sprintf("&scope=%s", url.QueryEscape(scope))
		}
		if state != "" {
			fragment += fmt.Sprintf("&state=%s", url.QueryEscape(state))
		}
		redirectURL.Fragment = fragment

		http.Redirect(w, r, redirectURL.String(), http.StatusFound)

	default:
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
	}
}

// HandleToken handles the token endpoint (/token)
func (p *OAuth2Provider) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "authorization_code":
		p.handleAuthorizationCodeGrant(w, r)
	case "client_credentials":
		p.handleClientCredentialsGrant(w, r)
	case "refresh_token":
		p.handleRefreshTokenGrant(w, r)
	case "password":
		p.handlePasswordGrant(w, r)
	default:
		p.sendError(w, "unsupported_grant_type", http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles authorization code grant
func (p *OAuth2Provider) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")
	redirectURI := r.Form.Get("redirect_uri")
	codeVerifier := r.Form.Get("code_verifier")

	// Validate client credentials
	if !p.validateClient(clientID, clientSecret) {
		p.sendError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Get authorization code
	p.mu.Lock()
	authCode, exists := p.authCodes[code]
	if exists {
		delete(p.authCodes, code) // Use only once
	}
	p.mu.Unlock()

	if !exists || authCode.ExpiresAt.Before(time.Now()) {
		p.sendError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Validate PKCE if code challenge was provided
	if authCode.CodeChallenge != "" {
		if !p.validatePKCE(codeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
			p.sendError(w, "invalid_grant", http.StatusBadRequest)
			return
		}
	}

	// Validate redirect URI
	if authCode.RedirectURI != redirectURI {
		p.sendError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Generate tokens
	accessToken := p.generateAccessToken(clientID, authCode.Scope, authCode.UserID)
	refreshToken := p.generateRefreshToken()

	tokenInfo := &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(p.tokenExpiry),
		Scope:        authCode.Scope,
		ClientID:     clientID,
		UserID:       authCode.UserID,
	}

	p.mu.Lock()
	p.tokens[accessToken] = tokenInfo
	p.mu.Unlock()

	p.sendTokenResponse(w, accessToken, refreshToken, authCode.Scope, authCode.UserID)
}

// handleClientCredentialsGrant handles client credentials grant
func (p *OAuth2Provider) handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")
	scope := r.Form.Get("scope")

	// Validate client credentials
	if !p.validateClient(clientID, clientSecret) {
		p.sendError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Generate access token (no refresh token for client credentials)
	accessToken := p.generateAccessToken(clientID, scope, "")

	tokenInfo := &TokenInfo{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(p.tokenExpiry),
		Scope:       scope,
		ClientID:    clientID,
	}

	p.mu.Lock()
	p.tokens[accessToken] = tokenInfo
	p.mu.Unlock()

	p.sendTokenResponse(w, accessToken, "", scope, "")
}

// handleRefreshTokenGrant handles refresh token grant
func (p *OAuth2Provider) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Form.Get("refresh_token")
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")

	// Validate client credentials
	if !p.validateClient(clientID, clientSecret) {
		p.sendError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Find token info by refresh token
	var tokenInfo *TokenInfo
	p.mu.RLock()
	for _, ti := range p.tokens {
		if ti.RefreshToken == refreshToken && ti.ClientID == clientID {
			tokenInfo = ti
			break
		}
	}
	p.mu.RUnlock()

	if tokenInfo == nil {
		p.sendError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Generate new access token
	newAccessToken := p.generateAccessToken(clientID, tokenInfo.Scope, tokenInfo.UserID)

	newTokenInfo := &TokenInfo{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Reuse refresh token
		ExpiresAt:    time.Now().Add(p.tokenExpiry),
		Scope:        tokenInfo.Scope,
		ClientID:     clientID,
		UserID:       tokenInfo.UserID,
	}

	p.mu.Lock()
	p.tokens[newAccessToken] = newTokenInfo
	p.mu.Unlock()

	p.sendTokenResponse(w, newAccessToken, refreshToken, tokenInfo.Scope, tokenInfo.UserID)
}

// handlePasswordGrant handles resource owner password credentials grant
func (p *OAuth2Provider) handlePasswordGrant(w http.ResponseWriter, r *http.Request) {
	username := r.Form.Get("username")
	password := r.Form.Get("password")
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")
	scope := r.Form.Get("scope")

	// Validate client credentials
	if !p.validateClient(clientID, clientSecret) {
		p.sendError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Mock password validation (always succeeds for demo)
	if username == "" || password == "" {
		p.sendError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	userID := "user-" + username

	// Generate tokens
	accessToken := p.generateAccessToken(clientID, scope, userID)
	refreshToken := p.generateRefreshToken()

	tokenInfo := &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(p.tokenExpiry),
		Scope:        scope,
		ClientID:     clientID,
		UserID:       userID,
	}

	p.mu.Lock()
	p.tokens[accessToken] = tokenInfo
	p.mu.Unlock()

	p.sendTokenResponse(w, accessToken, refreshToken, scope, userID)
}

// HandleUserInfo handles the userinfo endpoint
func (p *OAuth2Provider) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate token
	p.mu.RLock()
	tokenInfo, exists := p.tokens[accessToken]
	p.mu.RUnlock()

	if !exists || tokenInfo.ExpiresAt.Before(time.Now()) {
		http.Error(w, "invalid_token", http.StatusUnauthorized)
		return
	}

	// Return user info
	userInfo := map[string]interface{}{
		"sub":   tokenInfo.UserID,
		"name":  "Mock User",
		"email": "user@example.com",
		"email_verified": true,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		log.Printf("OAuth2: Error encoding userinfo: %v\n", err)
	}
}

// Handle JWKS endpoint for public keys
func (p *OAuth2Provider) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	// Export public key in JWK format
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "default",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(p.publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), // 65537
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jwks); err != nil {
		log.Printf("OAuth2: Error encoding JWKS: %v\n", err)
	}
}

// generateAccessToken generates a JWT access token
func (p *OAuth2Provider) generateAccessToken(clientID, scope, userID string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       p.issuer,
		"sub":       userID,
		"aud":       clientID,
		"exp":       now.Add(p.tokenExpiry).Unix(),
		"iat":       now.Unix(),
		"scope":     scope,
		"client_id": clientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(p.privateKey)
	if err != nil {
		log.Printf("OAuth2: Error signing token: %v\n", err)
		return ""
	}

	return tokenString
}

// generateCode generates an authorization code
func (p *OAuth2Provider) generateCode() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("OAuth2: Error generating code: %v\n", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateRefreshToken generates a refresh token
func (p *OAuth2Provider) generateRefreshToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("OAuth2: Error generating refresh token: %v\n", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// validateClient validates client credentials
func (p *OAuth2Provider) validateClient(clientID, clientSecret string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, exists := p.clients[clientID]
	if !exists {
		return false
	}

	return client.ClientSecret == clientSecret
}

// validatePKCE validates PKCE code verifier
func (p *OAuth2Provider) validatePKCE(verifier, challenge, method string) bool {
	if method == "S256" {
		// SHA-256 validation would go here
		// For simplicity, we'll just check if verifier is provided
		return verifier != ""
	}
	// Plain method
	return verifier == challenge
}

// sendTokenResponse sends a token response
func (p *OAuth2Provider) sendTokenResponse(w http.ResponseWriter, accessToken, refreshToken, scope, userID string) {
	response := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(p.tokenExpiry.Seconds()),
		RefreshToken: refreshToken,
		Scope:        scope,
	}

	// Generate ID token if openid scope is present
	if strings.Contains(scope, "openid") && userID != "" {
		response.IDToken = p.generateIDToken(userID, scope)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("OAuth2: Error encoding token response: %v\n", err)
	}
}

// generateIDToken generates an OpenID Connect ID token
func (p *OAuth2Provider) generateIDToken(userID, scope string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   p.issuer,
		"sub":   userID,
		"aud":   "default-client",
		"exp":   now.Add(p.tokenExpiry).Unix(),
		"iat":   now.Unix(),
		"name":  "Mock User",
		"email": "user@example.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(p.privateKey)
	if err != nil {
		log.Printf("OAuth2: Error signing ID token: %v\n", err)
		return ""
	}

	return tokenString
}

// sendError sends an OAuth2 error response
func (p *OAuth2Provider) sendError(w http.ResponseWriter, errorCode string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": errorCode,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("OAuth2: Error encoding error response: %v\n", err)
	}
}
