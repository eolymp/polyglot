package oauth

type GrantType string

const (
	PasswordGrant          GrantType = "password"
	RefreshTokenGrant      GrantType = "refresh_token"
	AuthorizationCodeGrant GrantType = "authorization_code"
)
