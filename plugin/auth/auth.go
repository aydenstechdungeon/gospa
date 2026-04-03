// Package auth provides authentication for GoSPA projects.
// Includes OAuth2 (Google, Facebook, GitHub, Microsoft, Discord), JWT sessions, and TOTP/OTP.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec //nolint:gosec
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aydenstechdungeon/gospa/plugin"
	"github.com/aydenstechdungeon/gospa/store"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const (
	oauthStateCookiePrefix = "gospa_oauth_state_"
	oauthStateTTL          = 10 * time.Minute
)

// EnableTOTP is an alias for EnableOTPHandler for backward compatibility.
func (p *AuthPlugin) EnableTOTP() fiber.Handler { return p.EnableOTPHandler() }

// VerifyTOTP is an alias for VerifyOTPHandler for backward compatibility.
func (p *AuthPlugin) VerifyTOTP() fiber.Handler { return p.VerifyOTPHandler() }

// otpRateEntry tracks in-memory OTP attempt counts with expiry.
type otpRateEntry struct {
	count     int32
	expiresAt int64 // unix timestamp
}

// AuthPlugin provides authentication capabilities.
//
//nolint:revive // changing name would break API
type AuthPlugin struct {
	config     *Config
	storage    store.Storage
	otpLimiter sync.Map // map[string]*otpRateEntry
}

// SetStorage sets the storage backend for the plugin.
func (p *AuthPlugin) SetStorage(s store.Storage) {
	p.storage = s
}

// Config holds auth plugin configuration.
type Config struct {
	// JWTSecret is the secret key for JWT signing.
	JWTSecret string `yaml:"jwt_secret" json:"jwtSecret"`

	// JWTExpiry is the JWT token expiry duration in hours.
	JWTExpiry int `yaml:"jwt_expiry" json:"jwtExpiry"`

	// Issuer is the JWT issuer.
	Issuer string `yaml:"issuer" json:"issuer"`

	// OAuthProviders is a list of enabled OAuth providers.
	OAuthProviders []string `yaml:"oauth_providers" json:"oauthProviders"`

	// Google OAuth config.
	GoogleClientID     string `yaml:"google_client_id" json:"googleClientId"`
	GoogleClientSecret string `yaml:"google_client_secret" json:"googleClientSecret"`

	// Facebook OAuth config.
	FacebookClientID     string `yaml:"facebook_client_id" json:"facebookClientId"`
	FacebookClientSecret string `yaml:"facebook_client_secret" json:"facebookClientSecret"`

	// GitHub OAuth config.
	GitHubClientID     string `yaml:"github_client_id" json:"githubClientId"`
	GitHubClientSecret string `yaml:"github_client_secret" json:"githubClientSecret"`

	// Microsoft OAuth config.
	MicrosoftClientID     string `yaml:"microsoft_client_id" json:"microsoftClientId"`
	MicrosoftClientSecret string `yaml:"microsoft_client_secret" json:"microsoftClientSecret"`

	// Discord OAuth config.
	DiscordClientID     string `yaml:"discord_client_id" json:"discordClientId"`
	DiscordClientSecret string `yaml:"discord_client_secret" json:"discordClientSecret"`

	// Telegram OAuth config.
	TelegramBotToken string `yaml:"telegram_bot_token" json:"telegramBotToken"`

	// Twitter/X OAuth config.
	TwitterClientID     string `yaml:"twitter_client_id" json:"twitterClientId"`
	TwitterClientSecret string `yaml:"twitter_client_secret" json:"twitterClientSecret"`

	// OTP config.
	OTPEnabled      bool   `yaml:"otp_enabled" json:"otpEnabled"`
	OTPIssuer       string `yaml:"otp_issuer" json:"otpIssuer"`
	OTPDigits       int    `yaml:"otp_digits" json:"otpDigits"`
	OTPPeriod       int    `yaml:"otp_period" json:"otpPeriod"`
	BackupCodeCount int    `yaml:"backup_code_count" json:"backupCodeCount"`

	// OutputDir is where generated auth code is written.
	OutputDir string `yaml:"output_dir" json:"outputDir"`
}

// OAuthProvider represents an OAuth provider configuration.
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserURL      string
	Scopes       []string
}

// OTPConfig represents TOTP configuration.
type OTPConfig struct {
	Secret  string
	Digits  int
	Period  int
	Issuer  string
	Account string
}

// User represents an authenticated user.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Claims represents JWT claims.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// BackupCode represents a backup code for 2FA.
type BackupCode struct {
	Code   string
	Used   bool
	UsedAt *time.Time
}

func oauthStateCookieName(provider string) string {
	return oauthStateCookiePrefix + strings.ToLower(provider)
}

func setOAuthStateCookie(c fiber.Ctx, provider, state string) {
	secure := c.Protocol() == "https" || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    state,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   secure,
		MaxAge:   int(oauthStateTTL.Seconds()),
		Expires:  time.Now().Add(oauthStateTTL),
	})
}

