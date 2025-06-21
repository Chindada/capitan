// Package resp package resp
package resp

import (
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const acceptHeader = "Accept"

func Fail(c *gin.Context, code int, err error) {
	fn := c.JSON
	if c.Request.Header.Get(acceptHeader) == binding.MIMEPROTOBUF {
		fn = c.ProtoBuf
	}
	resp := &pb.APIResponse{}
	switch v := err.(type) {
	case *repo.Error:
		resp.Code = v.Code
		resp.Response = v.Message
	case *usecases.UseCaseError:
		resp.Code = v.Code
		resp.Response = v.Message
	case *APIError:
		resp.Code = v.Code
		resp.Response = v.Message
	case error:
		resp.Code = -1
		resp.Response = v.Error()
	}
	fn(code, resp)
	c.Abort()
}

func Success(c *gin.Context, code int, data any) {
	fn := c.JSON
	if c.Request.Header.Get(acceptHeader) == binding.MIMEPROTOBUF {
		fn = c.ProtoBuf
	}
	fn(code, data)
}
