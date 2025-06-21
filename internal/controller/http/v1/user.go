package v1

import (
	"net/http"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

type userRoutes struct {
	system     usecases.System
	jwtHandler *jwt.GinJWTMiddleware
}

func NewUserRoutes(
	public *gin.RouterGroup,
	private *gin.RouterGroup,
	jwtHandler *jwt.GinJWTMiddleware,
	system usecases.System,
) {
	r := &userRoutes{
		system:     system,
		jwtHandler: jwtHandler,
	}
	public.POST("/login", r.loginHandler)
	public.GET("/logout", r.logutHandler)

	private.GET("/refresh", r.refreshTokenHandler)
	private.GET("/user/list", r.getAllUser)
	private.POST("/user/password", r.updateUserPasswordHandler)

	private.POST("/user", r.newUserHandler)
	private.PUT("/user", r.updateUserHandler)
	private.DELETE("/user", r.deleteUserByUsername)
}

// loginHandler _.
//
//	@tags		User V1
//	@Summary	Login
//	@accept		application/json
//	@produce	application/json
//	@param		body	body		pb.LoginRequest	true	"Body"
//	@success	200		{object}	pb.LoginResponse
//	@success	401		{object}	pb.APIResponse
//	@router		/api/capitan/v1/login [post]
func (u *userRoutes) loginHandler(c *gin.Context) {
	u.jwtHandler.LoginHandler(c)
}

// logutHandler _.
//
//	@tags		User V1
//	@Summary	Logout
//	@accept		application/json
//	@produce	application/json
//	@success	200	{object}	emptypb.Empty
//	@router		/api/capitan/v1/logout [get]
func (u *userRoutes) logutHandler(c *gin.Context) {
	u.jwtHandler.LogoutHandler(c)
}

// refreshTokenHandler _.
//
//	@tags		User V1
//	@Summary	Refresh token
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@success	200	{object}	pb.LoginResponse
//	@failure	401	{object}	pb.APIResponse
//	@router		/api/capitan/v1/refresh [get]
func (u *userRoutes) refreshTokenHandler(c *gin.Context) {
	u.jwtHandler.RefreshHandler(c)
}

// getAllUser _.
//
//	@tags		User V1
//	@Summary	Get all user
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@success	200	{object}	pb.UserList
//	@failure	500	{object}	pb.APIResponse
//	@router		/api/capitan/v1/user/list [get]
func (u *userRoutes) getAllUser(c *gin.Context) {
	all, err := u.system.GetAllUser(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, all)
}

// newUserHandler _.
//
//	@tags		User V1
//	@Summary	New user
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@param		body	body		pb.User	true	"Body"
//	@success	200		{object}	emptypb.Empty
//	@failure	400		{object}	pb.APIResponse
//	@failure	500		{object}	pb.APIResponse
//	@router		/api/capitan/v1/user [post]
func (u *userRoutes) newUserHandler(c *gin.Context) {
	user := pb.User{}
	if err := c.Bind(&user); err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	if err := u.system.CreateUser(c, &user); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// deleteUserByUsername _.
//
//	@tags		User V1
//	@Summary	Delete user by username
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@param		body	body		pb.User	true	"Body"
//	@success	200		{object}	emptypb.Empty
//	@failure	400		{object}	pb.APIResponse
//	@failure	500		{object}	pb.APIResponse
//	@router		/api/capitan/v1/user [delete]
func (u *userRoutes) deleteUserByUsername(c *gin.Context) {
	user := pb.User{}
	if err := c.Bind(&user); err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	claims := jwt.ExtractClaims(c)
	if v, ok := claims["username"]; ok {
		deleteUsername, _ := v.(string)
		if deleteUsername == user.GetBasic().GetUsername() {
			resp.Fail(c, http.StatusForbidden, resp.ErrCannotDeleteSelf)
			return
		}
	}
	if err := u.system.DeleteUser(c, user.GetBasic().GetUsername()); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// updateUserHandler _.
//
//	@tags		User V1
//	@Summary	Update user except password
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@param		body	body		pb.User	true	"Body"
//	@success	200		{object}	emptypb.Empty
//	@failure	400		{object}	pb.APIResponse
//	@failure	500		{object}	pb.APIResponse
//	@router		/api/capitan/v1/user [put]
func (u *userRoutes) updateUserHandler(c *gin.Context) {
	body := pb.User{}
	if err := c.Bind(&body); err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	if err := u.system.UpdateUser(c, &body); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// updateUserPasswordHandler _.
//
//	@tags		User V1
//	@Summary	Update user password
//	@security	JWT
//	@accept		application/json
//	@produce	application/json
//	@param		body	body		pb.ChangePasswordRequest	true	"Body"
//	@success	200		{object}	emptypb.Empty
//	@failure	400		{object}	pb.APIResponse
//	@failure	500		{object}	pb.APIResponse
//	@router		/api/capitan/v1/user/password [post]
func (u *userRoutes) updateUserPasswordHandler(c *gin.Context) {
	body := pb.ChangePasswordRequest{}
	if err := c.Bind(&body); err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	if err := u.system.ChangePassword(c, body.GetUsername(), body.GetOldPassword(), body.GetNewPassword()); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}