// isProductionRuntime reports whether the process environment indicates a production deployment.
// JWT_SECRET is required when this is true and [DefaultConfig] is used without an explicit [Config].
// Recognized signals: GOSPA_ENV=production|prod, ENV/APP_ENV/GO_ENV=production (case-insensitive),
// and GIN_MODE=release (legacy compatibility).
func isProductionRuntime() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOSPA_ENV"))) {
	case "production", "prod":
		return true
	}
	for _, key := range []string{"ENV", "APP_ENV", "GO_ENV"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") {
			return true
		}
	}
	return os.Getenv("GIN_MODE") == "release"
}

func clearOAuthStateCookie(c fiber.Ctx, provider string) {
	secure := c.Protocol() == "https" || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   secure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// CreateToken creates a new JWT token.
func (p *AuthPlugin) CreateToken(userID, email, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(p.config.JWTExpiry) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    p.config.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(p.config.JWTSecret))
}

// ValidateToken validates a JWT token and returns the claims.
func (p *AuthPlugin) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(parsed *jwt.Token) (interface{}, error) {
		if parsed.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", parsed.Header["alg"])
		}
		return []byte(p.config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Issuer != p.config.Issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	return claims, nil
}

// RequireAuth returns a middleware that requires authentication.
func (p *AuthPlugin) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization scheme"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := p.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("user", &User{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		})
		return c.Next()
	}
}

// OAuthRedirect returns a handler that redirects to an OAuth provider.
func (p *AuthPlugin) OAuthRedirect(providerName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		provider, err := p.getProvider(providerName)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		conf := &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
			Scopes: provider.Scopes,
		}

		state, err := generateRandomSecret(32)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to initialize oauth state"})
		}
		setOAuthStateCookie(c, providerName, state)

		url := conf.AuthCodeURL(state)
		return c.Redirect().Status(fiber.StatusFound).To(url)
	}
}

// OAuthCallback returns a handler that handles the OAuth callback.
func (p *AuthPlugin) OAuthCallback(providerName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		code := c.Query("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing code"})
		}

		returnedState := c.Query("state")
		expectedState := c.Cookies(oauthStateCookieName(providerName))
		if returnedState == "" || expectedState == "" || subtle.ConstantTimeCompare([]byte(returnedState), []byte(expectedState)) != 1 {
			clearOAuthStateCookie(c, providerName)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid oauth state"})
		}
		clearOAuthStateCookie(c, providerName)

		provider, err := p.getProvider(providerName)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		conf := &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
			Scopes: provider.Scopes,
		}

		_, err = conf.Exchange(c.Context(), code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token exchange failed"})
		}

		// SECURITY: Never return upstream provider tokens directly to the browser.
		// Applications should fetch provider user info server-side and mint an app session/JWT.
		return c.JSON(fiber.Map{"success": true})
	}
}

// EnableOTPHandler returns a handler that generates OTP setup info.
func (p *AuthPlugin) EnableOTPHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		userAny := c.Locals("user")
		user, ok := userAny.(*User)
		if !ok || user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		secret, url, err := p.GenerateOTP(user.Email)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate OTP"})
		}

		return c.JSON(fiber.Map{
			"secret":  secret,
			"qr_code": url,
		})
	}
}

// VerifyOTPHandler returns a handler that verifies an OTP code.
func (p *AuthPlugin) VerifyOTPHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Secret string `json:"secret"`
			Code   string `json:"code"`
		}
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}

		rateKey := c.IP()
		if u, ok := c.Locals("user").(*User); ok {
			rateKey = "user:" + u.ID
		}

		if p.storage != nil {
			return p.verifyOTPWithStorage(c, rateKey, req.Secret, req.Code)
		}
		return p.verifyOTPInMemory(c, rateKey, req.Secret, req.Code)
	}
}

func (p *AuthPlugin) verifyOTPWithStorage(c fiber.Ctx, limitKey, secret, code string) error {
	var count int
	if b, err := p.storage.Get(limitKey); err == nil {
		count, _ = strconv.Atoi(string(b))
	}
	if count >= 5 {
		return c.Status(429).JSON(fiber.Map{"error": "too many attempts. please wait."})
	}
	if p.VerifyOTP(secret, code) {
		_ = p.storage.Delete(limitKey)
		return c.JSON(fiber.Map{"success": true})
	}
	count++
	_ = p.storage.Set(limitKey, []byte(strconv.Itoa(count)), 5*time.Minute)
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "invalid OTP code"})
}

