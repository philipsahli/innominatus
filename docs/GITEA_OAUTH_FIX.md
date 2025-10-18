# Gitea OAuth2 Auto-Registration Fix

## Problem

When logging into Gitea via Keycloak OIDC, users encountered the error:

```
Registration is disabled. Please contact your site administrator.
```

This occurred even though:
- Gitea was configured with `ENABLE_AUTO_REGISTRATION: true`
- Gitea was configured with `ALLOW_ONLY_EXTERNAL_REGISTRATION: true`
- Keycloak OIDC client was properly configured

## Root Cause

**The issue was:**

**Incorrect `DISABLE_REGISTRATION` setting:** The Gitea configuration had `DISABLE_REGISTRATION: true`, which **blocks all registration** including external OAuth2 registration.

According to Gitea documentation:
> `ALLOW_ONLY_EXTERNAL_REGISTRATION` only works when `DISABLE_REGISTRATION` is set to `false`

The incorrect configuration was:
```yaml
service:
  DISABLE_REGISTRATION: true  # ← This blocks ALL registration!
  ALLOW_ONLY_EXTERNAL_REGISTRATION: true  # ← Ignored when DISABLE_REGISTRATION is true
```

**Note about `--auto-register` flag:**
- The Gitea CLI `gitea admin auth add-oauth` command does NOT have a `--auto-register` flag
- Auto-registration is controlled by the `app.ini` configuration: `[oauth2] ENABLE_AUTO_REGISTRATION = true`
- Previous versions of this documentation incorrectly mentioned the `--auto-register` CLI flag

## Solution

**Three fixes have been implemented:**

### 1. Fix Gitea Helm Configuration (Permanent)

The `internal/demo/components.go` has been updated with the correct configuration:

```go
"service": map[string]interface{}{
    "DISABLE_REGISTRATION":             false, // ← Changed from true to false
    "ALLOW_ONLY_EXTERNAL_REGISTRATION": true,
    "SHOW_REGISTRATION_BUTTON":         false, // ← Hide normal registration button
},
```

This ensures all future `demo-time` installations have the correct settings.

### 2. Fix OAuth2 Source (Immediate - for existing installations)

For existing demo environments that already have Gitea installed:

```bash
./innominatus-ctl fix-gitea-oauth
```

This command:
1. Finds the Gitea pod in Kubernetes
2. Lists existing OAuth2 sources
3. Removes the existing Keycloak OAuth2 source (if any)
4. Re-creates the OAuth2 source with the correct configuration:
   - `--provider openidConnect` - Use OpenID Connect protocol
   - `--skip-local-2fa` - Skip two-factor auth for OAuth users
   - `--scopes openid email profile` - Request necessary user information
   - Auto-registration is enabled via `app.ini` configuration, not CLI flag

### 3. Complete Reinstall (Recommended for existing installations)

To apply both fixes to an existing installation:

```bash
# Uninstall current demo environment
./innominatus-ctl demo-nuke

# Reinstall with corrected configuration
./innominatus-ctl demo-time
```

This will apply both the correct Helm configuration and the proper OAuth2 source settings.

---

**For developers:** The `internal/demo/installer.go` has been updated to correctly configure the OAuth2 source during `demo-time` installation.

**Updated code (line 1097-1109):**
```go
// Use Gitea CLI to add OAuth2 authentication source
// Note: Auto-registration is controlled by app.ini [oauth2] ENABLE_AUTO_REGISTRATION = true, not by CLI flag
addAuthCmd := exec.Command("kubectl", "--context", i.kubeContext,
    "exec", "-n", "gitea", podName, "--",
    "gitea", "admin", "auth", "add-oauth",
    "--name", "Keycloak",
    "--provider", "openidConnect",
    "--key", "gitea",
    "--secret", "gitea-client-secret",
    "--auto-discover-url", "http://keycloak.localtest.me/realms/demo-realm/.well-known/openid-configuration",
    "--skip-local-2fa",
    "--scopes", "openid", "email", "profile")
```

