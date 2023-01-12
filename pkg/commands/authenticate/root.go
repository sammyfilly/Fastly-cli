package authenticate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fastly/cli/pkg/cmd"
	"github.com/fastly/cli/pkg/config"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/profile"
	"github.com/fastly/cli/pkg/text"
	"github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/cap/oidc"
	"github.com/skratchdot/open-golang/open"
)

// RootCommand is the parent command for all subcommands in this package.
// It should be installed under the primary root command.
type RootCommand struct {
	cmd.Base
}

// AuthRemediation is a generic remediation message for an error authorizing.
const AuthRemediation = "Please re-run the command. If the problem persists, please file an issue: https://github.com/fastly/cli/issues/new?labels=bug&template=bug_report.md"

// AuthProviderCLIAppURL is the auth provider's device code URL.
const AuthProviderCLIAppURL = "https://dev-37kjpso9.us.auth0.com"

// AuthProviderClientID is the auth provider's Client ID.
const AuthProviderClientID = "7TAnqT4DTDhJTyXk9aXcuD48JoHRXK2X"

// AuthProviderAudience is the unique identifier of the API your app wants to access.
const AuthProviderAudience = "https://api.secretcdn-stg.net/"

// AuthProviderRedirectURL is the endpoint the auth provider will pass an authorization code to.
const AuthProviderRedirectURL = "http://localhost:8080/callback"

// NewRootCommand returns a new command registered in the parent.
func NewRootCommand(parent cmd.Registerer, g *global.Data) *RootCommand {
	var c RootCommand
	c.Globals = g
	c.CmdClause = parent.Command("authenticate", "Authenticate with Fastly (returns temporary, auto-rotated, API token)")
	return &c
}

// Exec implements the command interface.
func (c *RootCommand) Exec(_ io.Reader, out io.Writer) error {
	verifier, err := oidc.NewCodeVerifier()
	if err != nil {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to generate a code verifier: %w", err),
			Remediation: AuthRemediation,
		}
	}

	result := make(chan authorizationResult)

	s := server{
		result:   result,
		router:   http.NewServeMux(),
		verifier: verifier,
	}
	s.routes()

	var serverErr error

	go func() {
		err := s.startServer()
		if err != nil {
			serverErr = err
		}
	}()

	if serverErr != nil {
		return serverErr
	}

	text.Info(out, "Starting localhost server to handle the authentication flow.")

	authorizationURL, err := generateAuthorizationURL(verifier)
	if err != nil {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to generate an authorization URL: %w", err),
			Remediation: AuthRemediation,
		}
	}

	text.Break(out)
	text.Description(out, "We're opening the following URL in your default web browser so you may authenticate with Fastly", authorizationURL)

	err = open.Run(authorizationURL)
	if err != nil {
		return fmt.Errorf("failed to open your default browser: %w", err)
	}

	ar := <-result
	if ar.err != nil || ar.sessionToken == "" {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to authorize: %w", ar.err),
			Remediation: AuthRemediation,
		}
	}

	text.Success(out, "Session token (persisted to your local configuration): %s", ar.sessionToken)

	profileName, _ := profile.Default(c.Globals.Config.Profiles)
	if profileName == "" {
		// FIXME: Return a more appropriate remediation.
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("no profiles available"),
			Remediation: fsterr.ProfileRemediation,
		}
	}

	ps, ok := profile.Edit(profileName, c.Globals.Config.Profiles, func(p *config.Profile) {
		p.Token = ar.sessionToken
	})
	if !ok {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to update default profile with new session token"),
			Remediation: "Run `fastly profile update` and manually paste in the session token.",
		}
	}
	c.Globals.Config.Profiles = ps

	if err := c.Globals.Config.Write(c.Globals.Path); err != nil {
		c.Globals.ErrLog.Add(err)
		return fmt.Errorf("error saving config file: %w", err)
	}

	// FIXME: Don't just update the default profile.
	// Allow user to configure this via a --profile flag.

	return nil
}

type server struct {
	result   chan authorizationResult
	router   *http.ServeMux
	verifier *oidc.S256Verifier
}

func (s *server) startServer() error {
	// TODO: Consider using a random port to avoid local network conflicts.
	// Chat with authentication provider about how to use a random port.
	err := http.ListenAndServe(":8080", s.router)
	if err != nil {
		return fsterr.RemediationError{
			Inner:       fmt.Errorf("failed to start local server: %w", err),
			Remediation: AuthRemediation,
		}
	}
	return nil
}

func (s *server) routes() {
	s.router.HandleFunc("/callback", s.handleCallback())
}

