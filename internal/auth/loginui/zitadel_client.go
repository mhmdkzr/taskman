package loginui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// restClient is a small bearer-authenticated JSON HTTP client shared by the
// Session API and OIDC auth-request clients below, which otherwise differ
// only in which ZITADEL v2 REST resource they call.
type restClient struct {
	http   *http.Client
	issuer string
	tokens TokenSource
}

// doJSON issues a bearer-authenticated JSON request against the ZITADEL v2
// REST API and decodes a successful response into out.
func (c *restClient) doJSON(ctx context.Context, method, path string, body, out any) error {
	token, err := c.tokens.GetValidToken()
	if err != nil {
		return fmt.Errorf("get zitadel access token: %w", err)
	}
	var bodyReader io.Reader = http.NoBody
	if body != nil {
		var payload bytes.Buffer
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		bodyReader = &payload
	}
	req, err := http.NewRequestWithContext(ctx, method, c.issuer+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call %s %s: %w", method, path, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("close zitadel response body", "error", err)
		}
	}()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest ||
			resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w: %s %s returned %s", errInvalidCredentials, method, path, resp.Status)
		}
		return fmt.Errorf("%s %s returned %s: %s", method, path, resp.Status, respBody)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}

// sessionAPIClient calls ZITADEL's v2 REST Session API
// (https://zitadel.com/docs/guides/integrate/login-ui/username-password).
type sessionAPIClient struct{ *restClient }

type createSessionRequest struct {
	Checks createSessionChecks `json:"checks"`
}

type createSessionChecks struct {
	User *checkUser `json:"user"`
}

type checkUser struct {
	LoginName string `json:"loginName"`
}

type createSessionResponse struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
}

func (c *sessionAPIClient) createSession(ctx context.Context, loginName string) (string, string, error) {
	var resp createSessionResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v2/sessions", createSessionRequest{
		Checks: createSessionChecks{User: &checkUser{LoginName: loginName}},
	}, &resp); err != nil {
		return "", "", err
	}
	if resp.SessionID == "" || resp.SessionToken == "" {
		return "", "", fmt.Errorf("%w: create session: empty session id or token", errInvalidCredentials)
	}
	return resp.SessionID, resp.SessionToken, nil
}

type setSessionRequest struct {
	Checks setSessionChecks `json:"checks"`
}

type setSessionChecks struct {
	Password *checkPassword `json:"password,omitempty"`
	Totp     *checkTOTP     `json:"totp,omitempty"`
}

type checkPassword struct {
	Password string `json:"password"`
}

type checkTOTP struct {
	Code string `json:"code"`
}

type setSessionResponse struct {
	SessionToken string `json:"sessionToken"`
}

func (c *sessionAPIClient) checkPassword(ctx context.Context, sessionID, password string) (string, error) {
	var resp setSessionResponse
	if err := c.doJSON(ctx, http.MethodPatch, "/v2/sessions/"+url.PathEscape(sessionID), setSessionRequest{
		Checks: setSessionChecks{Password: &checkPassword{Password: password}},
	}, &resp); err != nil {
		return "", err
	}
	if resp.SessionToken == "" {
		return "", fmt.Errorf("%w: check password: empty session token", errInvalidCredentials)
	}
	return resp.SessionToken, nil
}

func (c *sessionAPIClient) checkTOTP(ctx context.Context, sessionID, code string) (string, error) {
	var resp setSessionResponse
	if err := c.doJSON(ctx, http.MethodPatch, "/v2/sessions/"+url.PathEscape(sessionID), setSessionRequest{
		Checks: setSessionChecks{Totp: &checkTOTP{Code: code}},
	}, &resp); err != nil {
		return "", err
	}
	if resp.SessionToken == "" {
		return "", fmt.Errorf("%w: check totp: empty session token", errInvalidCredentials)
	}
	return resp.SessionToken, nil
}

// oidcAuthRequestClient calls ZITADEL's v2 REST OIDC service to finalize an
// authorization request against an authenticated session
// (https://zitadel.com/docs/guides/integrate/login-ui/oidc-standard). The
// caller must hold the IAM_LOGIN_CLIENT role.
type oidcAuthRequestClient struct{ *restClient }

type createCallbackRequest struct {
	Session createCallbackSession `json:"session"`
}

type createCallbackSession struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
}

type createCallbackResponse struct {
	CallbackURL string `json:"callbackUrl"`
}

// createCallback finalizes authRequestID and returns the authorization code
// and state carried on the callback URL ZITADEL would otherwise have
// redirected the browser to.
func (c *oidcAuthRequestClient) createCallback(
	ctx context.Context, authRequestID, sessionID, sessionToken string,
) (string, string, error) {
	var resp createCallbackResponse
	if err := c.doJSON(ctx, http.MethodPost,
		"/v2/oidc/auth_requests/"+url.PathEscape(authRequestID),
		createCallbackRequest{Session: createCallbackSession{SessionID: sessionID, SessionToken: sessionToken}},
		&resp,
	); err != nil {
		return "", "", err
	}
	callback, err := url.Parse(resp.CallbackURL)
	if err != nil {
		return "", "", fmt.Errorf("parse callback url: %w", err)
	}
	code := callback.Query().Get("code")
	state := callback.Query().Get("state")
	if code == "" {
		return "", "", fmt.Errorf("callback url missing authorization code")
	}
	return code, state, nil
}