## How Auto-Registration Works

When Gitea is configured with `[oauth2] ENABLE_AUTO_REGISTRATION = true` in `app.ini`:

1. User clicks "Sign In with OAuth" → "Keycloak" in Gitea
2. User is redirected to Keycloak for authentication
3. User logs in with Keycloak credentials (e.g., demo-user/password123)
4. Keycloak redirects back to Gitea with user information
5. Gitea **automatically creates** a new user account with:
   - Username from Keycloak
   - Email from Keycloak
   - Profile information from Keycloak
6. User is logged into Gitea with their new account

**Important:** Auto-registration is controlled by the `app.ini` configuration, NOT by the CLI command flags.

## Testing the Fix

### Automated Verification Test

A comprehensive automated verification test is available to check the OAuth2 setup:

```bash
node verification/test-gitea-keycloak-oauth.mjs
```

This test verifies:
- ✅ Gitea service is running
- ✅ Keycloak service is running
- ✅ Kubernetes pods are healthy
- ✅ Gitea OAuth2 authentication source is configured
- ✅ Gitea `app.ini` has correct settings (`DISABLE_REGISTRATION = false`, `ENABLE_AUTO_REGISTRATION = true`)
- ✅ Keycloak OIDC client for Gitea exists with correct redirect URIs

**Run this test after:**
- `./innominatus-ctl demo-time` (initial installation)
- `./innominatus-ctl fix-gitea-oauth` (fixing existing installation)

### Prerequisites
- Demo environment running (`./innominatus-ctl demo-status`)
- Keycloak accessible at http://keycloak.localtest.me
- Gitea accessible at http://gitea.localtest.me
- Node.js installed (for automated verification test)

### Manual OAuth Login Test

After the automated verification passes, test the actual OAuth login flow:

1. **Ensure OAuth2 source is configured:**
   ```bash
   ./innominatus-ctl fix-gitea-oauth
   ```

2. **Navigate to Gitea:**
   - Open http://gitea.localtest.me in your browser
   - Click **"Sign In"**

3. **Use OAuth2 Login:**
   - Click **"Sign in with OAuth"**
   - Select **"Keycloak"**

4. **Login with Keycloak:**
   - Username: `demo-user`
   - Password: `password123`
   - (or use `test-user` / `test123`)

5. **Verify Success:**
   - You should be automatically logged into Gitea
   - Check your Gitea profile to confirm account was created
   - Username should match your Keycloak username

### Troubleshooting

**If auto-registration still doesn't work:**

1. **Check OAuth2 source configuration:**
   ```bash
   kubectl exec -n gitea $(kubectl get pods -n gitea -l app.kubernetes.io/name=gitea -o jsonpath='{.items[0].metadata.name}') -- gitea admin auth list
   ```

   You should see "Keycloak" in the list.

2. **Check Gitea logs:**
   ```bash
   kubectl logs -n gitea $(kubectl get pods -n gitea -l app.kubernetes.io/name=gitea -o jsonpath='{.items[0].metadata.name}') -f
   ```

3. **Verify Keycloak client exists:**
   - Go to http://keycloak.localtest.me/admin
   - Login: admin / adminpassword
   - Select "demo-realm"
   - Go to "Clients"
   - Find "gitea" client
   - Check redirect URIs include `http://gitea.localtest.me/user/oauth2/Keycloak/callback`

4. **Check Gitea app.ini configuration:**
   ```bash
   kubectl exec -n gitea $(kubectl get pods -n gitea -l app.kubernetes.io/name=gitea -o jsonpath='{.items[0].metadata.name}') -- cat /data/gitea/conf/app.ini | grep -A 5 "oauth2"
   ```

   Should show:
   ```ini
   [oauth2]
   ENABLE = true
   ENABLE_AUTO_REGISTRATION = true
   ```

