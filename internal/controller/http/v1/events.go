package v1

import (
	"net/http"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
)

type eventsRoutes struct {
	t usecases.Events
}

func NewEventsRoutes(handler *gin.RouterGroup, t usecases.Events) {
	r := &eventsRoutes{t}
	base := "/event"

	h := handler.Group(base)
	{
		h.GET("/login", r.loginEvents)
		h.GET("/shioaji", r.shioajiEvents)
	}
}

// loginEvents -.
//
//	@Tags		Event V1
//	@Summary	Get login events
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.LoginEventList
//	@Router		/api/capitan/v1/event/login [get]
func (r *eventsRoutes) loginEvents(c *gin.Context) {
	resp.Success(c, http.StatusOK, r.t.GetCurrentLoginEvent())
}

// shioajiEvents -.
//
//	@Tags		Event V1
//	@Summary	Get shioaji events
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.ShioajiEventList
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/event/shioaji [get]
func (r *eventsRoutes) shioajiEvents(c *gin.Context) {
	events, err := r.t.GetShioajiEvent()
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.ShioajiEventList{
		List: events,
	})
}
