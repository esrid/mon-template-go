package identity

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/esrid/mon-template-go/internal/platform/web"
)

const sessionUserID = "identity.user_id"

type HTTP struct {
	service  *Service
	sessions *scs.SessionManager
	google   *GoogleProvider
}

func NewHTTP(service *Service, sessions *scs.SessionManager, google *GoogleProvider) *HTTP {
	return &HTTP{service: service, sessions: sessions, google: google}
}
func Mount(mux *http.ServeMux, h *HTTP) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("GET /auth/me", h.me)
	mux.HandleFunc("GET /auth/google", h.googleStart)
	mux.HandleFunc("GET /auth/google/callback", h.googleCallback)
	mux.HandleFunc("POST /auth/email-verification", h.issueVerification)
	mux.HandleFunc("POST /auth/email-verification/confirm", h.confirmVerification)
	mux.HandleFunc("POST /auth/password-reset", h.issueReset)
	mux.HandleFunc("POST /auth/password-reset/confirm", h.confirmReset)
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	return json.NewDecoder(r.Body).Decode(dst)
}
func (h *HTTP) register(w http.ResponseWriter, r *http.Request) {
	var p struct{ Email, Password string }
	if decode(w, r, &p) != nil {
		web.JSON(w, 400, errorBody("invalid JSON"))
		return
	}
	u, err := h.service.Register(r.Context(), p.Email, p.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err = h.sessions.RenewToken(r.Context()); err != nil {
		h.fail(w, r, err)
		return
	}
	h.signIn(r.Context(), u.ID)
	web.JSON(w, 201, u)
}
func (h *HTTP) login(w http.ResponseWriter, r *http.Request) {
	var p struct{ Email, Password string }
	if decode(w, r, &p) != nil {
		web.JSON(w, 400, errorBody("invalid JSON"))
		return
	}
	u, err := h.service.Login(r.Context(), p.Email, p.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err = h.sessions.RenewToken(r.Context()); err != nil {
		h.fail(w, r, err)
		return
	}
	h.signIn(r.Context(), u.ID)
	web.JSON(w, 200, u)
}
func (h *HTTP) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Destroy(r.Context()); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *HTTP) me(w http.ResponseWriter, r *http.Request) {
	web.NoStore(w)
	id := h.sessions.GetInt64(r.Context(), sessionUserID)
	if id == 0 {
		web.JSON(w, 401, errorBody("authentication required"))
		return
	}
	u, err := h.service.ByID(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	web.JSON(w, 200, u)
}
func (h *HTTP) googleStart(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		h.fail(w, r, ErrGoogleDisabled)
		return
	}
	state, nonce := randomURLToken(), randomURLToken()
	h.sessions.Put(r.Context(), "google.state", state)
	h.sessions.Put(r.Context(), "google.nonce", nonce)
	http.Redirect(w, r, h.google.AuthCodeURL(state, nonce), http.StatusFound)
}
func (h *HTTP) googleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		h.fail(w, r, ErrGoogleDisabled)
		return
	}
	state := h.sessions.PopString(r.Context(), "google.state")
	nonce := h.sessions.PopString(r.Context(), "google.nonce")
	if state == "" || state != r.URL.Query().Get("state") || nonce == "" {
		h.fail(w, r, ErrInvalidCredentials)
		return
	}
	ext, err := h.google.Verify(r.Context(), r.URL.Query().Get("code"), nonce)
	if err != nil {
		h.fail(w, r, ErrInvalidCredentials)
		return
	}
	u, err := h.service.LoginWithGoogle(r.Context(), ext)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	_ = h.sessions.RenewToken(r.Context())
	h.signIn(r.Context(), u.ID)
	web.JSON(w, 200, u)
}
func (h *HTTP) issueVerification(w http.ResponseWriter, r *http.Request) {
	id := h.sessions.GetInt64(r.Context(), sessionUserID)
	if id == 0 {
		web.JSON(w, 401, errorBody("authentication required"))
		return
	}
	token, err := h.service.IssueToken(r.Context(), EmailVerificationToken, id, 24*time.Hour)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	web.JSON(w, 202, map[string]string{"token": token, "warning": "development transport only; send this token by email in production"})
}
func (h *HTTP) confirmVerification(w http.ResponseWriter, r *http.Request) {
	var p struct{ Token string }
	if decode(w, r, &p) != nil {
		web.JSON(w, 400, errorBody("invalid JSON"))
		return
	}
	u, err := h.service.VerifyEmail(r.Context(), p.Token)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	web.JSON(w, 200, u)
}
func (h *HTTP) issueReset(w http.ResponseWriter, r *http.Request) {
	var p struct{ Email string }
	if decode(w, r, &p) != nil {
		web.JSON(w, 202, map[string]string{"status": "accepted"})
		return
	}
	u, _, err := h.service.store.ByEmail(r.Context(), normalizeEmail(p.Email))
	if err == nil {
		token, e := h.service.IssueToken(r.Context(), PasswordResetToken, u.ID, time.Hour)
		if e == nil {
			slog.Info("development password reset token", "user_id", u.ID, "token", token)
		}
	}
	web.JSON(w, 202, map[string]string{"status": "accepted"})
}
func (h *HTTP) confirmReset(w http.ResponseWriter, r *http.Request) {
	var p struct{ Token, Password string }
	if decode(w, r, &p) != nil {
		web.JSON(w, 400, errorBody("invalid JSON"))
		return
	}
	userID, err := h.service.ResetPassword(r.Context(), p.Token, p.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.revokeSessions(r.Context(), userID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// revokeSessions destroys every live session of a user, so a password reset locks
// out whoever held the account before.
// ponytail: scans all live sessions; add a user_id column if session volume grows.
func (h *HTTP) revokeSessions(ctx context.Context, userID int64) error {
	return h.sessions.Iterate(ctx, func(sessionCtx context.Context) error {
		if h.sessions.GetInt64(sessionCtx, sessionUserID) != userID {
			return nil
		}
		return h.sessions.Destroy(sessionCtx)
	})
}
func (h *HTTP) signIn(ctx context.Context, id int64) { h.sessions.Put(ctx, sessionUserID, id) }
func (h *HTTP) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword), errors.Is(err, ErrPasswordTooLong):
		web.JSON(w, 400, errorBody(err.Error()))
	case errors.Is(err, ErrEmailTaken):
		web.JSON(w, 409, errorBody("email already registered"))
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidToken):
		web.JSON(w, 401, errorBody("invalid credentials or token"))
	case errors.Is(err, ErrGoogleDisabled):
		web.JSON(w, 404, errorBody("google login is disabled"))
	case errors.Is(err, ErrNotFound):
		web.JSON(w, 404, errorBody("user not found"))
	default:
		slog.Error("identity request failed", "request_id", web.RequestID(r.Context()), "err", err)
		web.JSON(w, 500, errorBody("internal server error"))
	}
}
func errorBody(message string) map[string]string { return map[string]string{"error": message} }
