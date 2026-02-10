// Package client provides HTTP client functionality with DNS timing
package client

import (
	"fmt"
	"net/http"

	"moxapp/internal/config"
)

// EnvGetter provides access to environment variables
type EnvGetter interface {
	GetEnv(key string) string
}

// ApplyAuth applies authentication to a request using resolved AuthConfig
func ApplyAuth(req *http.Request, authCfg *config.AuthConfig, tokenMgr *TokenManager) error {
	if authCfg == nil || authCfg.Type == config.AuthTypeNone {
		return nil
	}

	ctx := req.Context()

	switch authCfg.Type {
	case config.AuthTypeBearer:
		token, err := tokenMgr.GetToken(ctx, authCfg.Name)
		if err != nil {
			return fmt.Errorf("failed to get bearer token: %w", err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

	case config.AuthTypeAPIKey:
		token, err := tokenMgr.GetToken(ctx, authCfg.Name)
		if err != nil {
			return fmt.Errorf("failed to get api key: %w", err)
		}
		if token != "" {
			req.Header.Set(authCfg.HeaderName, token)
		}

	case config.AuthTypeAPIKeyQuery:
		token, err := tokenMgr.GetToken(ctx, authCfg.Name)
		if err != nil {
			return fmt.Errorf("failed to get api key: %w", err)
		}
		if token != "" {
			q := req.URL.Query()
			q.Set(authCfg.QueryParam, token)
			req.URL.RawQuery = q.Encode()
		}

	case config.AuthTypeBasic:
		username := tokenMgr.GetEnv(authCfg.UsernameEnv)
		password := tokenMgr.GetEnv(authCfg.PasswordEnv)
		if username != "" || password != "" {
			req.SetBasicAuth(username, password)
		}

	case config.AuthTypeCustom:
		token, err := tokenMgr.GetToken(ctx, authCfg.Name)
		if err != nil {
			return fmt.Errorf("failed to get custom token: %w", err)
		}
		if token != "" {
			req.Header.Set(authCfg.HeaderName, token)
		}

	default:
		return fmt.Errorf("unsupported auth type: %s", authCfg.Type)
	}

	return nil
}
