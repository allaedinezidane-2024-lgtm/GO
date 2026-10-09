package helper

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"latihan-fiber/app/model"
)

type JWTManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

type accessClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret, issuer string, accessTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
	}
}

func (j *JWTManager) AccessTTL() time.Duration {
	return j.accessTTL
}

func (j *JWTManager) GenerateAccess(user model.AuthUser) (string, error) {
	now := time.Now()

	claims := accessClaims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(user.ID),
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(j.secret)
}

func (j *JWTManager) Parse(tokenString string) (model.AuthUser, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&accessClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return j.secret, nil
		},
		jwt.WithIssuer(j.issuer),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return model.AuthUser{}, err
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return model.AuthUser{}, errors.New("invalid token")
	}

	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return model.AuthUser{}, errors.New("invalid user id")
	}

	return model.AuthUser{
		ID:       id,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

func ExtractBearerToken(header string) (string, error) {
	parts := strings.Fields(header)

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}

	return parts[1], nil
}
