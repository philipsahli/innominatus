# OIDC Authentication

**Date:** 2025-10-06
**Status:** ✅ Production Ready
**Feature:** OpenID Connect (OIDC) Authentication with Keycloak Integration

---

## Overview

innominatus supports enterprise-grade authentication through OpenID Connect (OIDC) integration with identity providers like Keycloak. This enables Single Sign-On (SSO) for the Web UI and API key management for OIDC-authenticated users.

### Key Features

- **SSO Login**: "Login with Keycloak" button in Web UI
- **Dual Authentication**: Session-based (Web UI) and API key-based (CLI/API)
- **Database-Backed API Keys**: Separate storage for OIDC users vs file-based local users
- **Secure Token Storage**: SHA-256 hashing for API keys, HttpOnly cookies for sessions
- **Profile Management**: Self-service API key generation and revocation
- **Production Ready**: Integrates with enterprise identity providers

---

## Architecture

### Authentication Flow

```
┌─────────────┐
│   Browser   │
│  (Web UI)   │
└──────┬──────┘
       │ 1. Click "Login with Keycloak"
       ▼
┌──────────────────┐
│  innominatus     │
│  /auth/oidc/login│
└──────┬───────────┘
       │ 2. Redirect to Keycloak
       ▼
┌─────────────────────┐
│    Keycloak IdP     │
│  (OIDC Provider)    │
└──────┬──────────────┘
       │ 3. User authenticates
       │ 4. Authorization code
       ▼
┌──────────────────────┐
│  innominatus         │
│  /auth/oidc/callback │
└──────┬───────────────┘
       │ 5. Exchange code for token
       │ 6. Create session
       │ 7. Redirect with session ID
       ▼
┌──────────────────┐
│  Web UI          │
│  /auth/oidc/     │
│  callback page   │
└──────┬───────────┘
       │ 8. Store token in localStorage
       │ 9. Redirect to dashboard
       ▼
┌──────────────────┐
│   Dashboard      │
│   (Authenticated)│
└──────────────────┘
```

### Database Architecture

**Local Users** (users.yaml):
- API keys stored in YAML file
- Traditional file-based authentication
- Used for admin and service accounts

**OIDC Users** (PostgreSQL):
```sql
Table: user_api_keys
├── id (SERIAL)
├── username (VARCHAR) - from OIDC preferred_username claim
├── key_hash (VARCHAR) - SHA-256 hash of API key
├── key_name (VARCHAR) - user-provided key name
├── created_at (TIMESTAMP)
├── last_used_at (TIMESTAMP)
└── expires_at (TIMESTAMP)
```

---

## Configuration

### Server Configuration

#### Environment Variables

```bash
# Enable OIDC authentication
export OIDC_ENABLED=true

# Keycloak configuration (required if OIDC_ENABLED=true)
export OIDC_ISSUER="https://keycloak.company.com/realms/production"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="your-client-secret-here"
export OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback"

# Database configuration (required for OIDC users' API keys)
export DB_HOST="postgres.production.internal"
export DB_PORT="5432"
export DB_USER="orchestrator_service"
export DB_PASSWORD="secure_password"
export DB_NAME="idp_orchestrator"
export DB_SSLMODE="require"
```

#### Keycloak Client Setup

1. **Create OIDC Client** in Keycloak admin console:
   - Client ID: `innominatus`
   - Client Protocol: `openid-connect`
   - Access Type: `confidential`
   - Valid Redirect URIs:
     - `https://innominatus.company.com/auth/oidc/callback`
     - `http://localhost:8081/auth/oidc/callback` (development)

2. **Configure Client Scopes**:
   - `openid` (required)
   - `profile` (recommended)
   - `email` (recommended)

3. **Protocol Mappers**:
   - `preferred_username` - Username claim (required)
   - `email` - Email address
   - `given_name` - First name
   - `family_name` - Last name

4. **Get Client Secret**:
   - Credentials tab → Copy `Secret` value
   - Set as `OIDC_CLIENT_SECRET` environment variable

### Start Server with OIDC

