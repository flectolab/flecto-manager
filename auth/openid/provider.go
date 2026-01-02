package openid

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/flectolab/flecto-manager/config"
	"golang.org/x/oauth2"
)

type UserInfo struct {
	Subject   string
	Email     string
	FirstName string
	LastName  string
	Name      string
	Roles     []string
}

type Provider interface {
	GetAuthURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token, idToken *oidc.IDToken) (*UserInfo, error)
}

type provider struct {
	config       *config.OpenIDConfig
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

func NewProvider(ctx context.Context, cfg *config.OpenIDConfig) (Provider, error) {
	oidcProvider, err := oidc.NewProvider(ctx, cfg.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       scopes,
	}

	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &provider{
		config:       cfg,
		oidcProvider: oidcProvider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
	}, nil
}

func (p *provider) GetAuthURL(state string) string {
	return p.oauth2Config.AuthCodeURL(state)
}

func (p *provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.oauth2Config.Exchange(ctx, code)
}

func (p *provider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, rawIDToken)
}

func (p *provider) GetUserInfo(ctx context.Context, token *oauth2.Token, idToken *oidc.IDToken) (*UserInfo, error) {
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	userInfo := &UserInfo{
		Subject: idToken.Subject,
	}

	if email, ok := claims["email"].(string); ok {
		userInfo.Email = email
	}

	if name, ok := claims["name"].(string); ok {
		userInfo.Name = name
	}

	if givenName, ok := claims["given_name"].(string); ok {
		userInfo.FirstName = givenName
	}

	if familyName, ok := claims["family_name"].(string); ok {
		userInfo.LastName = familyName
	}

	if userInfo.FirstName == "" && userInfo.LastName == "" && userInfo.Name != "" {
		parts := strings.SplitN(userInfo.Name, " ", 2)
		if len(parts) >= 1 {
			userInfo.FirstName = parts[0]
		}
		if len(parts) >= 2 {
			userInfo.LastName = parts[1]
		}
	}

	if p.config.RolesClaim != "" {
		userInfo.Roles = extractRoles(claims, p.config.RolesClaim)
	}

	return userInfo, nil
}

func extractRoles(claims map[string]interface{}, path string) []string {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	current := claims

	for i, part := range parts {
		if i == len(parts)-1 {
			switch v := current[part].(type) {
			case []interface{}:
				result := make([]string, 0, len(v))
				for _, r := range v {
					if s, ok := r.(string); ok {
						result = append(result, s)
					}
				}
				return result
			case []string:
				return v
			}
		} else {
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return nil
			}
		}
	}
	return nil
}
