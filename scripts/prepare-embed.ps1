# prepare-embed.ps1 - Copies static files to cmd/server/ for Go embed directives

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$EmbedDir = Join-Path $ProjectRoot "cmd\server"

Write-Host "Preparing static files for embedding..." -ForegroundColor Green

# Create cmd/server directory if it doesn't exist
if (-not (Test-Path $EmbedDir)) {
    New-Item -ItemType Directory -Path $EmbedDir -Force | Out-Null
}

# Copy migrations
Write-Host "Copying migrations..." -ForegroundColor Yellow
$MigrationsSrc = Join-Path $ProjectRoot "migrations"
$MigrationsDst = Join-Path $EmbedDir "migrations"
if (Test-Path $MigrationsDst) {
    Remove-Item -Recurse -Force $MigrationsDst
}
Copy-Item -Recurse $MigrationsSrc $MigrationsDst

# Copy swagger files
Write-Host "Copying swagger files..." -ForegroundColor Yellow
$SwaggerAdmin = Join-Path $ProjectRoot "swagger-admin.yaml"
$SwaggerUser = Join-Path $ProjectRoot "swagger-user.yaml"
Copy-Item -Force $SwaggerAdmin (Join-Path $EmbedDir "swagger-admin.yaml")
Copy-Item -Force $SwaggerUser (Join-Path $EmbedDir "swagger-user.yaml")

# Copy web-ui build output
Write-Host "Copying web-ui output..." -ForegroundColor Yellow
$WebUISrc = Join-Path $ProjectRoot "web-ui\out"
$WebUIDst = Join-Path $EmbedDir "web-ui-out"
if (Test-Path $WebUIDst) {
    Remove-Item -Recurse -Force $WebUIDst
}
if (Test-Path $WebUISrc) {
    Copy-Item -Recurse $WebUISrc $WebUIDst
} else {
    Write-Host "Warning: web-ui/out not found, creating minimal placeholder" -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $WebUIDst -Force | Out-Null
    '<!DOCTYPE html><html><body><p>Web UI not built. Run: cd web-ui && npm run build</p></body></html>' |
        Out-File -FilePath (Join-Path $WebUIDst "index.html") -Encoding utf8
}

Write-Host "Static files prepared successfully!" -ForegroundColor Green
Write-Host "  - $MigrationsDst" -ForegroundColor Cyan
Write-Host "  - $(Join-Path $EmbedDir 'swagger-*.yaml')" -ForegroundColor Cyan
Write-Host "  - $WebUIDst" -ForegroundColor Cyan
