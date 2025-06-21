// Package router implements routing paths. Each services in own file.
package router

import (
	"fmt"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/chindada/capitan/docs"
	"github.com/chindada/capitan/internal/controller/http/auth"
	"github.com/chindada/capitan/internal/controller/http/resp"
	v1 "github.com/chindada/capitan/internal/controller/http/v1"
	"github.com/chindada/capitan/internal/controller/http/ws"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/capitan/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	prefix = "/api/capitan"
	wsPath = "/ws/capitan"
)

var swagHandler gin.HandlerFunc

// Router -.
type Router struct {
	rootHandler *gin.Engine
	v1WSGroup   *gin.RouterGroup
	v1Group     *gin.RouterGroup
	jwtHandler  *jwt.GinJWTMiddleware
}

// NewRouter -.
//
//	@title						Capitan V1 OpenAPI
//	@description				Capitan V1 Srv's API docs
//	@version					v0.0
//	@securityDefinitions.apikey	JWT
//	@in							header
//	@name						Authorization
func NewRouter(system usecases.System) *Router {
	g := gin.New()
	g.Use(gin.Recovery())
	g.GET("/metrics", gin.WrapH(promhttp.Handler()))

	jwtHandler, err := auth.NewAuthMiddleware(system, time.Hour*8)
	if err != nil {
		panic(err)
	}

	root := g.Group(prefix)
	root.GET("/version", func(c *gin.Context) {
		resp.Success(c, http.StatusOK, structpb.NewStringValue(version.GetCore().GetVersion()))
	})
	if swagHandler != nil {
		ir := root.GET("/swagger/*any", swagHandler)
		ir.Use(func(c *gin.Context) {
			docs.SwaggerInfo.Host = c.Request.Host
			c.Next()
		})
		root.GET("/docs", func(c *gin.Context) {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/swagger/index.html", root.BasePath()))
		})
	} else {
		root.StaticFS("/docs", http.FS(docs.IndexHTML))
	}

	v1Prefix := fmt.Sprintf("%s/v1", prefix)
	v1WSPrefix := fmt.Sprintf("%s/v1", wsPath)

	v1Public := g.Group(v1Prefix)
	v1WSGroup := g.Group(v1WSPrefix)
	v1WSGroup.GET("/health", func(c *gin.Context) {
		forwardChan := make(chan []byte)
		ws, wsErr := ws.New(c, forwardChan)
		if wsErr != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
		ws.ReadMessage()
	})
	v1Private := g.Group(v1Prefix)
	v1WSGroup.Use(jwtHandler.MiddlewareFunc())
	v1Private.Use(jwtHandler.MiddlewareFunc())

	v1.NewUserRoutes(v1Public, v1Private, jwtHandler, system)

	return &Router{
		rootHandler: g,
		v1WSGroup:   v1WSGroup,
		v1Group:     v1Private,
		jwtHandler:  jwtHandler,
	}
}

func (r *Router) AddV1BasicRoutes(basic usecases.Basic) *Router {
	v1.NewBasicRoutes(r.v1Group, basic)
	return r
}

func (r *Router) AddV1StreamRoutes(stream usecases.Stream) *Router {
	v1.NewStreamRoutes(r.v1WSGroup, stream)
	return r
}

func (r *Router) AddV1SystemRoutes() *Router {
	v1.NewSystemRoutes(r.v1Group)
	return r
}

func (r *Router) GetHandler() *gin.Engine {
	return r.rootHandler
}
