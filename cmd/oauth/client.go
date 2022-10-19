package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/eolymp/go-packages/httpx"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidToken       = errors.New("oauth: token is invalid")
	ErrInvalidCredentials = errors.New("oauth: invalid credentials")
	ErrParsingResponse    = errors.New("oauth: unable to parse response")
	ErrEmptyEndpoint      = errors.New("oauth: endpoint is not configured")
	ErrBadRequestError    = errors.New("oauth: bad request error")
	ErrServerError        = errors.New("oauth: server error")
)

// Client provides mechanism to fetch OAuth token and scopes
type Client struct {
	url      string
	cache    cacheClient
	cacheTTL time.Duration
	client   httpx.Client
}

// NewClient for OAuth
func NewClient(endpoint string, opts ...Option) *Client {
	cli := &Client{
		url:      strings.TrimSuffix(endpoint, "/"),
		cache:    nopCache{},
		cacheTTL: 10 * time.Minute,
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli
}

type CreateTokenInput struct {
	GrantType    GrantType
	Username     string
	Password     string
	ClientId     string
	ClientSecret string
	Code         string
	CodeVerifier string
	Scope        string
	RefreshToken string
}

type CreateTokenOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// CreateToken requests a new OAuth token
func (c *Client) CreateToken(ctx context.Context, in *CreateTokenInput) (*CreateTokenOutput, error) {
	if c.url == "" {
		return nil, ErrEmptyEndpoint
	}

	query := url.Values{
		"grant_type":    []string{string(in.GrantType)},
		"username":      []string{in.Username},
		"password":      []string{in.Password},
		"client_id":     []string{in.ClientId},
		"client_secret": []string{in.ClientSecret},
		"code":          []string{in.Code},
		"code_verifier": []string{in.CodeVerifier},
		"scope":         []string{in.Scope},
		"refresh_token": []string{in.RefreshToken},
	}

	req, err := http.NewRequest(http.MethodPost, c.url+"/oauth/token", strings.NewReader(query.Encode()))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrInvalidCredentials
	}

	if resp.StatusCode/100 == 4 {
		return nil, ErrBadRequestError
	}

	if resp.StatusCode/100 == 5 {
		return nil, ErrServerError
	}

	out := &CreateTokenOutput{}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, ErrParsingResponse
	}

	return out, nil
}

type IntrospectTokenInput struct {
	Token string
}

type IntrospectTokenOutput struct {
	Active    bool   `json:"active"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Expires   int64  `json:"exp"`
	Scope     string `json:"scope"`
	TokenType string `json:"token_type"`
	JTI       string `json:"jti"`
}

// IntrospectToken requests information about token from OAuth server
func (c *Client) IntrospectToken(ctx context.Context, in *IntrospectTokenInput) (*IntrospectTokenOutput, error) {
	key := "/oauth/introspect/" + in.Token
	out := &IntrospectTokenOutput{}

	if c.cache.ShouldGet(key, &out) {
		return out, nil
	}

	if c.url == "" {
		return nil, ErrEmptyEndpoint
	}

	query := url.Values{
		"token": []string{in.Token},
	}

	req, err := http.NewRequest(http.MethodGet, c.url+"/oauth/introspect", strings.NewReader(query.Encode()))

	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrInvalidToken
	}

	if resp.StatusCode/100 == 4 {
		return nil, ErrBadRequestError
	}

	if resp.StatusCode/100 == 5 {
		return nil, ErrServerError
	}

	if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, ErrParsingResponse
	}

	c.cache.ShouldSet(key, out, c.cacheTTL)

	return out, nil
}

func (c *Client) AuthenticateHTTP(ctx context.Context, kind, cred string) (context.Context, error) {
	out, err := c.IntrospectToken(ctx, &IntrospectTokenInput{Token: cred})
	if err == ErrInvalidToken {
		return ctx, nil
	}

	if err != nil {
		return ctx, err
	}

	token := &Token{
		ID:      out.JTI,
		Active:  out.Active,
		Expires: time.Unix(out.Expires, 0),
		Scopes:  strings.Split(out.Scope, " "),
		Identity: &Identity{
			UserID:   out.UserID,
			Username: out.Username,
		},
	}

	if !token.Valid() {
		return ctx, nil
	}

	return ContextWithToken(ctx, token), nil
}
