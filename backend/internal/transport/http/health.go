package httpapi

import (
	"log/slog"
	"net/http"

	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type healthHandler struct {
	health *services.HealthService
	log    *slog.Logger
}

// check godoc
//
//	@Summary	Liveness and database check
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	dto.HealthResponse
//	@Failure	503	{object}	dto.ErrorResponse
//	@Router		/healthcheck [get]
func (h healthHandler) check(c *gin.Context) {
	if err := h.health.Check(c.Request.Context()); err != nil {
		h.log.Error("health check failed", "error", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "the service is not ready"})
		return
	}
	c.JSON(http.StatusOK, dto.HealthResponse{Status: "ok"})
}
