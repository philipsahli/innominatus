# Login Command Implementation

## Summary

Successfully implemented `./innominatus-ctl login` and `./innominatus-ctl logout` commands that authenticate users, generate API keys, and store them locally for seamless CLI usage.

## Implementation Details

### Files Created/Modified

1. **NEW: `internal/cli/credentials.go`**
   - Credentials file management
   - Stores credentials in `$HOME/.idp-o/credentials` (JSON format)
   - Secure file permissions (0600 - owner read/write only)
   - Functions:
     - `SaveCredentials()` - Save API key to file
     - `LoadCredentials()` - Load API key from file (with expiry validation)
     - `ClearCredentials()` - Remove credentials file
     - `GetCredentialsPath()` - Return path to credentials file
     - `HasValidCredentials()` - Check for valid stored credentials

2. **MODIFIED: `internal/cli/client.go`**
   - Updated `NewClient()` to load API keys from credentials file
   - Priority order:
     1. `IDP_API_KEY` environment variable (highest priority - for CI/CD)
     2. `$HOME/.idp-o/credentials` file
     3. No API key (prompts for login)
   - Automatic expiry handling

3. **MODIFIED: `internal/cli/commands.go`**
   - `LoginCommand()` - Authenticates user and stores API key
     - Prompts for username/password
     - Authenticates with server
     - Generates API key via `/api/profile/api-keys` endpoint
     - Stores credentials locally
     - Default key name: `cli-<hostname>-<timestamp>`
     - Default expiry: 90 days
   - `LogoutCommand()` - Removes stored credentials
     - Deletes `$HOME/.idp-o/credentials` file
     - Shows confirmation message

4. **MODIFIED: `cmd/cli/main.go`**
   - Added `login` and `logout` to local commands (no auth required)
   - Added command cases for login/logout
   - Updated help text with examples

### Credentials File Format

```json
{
  "server_url": "http://localhost:8081",
  "username": "alice",
  "api_key": "1a2b3c4d5e6f...",
  "created_at": "2025-10-06T12:00:00Z",
  "expires_at": "2026-01-04T12:00:00Z",
  "key_name": "cli-myhost-1633521600"
}
```

### Usage Examples

#### Login
```bash
# Basic login (default 90-day expiry)
./innominatus-ctl login

# Custom API key name
./innominatus-ctl login --name my-laptop

# Custom expiry period
./innominatus-ctl login --expiry-days 30

# Combined options
./innominatus-ctl login --name production-cli --expiry-days 7
```

#### Logout
```bash
# Remove stored credentials
./innominatus-ctl logout
```

#### Automatic Authentication
```bash
# After login, all commands work without prompts
./innominatus-ctl list
./innominatus-ctl status my-app
./innominatus-ctl environments
```

### Authentication Flow

1. User runs `./innominatus-ctl login`
2. CLI prompts for username/password (interactive)
3. CLI authenticates with server (`POST /api/login`)
4. CLI calls `POST /api/profile/api-keys` with session token to generate API key
5. Server generates API key and returns it (full key only shown once)
6. CLI saves credentials to `$HOME/.idp-o/credentials` with secure permissions (0600)
7. Future CLI commands automatically load API key from file
8. If API key expires, user is notified and file is auto-removed

### Security Considerations

- **File Permissions**: Credentials file created with 0600 (owner read/write only)
- **Environment Variable Priority**: `IDP_API_KEY` env var takes precedence over file
- **Expiry Validation**: Expired keys automatically removed on load
- **API Key Storage**: Stored in plaintext locally (standard practice for CLI tools like AWS CLI, gcloud, etc.)
- **Secure Transmission**: API key generated server-side and transmitted over HTTPS in production

### Priority Order

When running a command, the CLI checks for API keys in this order:

1. **Environment Variable** (`IDP_API_KEY`) - Highest priority
   - Useful for CI/CD pipelines
   - `export IDP_API_KEY=your-key-here`

2. **Credentials File** (`$HOME/.idp-o/credentials`)
   - Created by `login` command
   - Validated for expiry on load

3. **No API Key** - Falls back to session authentication
   - Prompts for username/password
   - Session-based authentication for that command only

### Benefits

✅ **Seamless Experience**: Login once, use forever (until expiry)
✅ **Secure Storage**: File permissions restrict access to owner
✅ **Auto-Expiry Handling**: Expired keys auto-removed with clear messages
✅ **CI/CD Friendly**: Environment variable takes precedence
✅ **Multiple Key Support**: Different keys for different machines
✅ **Easy Logout**: Simple command to remove credentials

### Help Text

The CLI help now includes:

```
Commands:
  ...
  login [options]       Authenticate and store API key locally
  logout                Remove stored credentials
  ...

Examples:
  ./innominatus-ctl login
  ./innominatus-ctl login --name my-laptop --expiry-days 30
  ./innominatus-ctl logout
```

## Testing

1. **Build CLI**:
   ```bash
   go build -o innominatus-ctl cmd/cli/main.go
   ```

2. **Start Server**:
   ```bash
   ./innominatus
   ```

3. **Login**:
   ```bash
   ./innominatus-ctl login
   # Enter username: alice
   # Enter password: alice123
   ```

4. **Verify Credentials**:
   ```bash
   cat ~/.idp-o/credentials
   ```

5. **Test Auto-Auth**:
   ```bash
   ./innominatus-ctl list
   # Should work without prompting for login
   ```

6. **Logout**:
   ```bash
   ./innominatus-ctl logout
   ```

7. **Verify Cleanup**:
   ```bash
   ls ~/.idp-o/
   # credentials file should be removed
   ```

## Future Enhancements

Potential improvements for future iterations:

- [ ] Support multiple profiles (like AWS CLI)
- [ ] Interactive key renewal before expiry
- [ ] Optional API key revocation on logout
- [ ] File locking for concurrent CLI usage
- [ ] Keychain integration (macOS/Linux)
- [ ] Config file for default server URL

---

*Implemented: 2025-10-06*
