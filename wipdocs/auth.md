# Auth Middleware: JWT + Trusted Header + API Key

## Context
The music app has no authentication — all API routes are fully open. The DB schema already has `users` and `api_keys` tables but they aren't wired to any HTTP layer. The goal is to gate the API and music file routes behind configurable authentication, driven entirely by Docker environment variables and supporting three auth methods: JWT Bearer tokens, trusted proxy headers, and DB-backed API keys.

---

## Scope

**Protected routes:** `/api/*` and `/music/*`
**Always public:** `/`, `/favicon.ico`, `/static/*`, `/ui/*` — the SPA must load in the browser before it can supply credentials

---

## New Env Vars

| Var | Default | Description |
|-----|---------|-------------|
| `AUTH_ENABLED` | `false` | Master toggle. No breaking change for existing installs. |
| `AUTH_JWT_SECRET` | — | HMAC secret for HS256 JWT validation |
| `AUTH_JWT_JWKS_URL` | — | JWKS endpoint URL for RS256/ES256 (external IdP) |
| `AUTH_JWT_ISSUER` | — | Optional: validate `iss` claim |
| `AUTH_JWT_AUDIENCE` | — | Optional: validate `aud` claim |
| `AUTH_TRUSTED_HEADER` | — | Header name trusted from upstream proxy (e.g. `X-Remote-User`) |
| `AUTH_TRUSTED_HEADER_VALUE` | — | Optional: header must equal this exact value |
| `AUTH_API_KEYS_ENABLED` | `false` | Validate `Authorization: Bearer <key>` against `api_keys` DB table |

Auth methods are tried in order: trusted header → JWT → API key. First success wins.

---

## New Go Dependency

`github.com/golang-jwt/jwt/v5` — JWT parsing (HS256/RS256/ES256). Minimal, no transitive deps.

```
cd /workspace/playground/web/music && go get github.com/golang-jwt/jwt/v5
```

---

## Implementation — all in `cmd/musicd/main.go`

### 1. Auth config struct (near top, after other globals)

```go
type authConfig struct {
    Enabled            bool
    JWTSecret          []byte
    JWKSUrl            string
    JWTIssuer          string
    JWTAudience        string
    TrustedHeader      string
    TrustedHeaderValue string
    APIKeysEnabled     bool
}
var authCfg authConfig
```

### 2. JWKS key cache (for RS256/ES256 from external IdP)

```go
type jwksCache struct {
    mu   sync.RWMutex
    keys map[string]crypto.PublicKey // kid → public key
}
var globalJWKSCache = &jwksCache{keys: make(map[string]crypto.PublicKey)}

func fetchJWKS(url string) error {
    // HTTP GET the JWKS URL
    // Parse JSON: {"keys":[{"kty":"RSA","kid":"...","n":"...","e":"..."},...]}
    // Build *rsa.PublicKey / *ecdsa.PublicKey using stdlib (crypto/rsa, math/big, base64url)
    // Store in globalJWKSCache.keys under kid
}
```

Uses only stdlib (`net/http`, `encoding/json`, `crypto/rsa`, `crypto/elliptic`, `math/big`). No extra deps for JWKS.

Keys are refreshed every 1 hour via background goroutine started in `main()`.

### 3. Load auth config in `main()` (after existing env var reads)

```go
authCfg = authConfig{
    Enabled:            os.Getenv("AUTH_ENABLED") == "true",
    JWTSecret:          []byte(os.Getenv("AUTH_JWT_SECRET")),
    JWKSUrl:            os.Getenv("AUTH_JWT_JWKS_URL"),
    JWTIssuer:          os.Getenv("AUTH_JWT_ISSUER"),
    JWTAudience:        os.Getenv("AUTH_JWT_AUDIENCE"),
    TrustedHeader:      os.Getenv("AUTH_TRUSTED_HEADER"),
    TrustedHeaderValue: os.Getenv("AUTH_TRUSTED_HEADER_VALUE"),
    APIKeysEnabled:     os.Getenv("AUTH_API_KEYS_ENABLED") == "true",
}
if authCfg.Enabled && authCfg.JWKSUrl != "" {
    go startJWKSRefreshLoop(authCfg.JWKSUrl)
}
```

