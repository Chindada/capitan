package v1

import (
	"net/http"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
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