func (p *AuthPlugin) verifyOTPInMemory(c fiber.Ctx, rateKey, secret, code string) error {
	now := time.Now().Unix()

	val, loaded := p.otpLimiter.Load(rateKey)
	if !loaded {
		entry := &otpRateEntry{count: 1, expiresAt: now + 300}
		p.otpLimiter.Store(rateKey, entry)
	} else {
		entry := val.(*otpRateEntry)
		expires := atomic.LoadInt64(&entry.expiresAt)
		if now > expires {
			atomic.StoreInt32(&entry.count, 1)
			atomic.StoreInt64(&entry.expiresAt, now+300)
		} else {
			if atomic.LoadInt32(&entry.count) >= 5 {
				return c.Status(429).JSON(fiber.Map{"error": "too many attempts. please wait."})
			}
			atomic.AddInt32(&entry.count, 1)
		}
	}

	if p.VerifyOTP(secret, code) {
		p.otpLimiter.Delete(rateKey)
		return c.JSON(fiber.Map{"success": true})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "invalid OTP code"})
}

// getProvider returns provider by name.
func (p *AuthPlugin) getProvider(name string) (*OAuthProvider, error) {
	providers := p.getEnabledProviders()
	for _, pr := range providers {
		if strings.EqualFold(pr.Name, name) {
			return &pr, nil
		}
	}
	return nil, fmt.Errorf("provider %s not found or enabled", name)
}

// DefaultConfig returns the default auth configuration.
// JWTSecret MUST be set via JWT_SECRET environment variable in production.
// The application will panic if JWT_SECRET is not set in production.
func DefaultConfig() *Config {
	// Check for JWT_SECRET environment variable
	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" && isProductionRuntime() {
		panic("JWT_SECRET environment variable is not set. " +
			"Generate a secure secret with: openssl rand -hex 32 " +
			"and set it in your environment before starting the application in production.")
	}

	if jwtSecret != "" && len(jwtSecret) < 32 {
		panic("JWT_SECRET must be at least 32 characters for HS256 security. " +
			"Generate a secure secret with: openssl rand -hex 32")
	}

	// Use provided secret or generate one for development only
	if jwtSecret == "" {
		// Generate a random JWT secret for development only
		// IMPORTANT: This should NEVER be used in production
		randomSecret, err := generateRandomSecret(32)
		if err != nil {
			panic("failed to generate JWT secret for development: " + err.Error())
		}
		return &Config{
			JWTSecret:       randomSecret,
			JWTExpiry:       24,
			Issuer:          "gospa-app",
			OAuthProviders:  []string{},
			OTPEnabled:      true,
			OTPIssuer:       "GoSPA",
			OTPDigits:       6,
			OTPPeriod:       30,
			BackupCodeCount: 10,
			OutputDir:       "generated/auth",
		}
	}

	return &Config{
		JWTSecret:       jwtSecret,
		JWTExpiry:       24,
		Issuer:          "gospa-app",
		OAuthProviders:  []string{},
		OTPEnabled:      true,
		OTPIssuer:       "GoSPA",
		OTPDigits:       6,
		OTPPeriod:       30,
		BackupCodeCount: 10,
		OutputDir:       "generated/auth",
	}
}

// generateRandomSecret generates a cryptographically secure random secret.
func generateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// New creates a new Auth plugin.
func New(cfg *Config) *AuthPlugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.JWTExpiry <= 0 {
		cfg.JWTExpiry = 24
	}
	return &AuthPlugin{config: cfg}
}

// Name returns the plugin name.
func (p *AuthPlugin) Name() string {
	return "auth"
}

// Init initializes the auth plugin.
func (p *AuthPlugin) Init() error {
	// Create output directory
	if err := os.MkdirAll(p.config.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// Dependencies returns required dependencies.
func (p *AuthPlugin) Dependencies() []plugin.Dependency {
	deps := []plugin.Dependency{
		// JWT for Go
		{Type: plugin.DepGo, Name: "github.com/golang-jwt/jwt/v5", Version: "latest"},
		// OAuth2 for Go
		{Type: plugin.DepGo, Name: "golang.org/x/oauth2", Version: "latest"},
	}

	// OTP support
	if p.config.OTPEnabled {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepGo, Name: "github.com/pquerna/otp", Version: "latest",
		})
	}

	return deps
}

// OnHook handles lifecycle hooks.
func (p *AuthPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	if hook == plugin.AfterGenerate {
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		return p.generateAuthCode(projectDir)
	}
	return nil
}

