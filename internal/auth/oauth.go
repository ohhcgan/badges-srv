package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"

	"github-badges-backend/internal/config"
	"github-badges-backend/internal/crypto"
	githubClient "github-badges-backend/internal/github"
	"github-badges-backend/internal/user"
)

const (
	stateCookieName   = "oauth_state"
	sessionCookieName = "session"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyLogin  = "login"
)

const (
	stateLength   = 16
	stateMaxAge   = 600              /* 10 minutes in seconds */
	sessionMaxAge = 7 * 24 * 60 * 60 /* 7 days in seconds */
)

type JWTPayload struct {
	UserID      string `json:"uid"`
	GithubLogin string `json:"login"`

	jwt.RegisteredClaims
}

/**
 * AuthHandler handles the GitHub OAuth login/callback flow.
 */
type AuthHandler struct {
	oauthConfig *oauth2.Config
	userStore   *user.Store
	cryptoKey   []byte
	jwtSecret   []byte
	frontendURL string
	logger      *zap.Logger
}

func NewHandler(conf *config.Config, userStore *user.Store, logger *zap.Logger) *AuthHandler {
	oauthCfg := &oauth2.Config{
		ClientID:     conf.GithubClientID,
		ClientSecret: conf.GithubClientSecret,
		RedirectURL:  conf.GithubRedirectURL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     githubOAuth.Endpoint,
	}

	return &AuthHandler{
		oauthConfig: oauthCfg,
		userStore:   userStore,
		cryptoKey:   conf.EncryptionKeyBytes(),
		jwtSecret:   []byte(conf.JWTSecret),
		frontendURL: conf.FrontendURL,
		logger:      logger,
	}
}

/**
 * HandleLogin redirects the browser to GitHub's OAuth consent page.
 */
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomString(stateLength)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		MaxAge:   stateMaxAge,
		Path:     "/",
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,

		// Domain:  "",                               /* TODO: add domain */
		// Quoted:  false,                            /* TODO: upgrade to go1.23 to use Quoted */
		// Expires: time.Now().Add(10 * time.Minute), /* TODO: Add Expires */
	})

	githubAuthServUrl := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, githubAuthServUrl, http.StatusTemporaryRedirect)
}

/**
 * HandleCallback handles the GitHub redirect after the user grants (or denies) access.
 */
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if stateCookie.Value != state {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   stateCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	if code == "" {
		http.Error(w, "missing oauth code", http.StatusBadRequest)
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		h.logger.Error("oauth token exchange failed", zap.Error(err))
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	githubID, login, name, email, avatarURL, err := githubClient.UserInfo(r.Context(), token.AccessToken)

	if err != nil {
		h.logger.Error("failed to fetch github user info", zap.Error(err))
		http.Error(w, "could not fetch github profile", http.StatusBadGateway)
		return
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	encToken, err := crypto.Encrypt(h.cryptoKey, tokenJSON)
	if err != nil {
		h.logger.Error("token encryption failed", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	u := &user.User{
		GithubID:       githubID,
		GithubLogin:    login,
		Name:           name,
		Email:          email,
		AvatarURL:      avatarURL,
		EncryptedToken: encToken,
	}

	if err := h.userStore.Upsert(r.Context(), u); err != nil {
		h.logger.Error("user upsert failed", zap.Error(err), zap.String("login", login))
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	sessionToken, err := h.generateJWT(u)
	if err != nil {
		h.logger.Error("jwt generation failed", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Path:     "/",
	})

	h.logger.Info("user logged in", zap.String("login", login), zap.String("user_id", u.ID))
	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) generateJWT(u *user.User) (string, error) {
	now := time.Now()

	payload := JWTPayload{
		UserID:      u.ID,
		GithubLogin: u.GithubLogin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString(h.jwtSecret)
}

func randomString(length int16) (string, error) {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
