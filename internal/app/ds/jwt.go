package ds

import (
	"sports_courses/internal/app/role"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type JWTClaims struct {
	jwt.StandardClaims           // все что точно необходимо по RFC
	UserUUID           uuid.UUID `json:"user_uuid"`            // наши данные - uuid этого пользователя в базе данных
	Scopes             []string  `json:"scopes" json:"scopes"` // список доступов в нашей системе
	Role               role.Role
}
