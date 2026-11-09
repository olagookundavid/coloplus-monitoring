package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func toUserDetail(v services.UserView) dto.UserDetailDTO {
	ids := v.SiteIDs
	if ids == nil {
		ids = []string{}
	}
	return dto.UserDetailDTO{UserDTO: toUserDTO(v.User), SiteIDs: ids, Sites: []dto.SiteRefDTO{}}
}

// createUser godoc
//
//	@Summary		Create a user (administrators)
//	@Description	Leave the password out and the person is emailed a link to choose their own.
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.CreateUserRequest	true	"New user"
//	@Success	201		{object}	dto.UserDetailDTO
//	@Failure	400		{object}	dto.ErrorResponse
//	@Failure	403		{object}	dto.ErrorResponse
//	@Failure	409		{object}	dto.ErrorResponse
//	@Router		/api/v1/users [post]
func (a *api) createUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	view, err := a.users.Create(c.Request.Context(), actorFrom(c), services.CreateUserInput{
		Email: req.Email, FullName: req.FullName, Role: domain.Role(req.Role), Password: req.Password, SiteIDs: req.SiteIDs,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	one := []dto.UserDetailDTO{toUserDetail(view)}
	a.nameUserSites(c, one)
	c.JSON(http.StatusCreated, one[0])
}

// listUsers godoc
//
//	@Summary	List users (administrators)
//	@Tags		users
//	@Produce	json
//	@Success	200	{object}	dto.UserListResponse
//	@Router		/api/v1/users [get]
func (a *api) listUsers(c *gin.Context) {
	views, err := a.users.List(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	out := make([]dto.UserDetailDTO, len(views))
	for i, v := range views {
		out[i] = toUserDetail(v)
	}
	a.nameUserSites(c, out)
	c.JSON(http.StatusOK, dto.UserListResponse{Users: out})
}

// getUser godoc
//
//	@Summary	One user with site assignments (administrators)
//	@Tags		users
//	@Produce	json
//	@Param		id	path		string	true	"User id"
//	@Success	200	{object}	dto.UserDetailDTO
//	@Failure	404	{object}	dto.ErrorResponse
//	@Router		/api/v1/users/{id} [get]
func (a *api) getUser(c *gin.Context) {
	view, err := a.users.Get(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	one := []dto.UserDetailDTO{toUserDetail(view)}
	a.nameUserSites(c, one)
	c.JSON(http.StatusOK, one[0])
}

// updateUser godoc
//
//	@Summary		Update a user (administrators)
//	@Description	Only the fields present are changed. site_ids replaces the whole assignment list.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"User id"
//	@Param			body	body		dto.UpdateUserRequest	true	"Changes"
//	@Success		200		{object}	dto.UserDetailDTO
//	@Router			/api/v1/users/{id} [patch]
func (a *api) updateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	in := services.UpdateUserInput{FullName: req.FullName, Active: req.Active, Password: req.Password, SiteIDs: req.SiteIDs}
	if req.Role != nil {
		role := domain.Role(*req.Role)
		in.Role = &role
	}
	view, err := a.users.Update(c.Request.Context(), actorFrom(c), c.Param("id"), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	one := []dto.UserDetailDTO{toUserDetail(view)}
	a.nameUserSites(c, one)
	c.JSON(http.StatusOK, one[0])
}

// logoutEveryone godoc
//
//	@Summary		Sign everybody out
//	@Description	Ends every session at once, the caller's included: all of them sign in again. Needs user management; recorded in the audit log.
//	@Tags			users
//	@Success		200	{object}	dto.LogoutAllResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/logout-all [post]
func (a *api) logoutEveryone(c *gin.Context) {
	n, err := a.users.LogoutEveryone(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.LogoutAllResponse{Accounts: n})
}

// resetUserMFA godoc
//
//	@Summary		Reset a person's two-step verification
//	@Description	For someone who lost both their phone and their recovery codes. Turns it off, deletes their recovery codes, signs them out everywhere and emails them; they set it up again at their next sign-in. You cannot reset your own.
//	@Tags			users
//	@Param			id	path	string	true	"User id"
//	@Success		204
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/users/{id}/reset-mfa [post]
func (a *api) resetUserMFA(c *gin.Context) {
	if err := a.users.ResetMFA(c.Request.Context(), actorFrom(c), c.Param("id")); err != nil {
		a.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