func (s *server) handleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorizationCode := r.URL.Query().Get("code")
		if authorizationCode == "" {
			fmt.Fprint(w, "ERROR: no authorization code returned\n")
			s.result <- authorizationResult{
				err: fmt.Errorf("no authorization code returned"),
			}
			return
		}

		// Exchange the authorization code and the code verifier for a JWT.
		// NOTE: I use the identifier `j` to avoid overlap with the `jwt` package.
		codeVerifier := s.verifier.Verifier()
		j, err := getJWT(codeVerifier, authorizationCode)
		if err != nil || j.AccessToken == "" || j.IDToken == "" {
			fmt.Fprint(w, "ERROR: failed to exchange code for JWT\n")
			s.result <- authorizationResult{
				err: fmt.Errorf("failed to exchange code for JWT"),
			}
			return
		}

		// FIXME: Patrick to move the session token inside of the access_token.
		// Currently it's inside of the id_token!
		_, err = verifyJWTSignature(j.AccessToken)
		if err != nil {
			s.result <- authorizationResult{
				err: err,
			}
			return
		}

		claims, err := verifyJWTSignature(j.IDToken)
		if err != nil {
			s.result <- authorizationResult{
				err: err,
			}
			return
		}

		sessionToken, err := extractSessionToken(claims)
		if err != nil {
			s.result <- authorizationResult{
				err: err,
			}
			return
		}

		fmt.Fprint(w, "Authenticated successfully. Please close this page and return to the Fastly CLI in your terminal.")
		s.result <- authorizationResult{
			jwt:          j,
			sessionToken: sessionToken,
		}
	}
}

type authorizationResult struct {
	err          error
	jwt          JWT
	sessionToken string
}

func generateAuthorizationURL(verifier *oidc.S256Verifier) (string, error) {
	challenge, err := oidc.CreateCodeChallenge(verifier)
	if err != nil {
		return "", err
	}

	authorizationURL := fmt.Sprintf(
		"%s/authorize?audience=%s"+
			"&scope=openid"+
			"&response_type=code&client_id=%s"+
			"&code_challenge=%s"+
			"&code_challenge_method=S256&redirect_uri=%s",
		AuthProviderCLIAppURL, AuthProviderAudience, AuthProviderClientID, challenge, AuthProviderRedirectURL)

	return authorizationURL, nil
}

func getJWT(codeVerifier, authorizationCode string) (JWT, error) {
	path := "/oauth/token"

	payload := fmt.Sprintf(
		"grant_type=authorization_code&client_id=%s&code_verifier=%s&code=%s&redirect_uri=%s",
		AuthProviderClientID,
		codeVerifier,
		authorizationCode,
		"http://localhost:8080", // NOTE: not redirected to, just a security check.
	)

	req, err := http.NewRequest("POST", AuthProviderCLIAppURL+path, strings.NewReader(payload))
	if err != nil {
		return JWT{}, err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return JWT{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return JWT{}, fmt.Errorf("failed to exchange code for jwt (status: %s)", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return JWT{}, err
	}

	// NOTE: I use the identifier `j` to avoid overlap with the `jwt` package.
	var j JWT
	err = json.Unmarshal(body, &j)
	if err != nil {
		return JWT{}, err
	}

	return j, nil
}

// JWT is the API response for a Token request.
type JWT struct {
	// AccessToken can be exchanged for a Fastly API token.
	AccessToken string `json:"access_token"`
	// ExpiresIn indicates the lifetime (in seconds) of the access token.
	ExpiresIn int `json:"expires_in"`
	// IDToken contains user information that must be decoded and extracted.
	IDToken string `json:"id_token"`
	// TokenType indicates which HTTP authentication scheme is used (e.g. Bearer).
	TokenType string `json:"token_type"`
}

func verifyJWTSignature(token string) (claims map[string]any, err error) {
	ctx := context.Background()

	// NOTE: The last argument is optional and is for validating the JWKs endpoint
	// (which we don't need to do, so we pass an empty string)
	keySet, err := jwt.NewJSONWebKeySet(ctx, AuthProviderCLIAppURL+"/.well-known/jwks.json", "")
	if err != nil {
		return claims, fmt.Errorf("failed to verify signature of access token: %w", err)
	}

	claims, err = keySet.VerifySignature(ctx, token)
	if err != nil {
		return claims, fmt.Errorf("failed to verify signature of access token: %w", err)
	}

	return claims, nil
}

func extractSessionToken(claims map[string]any) (string, error) {
	if i, ok := claims["ui_token"]; ok {
		if m, ok := i.(map[string]any); ok {
			if v, ok := m["access_token"]; ok {
				if t, ok := v.(string); ok {
					if t != "" {
						return t, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("failed to extract session token from JWT custom claim")
}