// Commands returns custom CLI commands.
func (p *AuthPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "auth:generate",
			Alias:       "ag",
			Description: "Generate authentication code",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.generateAuthCode(projectDir)
			},
		},
		{
			Name:        "auth:secret",
			Alias:       "as",
			Description: "Generate a secure JWT secret",
			Action: func(_ []string) error {
				secret, err := p.generateSecret(32)
				if err != nil {
					return err
				}
				fmt.Printf("JWT_SECRET=%s\n", secret)
				return nil
			},
		},
		{
			Name:        "auth:otp",
			Alias:       "ao",
			Description: "Generate OTP secret and QR code URL",
			Action: func(args []string) error {
				account := "user@example.com"
				if len(args) > 0 {
					account = args[0]
				}
				return p.generateOTPSetup(account)
			},
		},
		{
			Name:        "auth:backup",
			Alias:       "ab",
			Description: "Generate backup codes for 2FA",
			Action: func(args []string) error {
				count := p.config.BackupCodeCount
				if len(args) > 0 {
					if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
						count = n
					}
				}
				return p.generateBackupCodes(count)
			},
		},
		{
			Name:        "auth:verify",
			Alias:       "av",
			Description: "Verify an OTP code against a secret",
			Action: func(args []string) error {
				if len(args) < 2 {
					return fmt.Errorf("usage: auth:verify <secret> <code>")
				}
				return p.verifyOTP(args[0], args[1])
			},
		},
	}
}

// generateAuthCode generates authentication code files.
func (p *AuthPlugin) generateAuthCode(projectDir string) error {
	outputDir := filepath.Join(projectDir, p.config.OutputDir)

	// Generate JWT utilities
	if err := p.generateJWTCode(outputDir); err != nil {
		return err
	}

	// Generate OAuth handlers
	if err := p.generateOAuthCode(outputDir); err != nil {
		return err
	}

	// Generate OTP utilities
	if p.config.OTPEnabled {
		if err := p.generateOTPCode(outputDir); err != nil {
			return err
		}
	}

	fmt.Println("Generated authentication code in", outputDir)
	return nil
}

// generateJWTCode generates JWT utilities.
func (p *AuthPlugin) generateJWTCode(outputDir string) error {
	code := `// Auto-generated JWT utilities
package auth

import (
	"errors"
	"os"
	"time"

	json "github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
)

// getJWTSecret returns the JWT secret from environment variable.
// CRITICAL: JWT_SECRET must be set via environment variable.
// The application will panic if JWT_SECRET is not configured.
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set. " +
			"Generate a secure secret with: openssl rand -hex 32 " +
			"and set it in your environment before starting the application.")
	}
	if len(secret) < 32 {
		panic("JWT_SECRET must be at least 32 characters for HS256 security.")
	}
	return []byte(secret)
}

type Claims struct {
	UserID string ` + "`" + `json:"user_id"` + "`" + `
	Email  string ` + "`" + `json:"email"` + "`" + `
	Role   string ` + "`" + `json:"role"` + "`" + `
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token.
func GenerateToken(userID, email, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(` + fmt.Sprintf("%d", p.config.JWTExpiry) + ` * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "` + p.config.Issuer + `",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ValidateToken validates a JWT token and returns the claims.
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
`
	return os.WriteFile(filepath.Join(outputDir, "jwt.go"), []byte(code), 0600)
}