5. **Re-run demo-time to reset:**
   ```bash
   ./innominatus-ctl demo-nuke
   ./innominatus-ctl demo-time
   ```

## Files Modified

| File | Change | Purpose |
|------|--------|---------|
| `internal/demo/components.go:83-85` | Changed `DISABLE_REGISTRATION` from `true` to `false`; Added `SHOW_REGISTRATION_BUTTON: false` | **Critical fix**: Allow external registration to work |
| `internal/demo/installer.go:1097-1109` | Added `--auto-register`, `--skip-local-2fa`, `--scopes` flags | Fix OAuth2 source for auto-registration |
| `internal/cli/commands.go:1010-1093` | Added `FixGiteaOAuthCommand()` | New CLI command to fix existing installations |
| `cmd/cli/main.go:36` | Added `fix-gitea-oauth` to `localCommands` | Register command as local (no auth required) |
| `cmd/cli/main.go:221-222` | Added case for `fix-gitea-oauth` | Handle command execution |
| `cmd/cli/main.go:389` | Added command to usage | Display in `--help` output |
| `cmd/cli/main.go:412` | Added example | Show usage example |
| `scripts/fix-gitea-oauth.sh` | New shell script | Alternative fix method (standalone script) |

## Related Configuration

The Gitea Helm values now have the **corrected** configuration (see `internal/demo/components.go:82-89`):

```go
"service": map[string]interface{}{
    "DISABLE_REGISTRATION":             false, // ← MUST be false!
    "ALLOW_ONLY_EXTERNAL_REGISTRATION": true,  // Only allow OAuth2
    "SHOW_REGISTRATION_BUTTON":         false, // Hide normal registration button
},
"oauth2": map[string]interface{}{
    "ENABLE":                   true,  // Enable OAuth2
    "ENABLE_AUTO_REGISTRATION": true,  // Enable auto-registration
},
```

**Key Points:**
- `DISABLE_REGISTRATION` **must be `false`** for `ALLOW_ONLY_EXTERNAL_REGISTRATION` to work
- `ALLOW_ONLY_EXTERNAL_REGISTRATION: true` blocks the normal registration form
- `SHOW_REGISTRATION_BUTTON: false` hides the "Register" button in the UI
- **Auto-registration** is controlled by `[oauth2] ENABLE_AUTO_REGISTRATION = true` in `app.ini`, NOT by CLI flags

## Summary

- **Problem:**
  - Gitea configuration had `DISABLE_REGISTRATION: true` (blocks ALL registration including OAuth2)

- **Fixes:**
  1. **Configuration Fix:** Changed `DISABLE_REGISTRATION` to `false` in `components.go`
  2. **OAuth2 Fix:** Corrected `installer.go` to properly configure OAuth2 source (removed non-existent `--auto-register` CLI flag)
  3. **CLI Tool:** Added `fix-gitea-oauth` command for existing installations
  4. **Verification Test:** Created automated test script `verification/test-gitea-keycloak-oauth.mjs`

- **Testing:**
  - **Automated:** Run `node verification/test-gitea-keycloak-oauth.mjs`
  - **Manual:** Follow OAuth login steps in browser

- **Recommended Action:** Run `demo-nuke` then `demo-time` to apply all fixes
- **Alternative:** Run `./innominatus-ctl fix-gitea-oauth` (configures OAuth2 source only)
- **Result:** Users can now log in via Keycloak and accounts are automatically created

## References

- Gitea OAuth2 Configuration: https://docs.gitea.com/usage/oauth2-provider
- Gitea CLI Admin Commands: https://docs.gitea.com/usage/command-line#admin
- Keycloak OIDC: https://www.keycloak.org/docs/latest/server_admin/#_oidc

---

**Last Updated:** 2025-10-16
**Tested On:** innominatus demo environment (Docker Desktop Kubernetes)
