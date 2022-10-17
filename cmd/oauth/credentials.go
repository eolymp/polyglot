package oauth

import (
	"context"
	"fmt"
)

type CredentialsFunc func(context.Context) (string, error)

func (f CredentialsFunc) Credentials(ctx context.Context) (string, error) {
	return f(ctx)
}

func PasswordCredentials(cli *Client, username, password string) CredentialsFunc {
	return func(ctx context.Context) (string, error) {
		out, err := cli.CreateToken(ctx, &CreateTokenInput{
			GrantType: PasswordGrant,
			Username:  username,
			Password:  password,
		})

		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Bearer %v", out.AccessToken), nil
	}
}

func RefreshTokenCredentials(cli *Client, token string) CredentialsFunc {
	return func(ctx context.Context) (string, error) {
		out, err := cli.CreateToken(ctx, &CreateTokenInput{
			GrantType:    RefreshTokenGrant,
			RefreshToken: token,
		})

		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Bearer %v", out.AccessToken), nil
	}
}
