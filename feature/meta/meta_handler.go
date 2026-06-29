package meta

import (
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type MetaHandler interface {
	Get(c *gin.Context)
}

type metaHandler struct {
	svc MetaService
}

type MetaHandlerParams struct {
	fx.In
	Svc MetaService
}

func NewMetaHandler(p MetaHandlerParams) MetaHandler {
	return &metaHandler{svc: p.Svc}
}

func (h *metaHandler) Get(c *gin.Context) {
	meta, err := h.svc.Get(c.Request.Context())
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeMetaSuccess, meta)
}
