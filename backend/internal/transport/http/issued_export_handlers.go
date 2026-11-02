package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

func toIssuedDTO(e domain.IssuedExport) dto.IssuedExportDTO {
	return dto.IssuedExportDTO{
		ID: e.ID, Kind: e.Kind, Filename: e.Filename, Params: e.Params, RowCount: e.RowCount, SizeBytes: e.SizeBytes,
		SHA256: e.SHA256, Stored: e.Stored(), IssuedBy: e.IssuedBy, CreatedAt: e.CreatedAt,
	}
}

// listIssuedExports godoc
//
//	@Summary		Exports that were issued
//	@Description	Every CSV export handed out, newest first, with the request that made it, its row count and SHA-256. Each export response carries its id (X-Export-Id) and digest (X-Content-SHA256), so a file can be checked against this register. You see your own; someone who can read the audit log sees everyone's. Records are never changed or removed.
//	@Tags			exports
//	@Produce		json
//	@Param			limit	query		int	false	"Page size (default 50, max 200)"
//	@Param			offset	query		int	false	"Offset"
//	@Success		200		{object}	dto.IssuedExportListResponse
//	@Router			/api/v1/exports/issued [get]
func (a *api) listIssuedExports(c *gin.Context) {
	l, err := a.exports.Issued(c.Request.Context(), actorFrom(c), intQuery(c, "limit"), intQuery(c, "offset"))
	if err != nil {
		a.fail(c, err)
		return
	}
	resp := dto.IssuedExportListResponse{Exports: make([]dto.IssuedExportDTO, 0, len(l.Items)), Total: l.Total, Limit: l.Limit, Offset: l.Offset}
	for _, e := range l.Items {
		resp.Exports = append(resp.Exports, toIssuedDTO(e))
	}
	c.JSON(http.StatusOK, resp)
}

// downloadIssuedExport godoc
//
//	@Summary		A link to the exact file that was issued
//	@Description	Returns a short-lived link to the stored copy of an issued export, byte for byte what was sent. Only the person who issued it, or someone who can read the audit log, may have it; it is not found if the file was not stored.
//	@Tags			exports
//	@Produce		json
//	@Param			id	path		string	true	"Issued export id"
//	@Success		200	{object}	dto.DownloadResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/issued/{id}/download [get]
func (a *api) downloadIssuedExport(c *gin.Context) {
	url, exp, e, err := a.exports.IssuedLink(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.DownloadResponse{URL: url, ExpiresAt: exp, Filename: e.Filename})
}