// generateOAuthCode generates OAuth handlers.
func (p *AuthPlugin) generateOAuthCode(outputDir string) error {
	var sb strings.Builder
	sb.WriteString(`// Auto-generated OAuth handlers
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	oauthStateCookiePrefix = "gospa_oauth_state_"
	oauthStateTTL          = 10 * time.Minute
)

func oauthStateCookieName(provider string) string {
	return oauthStateCookiePrefix + strings.ToLower(provider)
}

func generateOAuthState(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func setOAuthStateCookie(c *fiber.Ctx, provider, state string) {
	secure := c.Protocol() == "https" || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    state,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   secure,
		MaxAge:   int(oauthStateTTL.Seconds()),
		Expires:  time.Now().Add(oauthStateTTL),
	})
}

func clearOAuthStateCookie(c *fiber.Ctx, provider string) {
	secure := c.Protocol() == "https" || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   secure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// UserInfo represents user information from OAuth providers.
type UserInfo struct {
	ID        string ` + "`" + `json:"id"` + "`" + `
	Email     string ` + "`" + `json:"email"` + "`" + `
	Name      string ` + "`" + `json:"name"` + "`" + `
	AvatarURL string ` + "`" + `json:"avatar_url"` + "`" + `
	Provider  string ` + "`" + `json:"provider"` + "`" + `
}

`)

	// Generate provider configs
	providers := p.getEnabledProviders()
	for _, provider := range providers {
		sb.WriteString(p.generateProviderConfig(provider))
	}

	sb.WriteString(`
// GetOAuthProviders returns all enabled OAuth providers.
func GetOAuthProviders() map[string]*oauth2.Config {
	return map[string]*oauth2.Config{
`)

	for _, provider := range providers {
		fmt.Fprintf(&sb, "\t\t\"%s\": get%sConfig(),\n", provider.Name, provider.Name)
	}

	sb.WriteString(`	}
}

// FetchUserInfo fetches user info from the OAuth provider.
func FetchUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*UserInfo, error) {
	providers := GetOAuthProviders()
	config, ok := providers[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	var userURL string
	switch provider {
`)

	for _, provider := range providers {
		fmt.Fprintf(&sb, "\tcase \"%s\":\n\t\tuserURL = \"%s\"\n", provider.Name, provider.UserURL)
	}

	sb.WriteString(`	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	client := config.Client(ctx, token)
	resp, err := client.Get(userURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseUserInfo(provider, body)
}

// BeginOAuthRedirect stores a per-request OAuth state and returns the provider redirect URL.
func BeginOAuthRedirect(c *fiber.Ctx, provider string, config *oauth2.Config) (string, error) {
	state, err := generateOAuthState(32)
	if err != nil {
		return "", err
	}
	setOAuthStateCookie(c, provider, state)
	return config.AuthCodeURL(state), nil
}

// ValidateOAuthCallbackState validates and clears the state cookie for an OAuth callback.
func ValidateOAuthCallbackState(c *fiber.Ctx, provider string) bool {
	returnedState := c.Query("state")
	expectedState := c.Cookies(oauthStateCookieName(provider))
	clearOAuthStateCookie(c, provider)
	if returnedState == "" || expectedState == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(returnedState), []byte(expectedState)) == 1
}

func parseUserInfo(provider string, body []byte) (*UserInfo, error) {
	info := &UserInfo{Provider: provider}
	
	switch provider {
	case "google":
		var g struct {
			ID      string ` + "`" + `json:"id"` + "`" + `
			Email   string ` + "`" + `json:"email"` + "`" + `
			Name    string ` + "`" + `json:"name"` + "`" + `
			Picture string ` + "`" + `json:"picture"` + "`" + `
		}
		if err := json.Unmarshal(body, &g); err != nil {
			return nil, err
		}
		info.ID, info.Email, info.Name, info.AvatarURL = g.ID, g.Email, g.Name, g.Picture

	case "github":
		var g struct {
			ID        int    ` + "`" + `json:"id"` + "`" + `
			Email     string ` + "`" + `json:"email"` + "`" + `
			Login     string ` + "`" + `json:"login"` + "`" + `
			AvatarURL string ` + "`" + `json:"avatar_url"` + "`" + `
		}
		if err := json.Unmarshal(body, &g); err != nil {
			return nil, err
		}
		info.ID = fmt.Sprintf("%d", g.ID)
		info.Email, info.Name, info.AvatarURL = g.Email, g.Login, g.AvatarURL

	case "facebook":
		var f struct {
			ID      string ` + "`" + `json:"id"` + "`" + `
			Email   string ` + "`" + `json:"email"` + "`" + `
			Name    string ` + "`" + `json:"name"` + "`" + `
			Picture struct {
				Data struct {
					URL string ` + "`" + `json:"url"` + "`" + `
				} ` + "`" + `json:"data"` + "`" + `
			} ` + "`" + `json:"picture"` + "`" + `
		}
		if err := json.Unmarshal(body, &f); err != nil {
			return nil, err
		}
		info.ID, info.Email, info.Name, info.AvatarURL = f.ID, f.Email, f.Name, f.Picture.Data.URL

	case "microsoft":
		var m struct {
			ID                string ` + "`" + `json:"id"` + "`" + `
			UserPrincipalName string ` + "`" + `json:"userPrincipalName"` + "`" + `
			DisplayName       string ` + "`" + `json:"displayName"` + "`" + `
		}
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, err
		}
		info.ID, info.Email, info.Name = m.ID, m.UserPrincipalName, m.DisplayName

	case "discord":
		var d struct {
			ID        string ` + "`" + `json:"id"` + "`" + `
			Email     string ` + "`" + `json:"email"` + "`" + `
			Username  string ` + "`" + `json:"username"` + "`" + `
			Avatar    string ` + "`" + `json:"avatar"` + "`" + `
			Discriminator string ` + "`" + `json:"discriminator"` + "`" + `
		}
		if err := json.Unmarshal(body, &d); err != nil {
			return nil, err
		}
		info.ID = d.ID
		info.Email = d.Email
		info.Name = d.Username + "#" + d.Discriminator
		info.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", d.ID, d.Avatar)

	case "telegram":
		// Telegram uses Login Widget, not standard OAuth2
		// The response format is different from other providers
		var t struct {
			OK bool ` + "`" + `json:"ok"` + "`" + `
			Result struct {
				ID        int64  ` + "`" + `json:"id"` + "`" + `
				FirstName string ` + "`" + `json:"first_name"` + "`" + `
				LastName  string ` + "`" + `json:"last_name"` + "`" + `
				Username  string ` + "`" + `json:"username"` + "`" + `
				PhotoURL  string ` + "`" + `json:"photo_url"` + "`" + `
			} ` + "`" + `json:"result"` + "`" + `
		}
		if err := json.Unmarshal(body, &t); err != nil {
			return nil, err
		}
		if !t.OK {
			return nil, fmt.Errorf("telegram API error")
		}
		info.ID = fmt.Sprintf("%d", t.Result.ID)
		info.Name = t.Result.FirstName + " " + t.Result.LastName
		info.AvatarURL = t.Result.PhotoURL
		// Telegram doesn't provide email via bot API

	case "twitter":
		var t struct {
			Data struct {
				ID            string ` + "`" + `json:"id"` + "`" + `
				Name          string ` + "`" + `json:"name"` + "`" + `
				Username      string ` + "`" + `json:"username"` + "`" + `
				ProfileImageURL string ` + "`" + `json:"profile_image_url"` + "`" + `
			} ` + "`" + `json:"data"` + "`" + `
		}
		if err := json.Unmarshal(body, &t); err != nil {
			return nil, err
		}
		info.ID = t.Data.ID
		info.Name = t.Data.Name
		info.AvatarURL = t.Data.ProfileImageURL
		// Twitter doesn't provide email by default
	}

	return info, nil
}
`)

	return os.WriteFile(filepath.Join(outputDir, "oauth.go"), []byte(sb.String()), 0600)
}

