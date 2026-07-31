package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type GoogleProvider struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewGoogleProvider(ctx context.Context, clientID, clientSecret, redirectURL string) (*GoogleProvider, error) {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, nil
	}
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	return &GoogleProvider{oauth: oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, Endpoint: provider.Endpoint(), RedirectURL: redirectURL, Scopes: []string{oidc.ScopeOpenID, "email", "profile"}}, verifier: provider.Verifier(&oidc.Config{ClientID: clientID})}, nil
}
func randomURLToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (g *GoogleProvider) AuthCodeURL(state, nonce string) string {
	return g.oauth.AuthCodeURL(state, oidc.Nonce(nonce))
}
func (g *GoogleProvider) Verify(ctx context.Context, code, nonce string) (GoogleIdentity, error) {
	token, err := g.oauth.Exchange(ctx, code)
	if err != nil {
		return GoogleIdentity{}, err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return GoogleIdentity{}, ErrInvalidCredentials
	}
	idToken, err := g.verifier.Verify(ctx, raw)
	if err != nil {
		return GoogleIdentity{}, err
	}
	if idToken.Nonce != nonce {
		return GoogleIdentity{}, ErrInvalidCredentials
	}
	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return GoogleIdentity{}, err
	}
	return GoogleIdentity{Subject: claims.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified}, nil
}
