package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func toAttachmentDTO(a domain.Attachment, staff bool) dto.AttachmentDTO {
	out := dto.AttachmentDTO{
		ID: a.ID, OwnerType: string(a.OwnerType), OwnerID: a.OwnerID, SiteID: a.SiteID, Filename: a.Filename,
		ContentType: a.ContentType, SizeBytes: a.SizeBytes, SHA256: a.SHA256, Status: string(a.Status),
		Party: string(a.Party), CreatedAt: a.CreatedAt, ConfirmedAt: a.ConfirmedAt,
	}
	if staff {
		out.Internal, out.UploadedBy, out.ScanStatus = a.Internal, a.UploadedBy, a.ScanStatus
	}
	return out
}

// requestUpload godoc
//
//	@Summary		Ask for a link to upload one file
//	@Description	Step 1 of 3. Say what the file is and what it belongs to; the answer is a short-lived link. Step 2: PUT the file to that link with exactly the headers given. Step 3: POST /attachments/{id}/complete. Accepted: JPEG, PNG, WebP, PDF, plain text, CSV, Word and Excel files, up to the size limit, at most 25 per ticket, incident, dispute or acceptance. The file goes straight to private storage and is checked against the SHA-256 you give. A file is only visible once confirmed. 503 when file storage is not set up.
//	@Tags			attachments
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.RequestUploadRequest	true	"File"
//	@Success		201		{object}	dto.RequestUploadResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		503		{object}	dto.ErrorResponse
//	@Router			/api/v1/attachments [post]
func (a *api) requestUpload(c *gin.Context) {
	var req dto.RequestUploadRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	up, err := a.attachments.Request(c.Request.Context(), actor, services.AttachmentInput{
		OwnerType: domain.AttachmentOwner(req.OwnerType), OwnerID: req.OwnerID, Filename: req.Filename,
		ContentType: req.ContentType, SizeBytes: req.SizeBytes, SHA256: req.SHA256, Shared: req.Shared,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.RequestUploadResponse{
		Attachment: toAttachmentDTO(up.Attachment, a.attachments.IsStaff(actor)),
		Upload: dto.UploadTarget{
			Method: up.Upload.Method, URL: up.Upload.URL, Headers: up.Upload.Headers, ExpiresAt: up.Upload.ExpiresAt,
		},
	})
}

// completeUpload godoc
//
//	@Summary		Confirm an upload
//	@Description	Step 3 of 3. Checks the stored file is the size and SHA-256 declared; if not it is discarded and you start again. Repeating it is safe. Only the sender can confirm.
//	@Tags			attachments
//	@Produce		json
//	@Param			id	path		string	true	"Attachment id"
//	@Success		200	{object}	dto.AttachmentDTO
//	@Failure		400	{object}	dto.ErrorResponse
//	@Router			/api/v1/attachments/{id}/complete [post]
func (a *api) completeUpload(c *gin.Context) {
	actor := actorFrom(c)
	att, err := a.attachments.Complete(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toAttachmentDTO(att, a.attachments.IsStaff(actor)))
}

// listAttachments godoc
//
//	@Summary		Files on a ticket, incident, dispute or acceptance
//	@Description	Received files only. Coloplus's internal files are left out for Airtel. For acceptance, owner_id is the site's id.
//	@Tags			attachments
//	@Produce		json
//	@Param			owner_type	query		string	true	"ticket, incident, dispute, report or acceptance"
//	@Param			owner_id	query		string	true	"Its id"
//	@Success		200			{object}	dto.AttachmentListResponse
//	@Router			/api/v1/attachments [get]
func (a *api) listAttachments(c *gin.Context) {
	actor := actorFrom(c)
	list, err := a.attachments.List(c.Request.Context(), actor, domain.AttachmentOwner(c.Query("owner_type")), c.Query("owner_id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	staff := a.attachments.IsStaff(actor)
	resp := dto.AttachmentListResponse{Attachments: make([]dto.AttachmentDTO, 0, len(list))}
	for _, att := range list {
		resp.Attachments = append(resp.Attachments, toAttachmentDTO(att, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// downloadAttachment godoc
//
//	@Summary		A link to read one file
//	@Description	Checks you may see the file and returns a link that works for a couple of minutes and makes the browser save it, never display it. Every link issued is recorded in the audit log. Open the link at once; do not store it.
//	@Tags			attachments
//	@Produce		json
//	@Param			id	path		string	true	"Attachment id"
//	@Success		200	{object}	dto.DownloadResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/attachments/{id}/download [get]
func (a *api) downloadAttachment(c *gin.Context) {
	link, err := a.attachments.Download(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.DownloadResponse{URL: link.URL, ExpiresAt: link.ExpiresAt, Filename: link.Filename})
}