// generateProviderConfig generates OAuth config for a provider.
// IMPORTANT: All credentials are read from environment variables at runtime.
// This prevents secrets from being embedded in generated source code.
func (p *AuthPlugin) generateProviderConfig(provider OAuthProvider) string {
	scopes := "[]string{"
	for i, scope := range provider.Scopes {
		if i > 0 {
			scopes += ", "
		}
		scopes += fmt.Sprintf(`"%s"`, scope)
	}
	scopes += "}"

	// All providers now read credentials from environment variables
	// This prevents secrets from being committed to version control
	switch provider.Name {
	case "Google":
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint:     endpoints.Google,
	}
}
`, provider.Name, scopes)
	case "Facebook":
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint:     endpoints.Facebook,
	}
}
`, provider.Name, scopes)
	case "GitHub":
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint:     endpoints.GitHub,
	}
}
`, provider.Name, scopes)
	case "Microsoft":
		return fmt.Sprintf(`
func getMicrosoftConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
	}
}
`, scopes)
	case "Discord":
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
}
`, provider.Name, scopes)
	case "Telegram":
		// Telegram uses Login Widget, not standard OAuth2
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	// Telegram uses Login Widget with hash verification, not standard OAuth2
	// Bot token read from TELEGRAM_BOT_TOKEN environment variable
	return &oauth2.Config{
		ClientID:     os.Getenv("TELEGRAM_BOT_TOKEN"),
		ClientSecret: "",
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "",
			TokenURL: "",
		},
	}
}

// VerifyTelegramAuth verifies Telegram Login Widget authentication
// See: https://core.telegram.org/widgets/login#checking-authorization
func VerifyTelegramAuth(authData map[string]string, botToken string) bool {
	// 1. Extract and remove the hash from auth data
	hash, ok := authData["hash"]
	if !ok || hash == "" {
		return false
	}
	
	// 2. Sort remaining keys alphabetically
	keys := make([]string, 0, len(authData)-1)
	for k := range authData {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	
	// 3. Build data-check string from key=value pairs
	var dataCheck strings.Builder
	for i, k := range keys {
		if i > 0 {
			dataCheck.WriteString("\n")
		}
		dataCheck.WriteString(k)
		dataCheck.WriteString("=")
		dataCheck.WriteString(authData[k])
	}
	
	// 4. Compute SHA256 of bot token
	secretKey := sha256.Sum256([]byte(botToken))
	
	// 5. Compute HMAC-SHA256 of data-check string
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheck.String()))
	computedHash := hex.EncodeToString(h.Sum(nil))
	
	// 6. Compare with provided hash (constant-time comparison)
	return hmac.Equal([]byte(hash), []byte(computedHash))
}
`, provider.Name)
	case "Twitter":
		// Twitter uses OAuth 2.0 with PKCE
		return fmt.Sprintf(`
func get%sConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("TWITTER_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITTER_CLIENT_SECRET"),
		Scopes:       %s,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
	}
}

// getTwitterPKCEConfig returns OAuth2 config with PKCE for Twitter
func getTwitterPKCEConfig() *oauth2.Config {
	return getTwitterConfig()
}
`, provider.Name, scopes)
	}

	return ""
}

// generateOTPCode generates OTP utilities.
func (p *AuthPlugin) generateOTPCode(outputDir string) error {
	code := `// Auto-generated OTP utilities
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	otpDigits   = ` + fmt.Sprintf("%d", p.config.OTPDigits) + `
	otpPeriod   = ` + fmt.Sprintf("%d", p.config.OTPPeriod) + `
	backupCodeLength = 8
)

