// Package auth package auth
package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	v4jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	tokenHeaderName = "Bearer"
	identityKey     = "capitan_identity"
)

func NewAuthMiddleware(system usecases.System, expired time.Duration) (*jwt.GinJWTMiddleware, error) {
	key, err := system.GetLastJWT(context.Background())
	if err != nil {
		return nil, err
	} else if key == "" {
		key = uuid.New().String()
		err = system.InsertJWT(context.Background(), key)
		if err != nil {
			return nil, err
		}
	}

	m := jwt.GinJWTMiddleware{
		TokenLookup:      "header:Authorization, query:token",
		SigningAlgorithm: "HS256",
		Timeout:          expired,
		TimeFunc:         time.Now,
		TokenHeadName:    tokenHeaderName,
		Authorizator: func(any, *gin.Context) bool {
			return true
		},
		Unauthorized:          unauthorized,
		LoginResponse:         loginResponse,
		LogoutResponse:        logoutResponse,
		RefreshResponse:       refreshResponse,
		IdentityKey:           identityKey,
		IdentityHandler:       identityHandler,
		HTTPStatusMessageFunc: hTTPStatusMessageFunc,
		Realm:                 "capitan_jwt",
		CookieMaxAge:          expired,
		CookieName:            "capitan",

		Key:           []byte(key),
		MaxRefresh:    expired,
		Authenticator: authenticator(system),
		PayloadFunc:   payloadFunc,

		// PrivKeyFile:          "",
		// PrivKeyBytes:         []byte{},
		// PubKeyFile:           "",
		// PrivateKeyPassphrase: "",
		// PubKeyBytes:          []byte{},
		// CookieDomain:      "",
		// SendCookie:        false,
		// SecureCookie:      false,
		// CookieHTTPOnly:    false,
		// SendAuthorization: false,
		// DisabledAbort:     false,
		// CookieSameSite:    1,

		ParseOptions: []v4jwt.ParserOption{},
	}
	return jwt.New(&m)
}

func hTTPStatusMessageFunc(e error, _ *gin.Context) string {
	return e.Error()
}

func unauthorized(c *gin.Context, code int, message string) {
	resp.Success(c, code, &pb.APIResponse{
		Code:     int64(code),
		Response: message,
	})
}

func loginResponse(c *gin.Context, code int, token string, expire time.Time) {
	resp.Success(c, http.StatusOK, &pb.LoginResponse{
		Token:  fmt.Sprintf("%s %s", tokenHeaderName, token),
		Expire: expire.Format(time.RFC3339),
		Code:   int64(code),
	})
}

func logoutResponse(c *gin.Context, code int) {
	resp.Success(c, code, &emptypb.Empty{})
}

func refreshResponse(c *gin.Context, code int, token string, expire time.Time) {
	resp.Success(c, http.StatusOK, &pb.LoginResponse{
		Token:  fmt.Sprintf("%s %s", tokenHeaderName, token),
		Expire: expire.Format(time.RFC3339),
		Code:   int64(code),
	})
}

func identityHandler(c *gin.Context) any {
	claims := jwt.ExtractClaims(c)
	return claims[identityKey]
}

func authenticator(system usecases.System) func(c *gin.Context) (any, error) {
	return func(c *gin.Context) (any, error) {
		var loginVals pb.LoginRequest
		err := c.Bind(&loginVals)
		if err != nil {
			return nil, jwt.ErrMissingLoginValues
		}
		user, loginErr := system.Login(c, &loginVals)
		if loginErr != nil {
			return nil, loginErr
		}
		return user, nil
	}
}

func payloadFunc(data any) jwt.MapClaims {
	if v, ok := data.(*pb.User); ok {
		return jwt.MapClaims{
			"username": v.GetBasic().GetUsername(),
			"user_id":  v.GetId(),
		}
	}
	return nil
}
