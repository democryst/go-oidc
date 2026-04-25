package handlers

import (
	"net/http"
	"net/url"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

type OIDCHandler struct {
	svc interfaces.OIDCService
}

func NewOIDCHandler(svc interfaces.OIDCService) *OIDCHandler {
	return &OIDCHandler{svc: svc}
}

func (h *OIDCHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	req := interfaces.AuthorizeRequest{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}

	resp, err := h.svc.Authorize(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Build redirect URI with code and state
	u, err := url.Parse(req.RedirectURI)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_redirect_uri", "The provided redirect_uri is malformed")
		return
	}

	vals := u.Query()
	vals.Set("code", resp.Code)
	if resp.State != "" {
		vals.Set("state", resp.State)
	}
	u.RawQuery = vals.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *OIDCHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	req := interfaces.TokenRequest{
		GrantType:    r.PostFormValue("grant_type"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Code:         r.PostFormValue("code"),
		RedirectURI:  r.PostFormValue("redirect_uri"),
		CodeVerifier: r.PostFormValue("code_verifier"),
		RefreshToken: r.PostFormValue("refresh_token"),
	}

	resp, err := h.svc.Token(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid_grant", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *OIDCHandler) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	doc := h.svc.Discovery()
	h.writeJSON(w, http.StatusOK, doc)
}

func (h *OIDCHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	keys := h.svc.JWKS()
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"keys": keys})
}

func (h *OIDCHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	payload, release, err := EncodeJSONPooled(data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer release()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(payload)
}

func (h *OIDCHandler) writeError(w http.ResponseWriter, status int, errorType, description string) {
	h.writeJSON(w, status, map[string]string{
		"error":             errorType,
		"error_description": description,
	})
}