// GenerateOTPSecret generates a new OTP secret.
func GenerateOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateOTPURL generates the otpauth:// URL for QR codes.
func GenerateOTPURL(secret, account, issuer string) string {
	u := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   fmt.Sprintf("/%s:%s", issuer, account),
	}
	q := u.Query()
	q.Set("secret", strings.TrimSuffix(secret, "="))
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", otpDigits))
	q.Set("period", fmt.Sprintf("%d", otpPeriod))
	u.RawQuery = q.Encode()
	return u.String()
}

// VerifyOTP verifies a TOTP code against a secret.
func VerifyOTP(secret, code string) bool {
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}

	// Check current and adjacent time windows
	now := time.Now().Unix() / int64(otpPeriod)
	for i := -1; i <= 1; i++ {
		if generateOTP(key, now+int64(i)) == code {
			return true
		}
	}
	return false
}

// generateOTP generates a TOTP code.
func generateOTP(key []byte, counter int64) string {
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte{
		byte(counter >> 56), byte(counter >> 48), byte(counter >> 40),
		byte(counter >> 32), byte(counter >> 24), byte(counter >> 16),
		byte(counter >> 8), byte(counter),
	})
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]) << 16) |
		(int(hash[offset+2]) << 8) |
		int(hash[offset+3])
	code = code % int(math.Pow10(otpDigits))

	return fmt.Sprintf("%0` + fmt.Sprintf("%d", p.config.OTPDigits) + `d", code)
}

// GenerateBackupCodes generates backup codes for 2FA.
func GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

