package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
)

func writeError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.AbortWithStatusJSON(appErr.StatusCode, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: appErr.Code, Message: appErr.Message}})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "An unexpected error occurred."}})
}

func WriteErrorShim(c *gin.Context, err error) {
	writeError(c, err)
}