```bash
# Production
OIDC_ENABLED=true \
  OIDC_ISSUER="https://keycloak.company.com/realms/production" \
  OIDC_CLIENT_ID="innominatus" \
  OIDC_CLIENT_SECRET="your-secret" \
  OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback" \
  DB_HOST="postgres.internal" \
  DB_PASSWORD="..." \
  ./innominatus

# Development (with demo Keycloak)
OIDC_ENABLED=true ./innominatus
# Uses default demo configuration:
#   Issuer: http://keycloak.localtest.me/realms/demo-realm
#   Client ID: innominatus-web
#   Redirect: http://localhost:8081/auth/oidc/callback
```

---

## Authentication Methods

### 1. Web UI Login (Session-Based)

**User Flow:**
1. Navigate to http://innominatus.company.com/login
2. Click "Login with Keycloak" button
3. Redirect to Keycloak login page
4. Authenticate with corporate credentials
5. Redirect back to innominatus dashboard
6. Session stored in HttpOnly cookie + localStorage token

**Session Management:**
- Session ID stored in HttpOnly cookie (secure, not accessible to JavaScript)
- Session token also stored in localStorage (for API calls)
- Dual storage enables both cookie-based and token-based authentication

**Endpoints:**
```bash
# Check if OIDC is enabled
GET /api/auth/config
# Response: {"oidc_enabled": true, "oidc_provider_name": "Keycloak"}

# Initiate OIDC login
GET /auth/oidc/login
# Redirects to Keycloak

# OIDC callback (handles authorization code)
GET /auth/oidc/callback?code=...
# Creates session and redirects to /auth/oidc/callback?token=SESSION_ID
```

### 2. API Key Authentication (CLI/API Access)

**OIDC users** cannot use username/password for API access. They must generate API keys:

#### Generate API Key

**Web UI:**
1. Login via OIDC
2. Navigate to Profile page
3. Click "Generate New API Key"
4. Provide key name and expiry days
5. Copy the generated key (shown only once)

**API:**
```bash
# Generate API key (requires active session)
curl -X POST http://innominatus.company.com/api/profile/api-keys \
  -H "Cookie: session_id=YOUR_SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "cli-key",
    "expiry_days": 90
  }'

# Response:
{
  "key": "a1b2c3d4e5f6...64-char-hex-key",
  "name": "cli-key",
  "created_at": "2025-10-06T10:00:00Z",
  "expires_at": "2026-01-04T10:00:00Z"
}
```

#### Use API Key

```bash
# CLI usage with API key
export IDP_API_KEY="a1b2c3d4e5f6...64-char-hex-key"

# API requests
curl -H "Authorization: Bearer $IDP_API_KEY" \
  http://innominatus.company.com/api/specs
```

#### List API Keys

```bash
# List your API keys (masked for security)
curl -H "Cookie: session_id=YOUR_SESSION_ID" \
  http://innominatus.company.com/api/profile/api-keys

# Response:
[
  {
    "name": "cli-key",
    "masked_key": "...e5f6a1b2",
    "created_at": "2025-10-06T10:00:00Z",
    "last_used_at": "2025-10-06T14:30:00Z",
    "expires_at": "2026-01-04T10:00:00Z"
  }
]
```

#### Revoke API Key

```bash
# Revoke API key by name
curl -X DELETE \
  -H "Cookie: session_id=YOUR_SESSION_ID" \
  http://innominatus.company.com/api/profile/api-keys/cli-key

# Response: 204 No Content
```

---

## Dual-Source Authentication

innominatus supports **two types of users** with different authentication backends:

### Local Users (File-Based)

**Source:** `users.yaml` file

**Characteristics:**
- Username/password authentication
- API keys stored in users.yaml
- Admin and service accounts
- Always available (no database required)

**Example:**
```yaml
# users.yaml
users:
  - username: admin
    password: admin123  # hashed in production
    team: platform
    role: admin
    api_keys:
      - key: "local-api-key-hash"
        name: "admin-cli"
        created_at: "2025-01-01T00:00:00Z"
        expires_at: "2026-01-01T00:00:00Z"
```