// generateBackupCode generates a single backup code.
func generateBackupCode() (string, error) {
	bytes := make([]byte, backupCodeLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := hex.EncodeToString(bytes)
	return code[:4] + "-" + code[4:], nil
}

// HashBackupCode hashes a backup code for storage using SHA256.
func HashBackupCode(code string) string {
	code = strings.ReplaceAll(code, "-", "")
	h := sha256.New()
	h.Write([]byte(code))
	return hex.EncodeToString(h.Sum(nil))
}
`
	return os.WriteFile(filepath.Join(outputDir, "otp.go"), []byte(code), 0600)
}

// getEnabledProviders returns enabled OAuth providers.
func (p *AuthPlugin) getEnabledProviders() []OAuthProvider {
	providers := map[string]OAuthProvider{
		"google": { // #nosec //nolint:gosec
			Name:         "Google",
			ClientID:     p.config.GoogleClientID,
			ClientSecret: p.config.GoogleClientSecret,
			AuthURL:      "https://accounts.google.com/o/oauth2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserURL:      "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:       []string{"openid", "email", "profile"},
		},
		"facebook": { // #nosec //nolint:gosec
			Name:         "Facebook",
			ClientID:     p.config.FacebookClientID,
			ClientSecret: p.config.FacebookClientSecret,
			AuthURL:      "https://www.facebook.com/v18.0/dialog/oauth",
			TokenURL:     "https://graph.facebook.com/v18.0/oauth/access_token",
			UserURL:      "https://graph.facebook.com/me?fields=id,email,name,picture",
			Scopes:       []string{"email", "public_profile"},
		},
		"github": { // #nosec //nolint:gosec
			Name:         "GitHub",
			ClientID:     p.config.GitHubClientID,
			ClientSecret: p.config.GitHubClientSecret,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserURL:      "https://api.github.com/user",
			Scopes:       []string{"user:email"},
		},
		// #nosec //nolint:gosec
		"microsoft": {
			Name:         "Microsoft",
			ClientID:     p.config.MicrosoftClientID,
			ClientSecret: p.config.MicrosoftClientSecret,
			AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserURL:      "https://graph.microsoft.com/v1.0/me",
			Scopes:       []string{"openid", "email", "profile"},
		},
		// #nosec //nolint:gosec
		"discord": {
			Name:         "Discord",
			ClientID:     p.config.DiscordClientID,
			ClientSecret: p.config.DiscordClientSecret,
			AuthURL:      "https://discord.com/api/oauth2/authorize",
			TokenURL:     "https://discord.com/api/oauth2/token",
			UserURL:      "https://discord.com/api/users/@me",
			Scopes:       []string{"identify", "email"},
		},
		"telegram": {
			Name:         "Telegram",
			ClientID:     p.config.TelegramBotToken,
			ClientSecret: "", // Telegram uses bot token validation, not client secret
			AuthURL:      "", // Telegram uses widget-based login
			TokenURL:     "",
			UserURL:      "https://api.telegram.org/bot" + p.config.TelegramBotToken + "/getMe",
			Scopes:       []string{},
		},
		// #nosec //nolint:gosec
		"twitter": {
			Name:         "Twitter",
			ClientID:     p.config.TwitterClientID,
			ClientSecret: p.config.TwitterClientSecret,
			AuthURL:      "https://twitter.com/i/oauth2/authorize",
			TokenURL:     "https://api.twitter.com/2/oauth2/token",
			UserURL:      "https://api.twitter.com/2/users/me",
			Scopes:       []string{"tweet.read", "users.read"},
		},
	}

	var enabled []OAuthProvider
	for _, name := range p.config.OAuthProviders {
		if provider, ok := providers[name]; ok {
			enabled = append(enabled, provider)
		}
	}
	return enabled
}

// generateSecret generates a random secret.
func (p *AuthPlugin) generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateOTPSetup generates OTP setup info.
func (p *AuthPlugin) generateOTPSetup(account string) error {
	secret, err := p.GenerateOTPSecret()
	if err != nil {
		return err
	}

	url := p.GenerateOTPURL(secret, account, p.config.OTPIssuer)

	fmt.Println("OTP Setup:")
	fmt.Printf("  Secret: %s\n", secret)
	fmt.Printf("  Account: %s\n", account)
	fmt.Printf("  Issuer: %s\n", p.config.OTPIssuer)
	fmt.Printf("  URL: %s\n", url)
	fmt.Println("\nUse this URL to generate a QR code for authenticator apps.")

	return nil
}

// generateBackupCodes generates and prints backup codes.
func (p *AuthPlugin) generateBackupCodes(count int) error {
	codes, err := GenerateBackupCodes(count)
	if err != nil {
		return err
	}

	fmt.Println("Backup Codes (store these securely):")
	for _, code := range codes {
		fmt.Printf("  %s\n", code)
	}
	fmt.Println("\nHashed versions for database storage:")
	for _, code := range codes {
		fmt.Printf("  %s -> %s\n", code, HashBackupCode(code))
	}

	return nil
}

// verifyOTP verifies an OTP code.
func (p *AuthPlugin) verifyOTP(secret, code string) error {
	if p.VerifyOTP(secret, code) {
		fmt.Println("✓ Code is valid")
		return nil
	}
	fmt.Println("✗ Code is invalid")
	return fmt.Errorf("invalid OTP code")
}

// GenerateOTPSecret generates a new OTP secret.
func (p *AuthPlugin) GenerateOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateOTPURL generates the otpauth:// URL.
func (p *AuthPlugin) GenerateOTPURL(secret, account, issuer string) string {
	u := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   fmt.Sprintf("/%s:%s", issuer, account),
	}
	q := u.Query()
	q.Set("secret", strings.TrimSuffix(secret, "="))
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", p.config.OTPDigits))
	q.Set("period", fmt.Sprintf("%d", p.config.OTPPeriod))
	u.RawQuery = q.Encode()
	return u.String()
}

// GenerateOTP generates an OTP secret and URL for 2FA setup.
func (p *AuthPlugin) GenerateOTP(account string) (string, string, error) {
	secret, err := p.GenerateOTPSecret()
	if err != nil {
		return "", "", err
	}
	url := p.GenerateOTPURL(secret, account, p.config.OTPIssuer)
	return secret, url, nil
}

// VerifyOTP verifies a TOTP code.
func (p *AuthPlugin) VerifyOTP(secret, code string) bool {
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}

	now := time.Now().Unix() / int64(p.config.OTPPeriod)
	for i := -1; i <= 1; i++ {
		if p.generateOTP(key, now+int64(i)) == code {
			return true
		}
	}
	return false
}

// generateOTP generates a TOTP code.
//
// #nosec //nolint:gosec // intentional conversion for bit operations
func (p *AuthPlugin) generateOTP(key []byte, counter int64) string {
	mac := hmac.New(sha1.New, key)
	c := uint64(counter)
	mac.Write([]byte{
		byte(c >> 56), byte(c >> 48), byte(c >> 40),
		byte(c >> 32), byte(c >> 24), byte(c >> 16),
		byte(c >> 8), byte(c),
	})
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]) << 16) |
		(int(hash[offset+2]) << 8) |
		int(hash[offset+3])
	code %= int(math.Pow10(p.config.OTPDigits))

	return fmt.Sprintf(fmt.Sprintf("%%0%dd", p.config.OTPDigits), code)
}

// GenerateBackupCodes generates backup codes.
func GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		code := hex.EncodeToString(bytes)
		codes[i] = code[:6] + "-" + code[6:]
	}
	return codes, nil
}

// HashBackupCode hashes a backup code using HMAC-SHA256 with a per-code salt.
func HashBackupCode(code string) string {
	code = strings.ReplaceAll(code, "-", "")
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic("failed to generate salt for backup code hash")
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(code))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyBackupCode checks whether a plaintext code matches a stored salt:hash.
func VerifyBackupCode(code, storedHash string) bool {
	parts := strings.SplitN(storedHash, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	code = strings.ReplaceAll(code, "-", "")
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(code))
	return hmac.Equal(mac.Sum(nil), expectedHash)
}

// GetConfig returns the current configuration.
func (p *AuthPlugin) GetConfig() *Config {
	return p.config
}

// Ensure AuthPlugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*AuthPlugin)(nil)
