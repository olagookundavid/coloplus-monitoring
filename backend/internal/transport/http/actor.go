package httpapi

import (
	"log/slog"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/services"

	"pkg-common/middleware"

	"github.com/gin-gonic/gin"
)

const actorKey = "actor"

// loadActor turns the verified token into the signed-in Actor. It runs on
// every authenticated request, so deactivating a user cuts access at once.
func loadActor(auth *services.AuthService, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, ok := middleware.PayloadFrom(c)
		if !ok {
			respondError(c, log, apperrors.ErrUnauthorized)
			return
		}
		actor, err := auth.Authenticate(c.Request.Context(), payload.UserID)
		if err != nil {
			respondError(c, log, err)
			return
		}
		c.Set(actorKey, actor)
		c.Next()
	}
}

func actorFrom(c *gin.Context) domain.Actor {
	v, _ := c.Get(actorKey)
	actor, _ := v.(domain.Actor)
	return actor
}

// abortWith writes err using the shared mapper. Handlers use it so every
// failure takes the same path.
func abortWith(c *gin.Context, err error) {
	respondError(c, nil, err)
}