### OIDC Users (Database-Backed)

**Source:** PostgreSQL `user_api_keys` table

**Characteristics:**
- OIDC SSO authentication (no password in innominatus)
- API keys stored in database
- Enterprise user accounts
- Requires PostgreSQL connection

**Authentication Flow:**
1. User logs in via OIDC → username from `preferred_username` claim
2. innominatus checks if user exists in `users.yaml` → NOT FOUND
3. System identifies as OIDC user → stores API keys in database
4. API key authentication → checks database for key hash

### Automatic User Type Detection

The system automatically detects user type:

```go
// Pseudo-code logic
store, _ := users.LoadUsers()
_, err := store.GetUser(username)

if err != nil {
    // User not found in users.yaml → OIDC user
    // Use database for API keys
    db.CreateAPIKey(username, keyHash, keyName, expiresAt)
} else {
    // User found in users.yaml → Local user
    // Use file-based API keys
    store.GenerateAPIKey(username, keyName, expiryDays)
}
```

---

## API Endpoints

### Authentication Endpoints

| Endpoint | Method | Description | Authentication |
|----------|--------|-------------|----------------|
| `/api/auth/config` | GET | Get OIDC configuration | None (public) |
| `/auth/oidc/login` | GET | Initiate OIDC login | None |
| `/auth/oidc/callback` | GET | OIDC callback handler | None (code exchange) |
| `/api/login` | POST | Local user login | None |
| `/api/logout` | POST | Logout (clear session) | Session |

### Profile & API Key Management

| Endpoint | Method | Description | Authentication |
|----------|--------|-------------|----------------|
| `/api/profile` | GET | Get user profile | Session or API Key |
| `/api/profile/api-keys` | GET | List user's API keys | Session |
| `/api/profile/api-keys` | POST | Generate new API key | Session |
| `/api/profile/api-keys/{name}` | DELETE | Revoke API key | Session |

---

## Security Considerations

### Production Deployment

**1. Use HTTPS for all communication:**
```bash
# Never use HTTP in production
OIDC_ISSUER="https://keycloak.company.com/realms/production"
OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback"
```

**2. Secure client secret:**
```bash
# Use secrets management (Vault, Kubernetes Secrets, etc.)
OIDC_CLIENT_SECRET=$(kubectl get secret oidc-secret -o jsonpath='{.data.client-secret}' | base64 -d)
```

**3. Database SSL:**
```bash
# Require SSL for database connections
DB_SSLMODE="require"
```

**4. Token storage best practices:**
- Session cookies: HttpOnly, Secure, SameSite=Strict
- API keys: SHA-256 hashed in database (never store plaintext)
- localStorage tokens: Used only for frontend API calls

### API Key Security

**SHA-256 Hashing:**
```go
// API keys are hashed before storage
hash := sha256.Sum256([]byte(apiKey))
keyHash := hex.EncodeToString(hash[:])
// Only hash is stored in database
```

**Key Rotation:**
```bash
# Generate new key before old one expires
# Revoke old key after transition period
```

**Least Privilege:**
- Generate separate API keys for different use cases
- Use descriptive names (e.g., "ci-pipeline", "local-dev")
- Set appropriate expiry dates (30-90 days recommended)

### Session Security

**HttpOnly Cookies:**
- Prevents JavaScript access to session ID
- Mitigates XSS attacks
- Automatically sent with requests

**Session Expiry:**
- Default session lifetime: 24 hours
- Extended on user activity
- Expired sessions automatically cleaned up

---

## Troubleshooting

### OIDC Login Not Working

**Symptom:** "Login with Keycloak" button not appearing

**Solution:**
```bash
# Check server logs for OIDC initialization
# Should see: "OIDC authentication enabled"

# Verify environment variable
echo $OIDC_ENABLED  # Should be "true"

# Test auth config endpoint
curl http://localhost:8081/api/auth/config
# Should return: {"oidc_enabled": true, "oidc_provider_name": "Keycloak"}
```

**Symptom:** Redirect loop or "Invalid redirect URI"

