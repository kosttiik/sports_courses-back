package app

import (
	"log"
	"net/http"
	"sports_courses/internal/app/ds"
	"sports_courses/internal/app/role"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

const jwtPrefix = "Bearer "

func (a *Application) WithAuthCheck(assignedRoles ...role.Role) func(context *gin.Context) {
	return func(c *gin.Context) {
		jwtStr := c.GetHeader("Authorization")

		if jwtStr == "" {
			var cookieErr error
			jwtStr, cookieErr = c.Cookie("sports_courses-api-token")
			if cookieErr != nil {
				c.AbortWithStatus(http.StatusBadRequest)
			}
		}

		if !strings.HasPrefix(jwtStr, jwtPrefix) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		jwtStr = jwtStr[len(jwtPrefix):]
		log.Println(jwtStr)

		err := a.redis.CheckJWTInBlackList(c.Request.Context(), jwtStr)

		if err == nil { // значит что токен в блеклисте
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		token, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte("test"), nil
		})
		if err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			log.Println(err)

			return
		}

		myClaims := token.Claims.(*ds.JWTClaims)

		isAssigned := false

		for _, oneOfAssignedRole := range assignedRoles {
			if myClaims.Role == oneOfAssignedRole {
				isAssigned = true
				break
			}
		}

		if !isAssigned {
			c.AbortWithStatus(http.StatusForbidden)
			log.Printf("role %d is not assigned in %d", myClaims.Role, assignedRoles)

			return
		}

		c.Set("role", myClaims.Role)
		c.Set("userUUID", myClaims.UserUUID)
	}
}
