package httpapi

import (
	"net/http"
	"time"

	"coloplus-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

// CookieConfig describes the browser cookie that carries the access token.
type CookieConfig struct {
	Name     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

func (a *api) setAuthCookie(c *gin.Context, token string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: a.cookie.Name, Value: token, Path: "/", Domain: a.cookie.Domain,
		Expires: expires, HttpOnly: true, Secure: a.cookie.Secure, SameSite: a.cookie.SameSite,
	})
}

func (a *api) clearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: a.cookie.Name, Value: "", Path: "/", Domain: a.cookie.Domain,
		MaxAge: -1, HttpOnly: true, Secure: a.cookie.Secure, SameSite: a.cookie.SameSite,
	})
}

// login godoc
//
//	@Summary		Sign in
//	@Description	Checks the password and, when two-step verification is on, the authenticator code. On success the access token is set as an HttpOnly cookie and also returned. When a code is needed the response status is "mfa_required" and no token is issued: send the request again with totp_code.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"Credentials"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		429		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/login [post]
func (a *api) login(c *gin.Context) {
	var req dto.LoginRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	res, err := a.auth.Login(c.Request.Context(), req.Email, req.Password, req.TOTPCode)
	if err != nil {
		a.fail(c, err)
		return
	}
	if res.MFARequired {
		c.JSON(http.StatusOK, dto.LoginResponse{Status: "mfa_required"})
		return
	}
	a.setAuthCookie(c, res.Token, res.ExpiresAt)
	u := toUserDTO(res.User)
	c.JSON(http.StatusOK, dto.LoginResponse{Status: "ok", AccessToken: res.Token, ExpiresAt: &res.ExpiresAt, User: &u})
}

// logout godoc
//
//	@Summary	Sign out
//	@Tags		auth
//	@Accept		json
//	@Success	204
//	@Router		/api/v1/auth/logout [post]
func (a *api) logout(c *gin.Context) {
	a.clearAuthCookie(c)
	c.Status(http.StatusNoContent)
}

// me godoc
//
//	@Summary	The signed-in user, their permissions and site scope
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	dto.MeResponse
//	@Failure	401	{object}	dto.ErrorResponse
//	@Router		/api/v1/auth/me [get]
func (a *api) me(c *gin.Context) {
	actor := actorFrom(c)
	user, err := a.auth.Me(c.Request.Context(), actor)
	if err != nil {
		a.fail(c, err)
		return
	}
	perms := make([]string, 0, len(actor.Permissions))
	for p := range actor.Permissions {
		perms = append(perms, string(p))
	}
	ids := actor.SiteIDList()
	c.JSON(http.StatusOK, dto.MeResponse{User: toUserDTO(user), Permissions: perms, AllSites: actor.AllSites, SiteIDs: ids})
}

// setupMFA godoc
//
//	@Summary		Start two-step verification setup
//	@Description	Returns a secret and an otpauth:// link to show as a QR code. Two-step verification turns on only after /auth/mfa/enable confirms a code.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.MFASetupResponse
//	@Router			/api/v1/auth/mfa/setup [post]
func (a *api) setupMFA(c *gin.Context) {
	setup, err := a.auth.SetupMFA(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MFASetupResponse{Secret: setup.Secret, OTPAuthURL: setup.URL})
}

// enableMFA godoc
//
//	@Summary	Confirm a code and turn on two-step verification
//	@Tags		auth
//	@Accept		json
//	@Param		body	body	dto.EnableMFARequest	true	"Code from the authenticator app"
//	@Success	204
//	@Failure	401	{object}	dto.ErrorResponse
//	@Router		/api/v1/auth/mfa/enable [post]
func (a *api) enableMFA(c *gin.Context) {
	var req dto.EnableMFARequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	if err := a.auth.EnableMFA(c.Request.Context(), actorFrom(c), req.Code); err != nil {
		a.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