### 4. Auth middleware

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !authCfg.Enabled || tryAuth(r) {
            next.ServeHTTP(w, r)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
    })
}

func tryAuth(r *http.Request) bool {
    // 1. Trusted header
    if authCfg.TrustedHeader != "" {
        val := r.Header.Get(authCfg.TrustedHeader)
        if val != "" && (authCfg.TrustedHeaderValue == "" || val == authCfg.TrustedHeaderValue) {
            return true
        }
    }
    // 2. JWT Bearer token
    if len(authCfg.JWTSecret) > 0 || authCfg.JWKSUrl != "" {
        if token := extractBearer(r); token != "" && validateJWT(token) == nil {
            return true
        }
    }
    // 3. DB API key
    if authCfg.APIKeysEnabled {
        if key := extractBearer(r); key != "" && validateAPIKey(key) == nil {
            return true
        }
    }
    return false
}
```

`validateJWT`: uses `github.com/golang-jwt/jwt/v5` with a `keyFunc` that returns `authCfg.JWTSecret` for HMAC or looks up `kid` in `globalJWKSCache` for RSA/EC. Validates `exp`, `iss`, `aud` if configured.

`validateAPIKey`: `SELECT user_id FROM api_keys WHERE api_key = $1 AND is_active = true`

### 5. Apply middleware in router setup

```go
// API routes (currently):
api := r.PathPrefix(prefixPath("/api")).Subrouter()

// Change to:
api := r.PathPrefix(prefixPath("/api")).Subrouter()
api.Use(authMiddleware)

// Music file streaming:
musicHandler := authMiddleware(http.StripPrefix(musicPath, http.FileServer(http.Dir(musicDir))))
r.PathPrefix(musicPath).Handler(musicHandler)
```

---

## Critical Files

- `cmd/musicd/main.go` — all backend changes
- `go.mod` / `go.sum` — add `github.com/golang-jwt/jwt/v5`
- `docker-compose.yml` — document new env vars with comments

---

## docker-compose.yml example additions

```yaml
environment:
  # Authentication (disabled by default)
  AUTH_ENABLED: "false"
  # JWT: set either SECRET (HS256) or JWKS_URL (RS256/ES256), not both
  # AUTH_JWT_SECRET: "your-hmac-secret-here"
  # AUTH_JWT_JWKS_URL: "https://your-idp.example.com/.well-known/jwks.json"
  # AUTH_JWT_ISSUER: "https://your-idp.example.com/"
  # AUTH_JWT_AUDIENCE: "musicd"
  # Trusted proxy header (e.g. from nginx/Traefik with auth_request)
  # AUTH_TRUSTED_HEADER: "X-Remote-User"
  # AUTH_TRUSTED_HEADER_VALUE: ""  # leave empty to allow any non-empty value
  # DB API keys
  # AUTH_API_KEYS_ENABLED: "true"
```

---

## Verification

1. Default (`AUTH_ENABLED` unset) — existing behavior unchanged, all routes open
2. `AUTH_ENABLED=true` only — all `/api/*` and `/music/*` return 401; `/ui/*` still loads
3. Trusted header: `curl -H "X-Remote-User: alice" /api/tracks` → 200
4. Trusted header with value check: wrong value → 401, correct value → 200
5. HS256 JWT: generate token at jwt.io with shared secret → `Authorization: Bearer <token>` → 200
6. RS256 JWT: test with a local JWKS mock or real IdP → 200
7. API key: `INSERT INTO api_keys (api_key, user_id, is_active) VALUES (...)` → `Bearer <key>` → 200
8. Expired JWT → 401
9. `/ui/tracks` (SPA page) → 200 even with auth enabled (no auth required for static)
10. `/music/file.mp3` → 401 without auth, 200 with valid credentials