**Solution:**
1. Check Keycloak client configuration
2. Verify `OIDC_REDIRECT_URL` matches Keycloak's "Valid Redirect URIs"
3. Ensure protocol matches (HTTP vs HTTPS)

```bash
# Check redirect URL configuration
echo $OIDC_REDIRECT_URL

# Must match Keycloak client config exactly:
# Keycloak → Clients → innominatus → Valid Redirect URIs
```

### API Key Generation Fails

**Symptom:** "400 user 'demo-user' not found"

**Cause:** Database not connected, OIDC user trying to use file-based auth

**Solution:**
```bash
# Verify database connection
# Check server logs for:
# "Database connected successfully"
# "Successfully executed 5 migration(s)"

# Verify database environment variables
echo $DB_HOST
echo $DB_NAME

# Test database connection
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT * FROM user_api_keys LIMIT 1;"
```

**Symptom:** API key not working after generation

**Solution:**
```bash
# Verify API key format (64 hex characters)
echo $IDP_API_KEY | wc -c  # Should be 64

# Test API key authentication
curl -H "Authorization: Bearer $IDP_API_KEY" \
  http://innominatus.company.com/api/health

# Check last_used_at timestamp (should update on each use)
```

### Session Not Persisting

**Symptom:** Logged out immediately after OIDC login

**Cause:** Frontend not storing token correctly

**Solution:**
1. Check browser console for errors
2. Verify callback page loads (`/auth/oidc/callback`)
3. Check localStorage for `auth-token`

```javascript
// Browser console
localStorage.getItem('auth-token')  // Should have session ID
```

---

## Migration Guide

### From File-Based to OIDC Authentication

**Step 1: Deploy Database**
```bash
# Create PostgreSQL database
createdb idp_orchestrator

# Migrations run automatically on server startup
```

**Step 2: Configure Keycloak**
- Create realm, users, and OIDC client (see Configuration section)

**Step 3: Update Environment Variables**
```bash
# Enable OIDC
export OIDC_ENABLED=true
export OIDC_ISSUER="https://keycloak.company.com/realms/production"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="your-secret"
export OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback"
```

**Step 4: Migrate Users**
```bash
# File-based users remain in users.yaml (admin, service accounts)
# Enterprise users authenticate via OIDC
# Both types coexist seamlessly
```

**Step 5: Generate API Keys**
```bash
# OIDC users: Login via Web UI → Profile → Generate API Key
# Local users: Continue using existing API keys in users.yaml
```

---

## API Examples

### Complete OIDC User Workflow

```bash
# 1. Check OIDC status
curl http://innominatus.company.com/api/auth/config
# {"oidc_enabled": true, "oidc_provider_name": "Keycloak"}

# 2. Login via Web UI (browser)
# - Click "Login with Keycloak"
# - Authenticate with corporate credentials
# - Redirected to dashboard
# - Session cookie + localStorage token created

# 3. Generate API key (Web UI or API)
curl -X POST http://innominatus.company.com/api/profile/api-keys \
  -H "Cookie: session_id=YOUR_SESSION" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-cli-key", "expiry_days": 90}'

# Copy the generated key (shown only once!)
export IDP_API_KEY="a1b2c3d4e5f6...64-char-hex"

# 4. Use API key for CLI/API access
curl -H "Authorization: Bearer $IDP_API_KEY" \
  http://innominatus.company.com/api/specs

# 5. List API keys
curl -H "Authorization: Bearer $IDP_API_KEY" \
  http://innominatus.company.com/api/profile/api-keys

# 6. Revoke API key when done
curl -X DELETE \
  -H "Cookie: session_id=YOUR_SESSION" \
  http://innominatus.company.com/api/profile/api-keys/my-cli-key
```

---

## References

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [RFC 6749 - OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc6749)
- [API Security Best Practices](./API_SECURITY_PHASE1.md)
- [Keycloak Demo Integration](../KEYCLOAK-DEMO-INTEGRATION.md)

---

**Status:** ✅ Production Ready
**Last Updated:** 2025-10-06
**Version:** v1.0.0
