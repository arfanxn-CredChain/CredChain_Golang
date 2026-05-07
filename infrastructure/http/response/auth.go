package response

type Auth struct {
	User                  `json:",inline"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresIn  int    `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
}

func NewAuth(user User, accessToken string, refreshToken string, accessTokenExpiresIn int, refreshTokenExpiresIn int) Auth {
	return Auth{
		User:                  user,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresIn:  accessTokenExpiresIn,
		RefreshTokenExpiresIn: refreshTokenExpiresIn,
		TokenType:             "Bearer",
	}
}