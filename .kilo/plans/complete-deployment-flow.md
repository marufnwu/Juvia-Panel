# Plan: Complete Deployment Flow Fix & Upload Source Support

## Current State

### Completed
- **CORS fix** — committed (`d34d0a4`), deployed, verified
- **Error persistence** — `handleDeploymentFailure` now persists error to DB via `UpdateDeploymentLogs` and logs to journal (uncommitted)
- **Deployment logging** — `executeDeployment` now logs build start, agent call errors, and build result failures (uncommitted)
- **nixpacks installed** on server at `/usr/local/bin/nixpacks` (v1.41.0)

### Remaining Issue
App `app_4R6wDvEJZ` has `source_config: {"type":"upload","provider":"other",...}` — no git repo URL. The build process tries to clone from empty URL, then nixpacks fails on an empty directory. The **upload source type is not yet supported** in the build pipeline or the frontend.

### Uncommitted Changes
- `backend/internal/handlers/apps/apps.go` — `handleDeploymentFailure` and `executeDeployment` improvements

---

## Phase 1: Fix Build Process for Upload-type Apps

### Problem
`executeDeployment()` always creates `BuildParams` with `RepoURL` from source config. For "upload" type, there's no repo URL. The `BuildManager.Build()` then:
1. Skips clone (since RepoURL is empty)
2. Uses `GetBuildContextDir()` as buildDir which is empty → nixpacks fails

The `BuildParams` struct already supports `BuildPath` to override the default build directory, and `BuildManager.Build()` respects it — we just need to set it.

### Changes

#### A. `apps.go:executeDeployment()` — lines 1860-1874

After parsing `sourceConfig`, check the source type:

```go
// Determine build path based on source type
var buildPath string
if sourceConfig.Type == "upload" {
    uploadDir := filepath.Join(h.config.DataDir, "apps", appID, "source")
    if info, err := os.Stat(uploadDir); err != nil || !info.IsDir() {
        h.handleDeploymentFailure(ctx, appID, deploymentID, "No source files uploaded. Please upload your project files before deploying.", "build")
        return
    }
    // Check if directory is empty
    entries, _ := os.ReadDir(uploadDir)
    if len(entries) == 0 {
        h.handleDeploymentFailure(ctx, appID, deploymentID, "No source files uploaded. Please upload your project files before deploying.", "build")
        return
    }
    buildPath = uploadDir
}
```

Then add `BuildPath` to `buildParams`:
```go
buildParams := agent.BuildParams{
    AppID:         appID,
    AppName:       app.Name,
    RepoURL:       sourceConfig.RepoURL,
    Branch:        derefOrEmpty(deployment.Branch),
    Commit:        "",
    BuildStrategy: app.BuildStrategy,
    BuildCommand:  buildConfig.BuildCommand,
    StartCommand:  buildConfig.StartCommand,
    BuildPath:     buildPath,  // <-- ADD THIS
}
```

#### B. Add imports to `apps.go`

Add `"os"` and `"path/filepath"` to imports (if not already present — check).

---

## Phase 2: Add File Upload Endpoint (Backend)

### Problem
There's no way to upload source files. The frontend has a UI placeholder ("Browse Files" button) but no handler.

### Changes

#### A. `apps.go` — Add `UploadSource` handler

```go
// UploadSource handles POST /apps/:id/upload
func (h *Handler) UploadSource(c *gin.Context) {
    requestID := c.GetString("request_id")
    appID := c.Param("id")
    ctx := context.Background()

    // Verify app exists
    app, _ := h.repo.GetApp(ctx, appID)
    if app == nil {
        c.JSON(http.StatusNotFound, ErrorResponse{...})
        return
    }

    // Parse multipart form (max 100MB)
    if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{...})
        return
    }

    file, header, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{...})
        return
    }
    defer file.Close()

    // Validate file type
    fileName := header.Filename
    ext := strings.ToLower(filepath.Ext(fileName))
    if !strings.HasSuffix(fileName, ".tar.gz") && ext != ".zip" && ext != ".tar" {
        c.JSON(http.StatusBadRequest, ErrorResponse{...})
        return
    }

    // Prepare upload directory
    uploadDir := filepath.Join(h.config.DataDir, "apps", appID, "source")
    os.RemoveAll(uploadDir)
    os.MkdirAll(uploadDir, 0755)

    // Save and extract
    tmpFile := filepath.Join(os.TempDir(), fileName)
    // ... save to tmp, extract using archive/tar or archive/zip ...

    c.JSON(http.StatusOK, gin.H{"status": "uploaded", "file": fileName})
}
```

#### B. `main.go` — Register route

```go
appsGroup.POST("/:id/upload", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
    c.Set("db", db)
    c.Set("config", cfg)
    appsHandler.UploadSource(c)
})
```

---

## Phase 3: Wire Up Frontend Upload UI

### Problem
The "Browse Files" button in `apps/new/page.tsx` (line 317) has no `onClick` handler. Upload is not functional.

### Changes

#### A. `frontend/src/lib/api.ts` — Add `uploadSource` method

```typescript
async uploadSource(appId: string, file: File): Promise<void> {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(`${this.baseUrl}/apps/${appId}/upload`, {
        method: 'POST',
        body: formData,
        headers: { Authorization: `Bearer ${this.token}` },
    })
    if (!res.ok) throw new Error('Upload failed')
}
```

#### B. `frontend/src/app/apps/new/page.tsx` — Wire upload button

Replace the static "Browse Files" text with a functional upload:
```tsx
<label className="mt-4 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-md text-sm cursor-pointer">
  Browse Files
  <input
    type="file"
    accept=".zip,.tar.gz,.tar"
    onChange={async (e) => {
      const file = e.target.files?.[0]
      if (!file) return
      setUploading(true)
      try {
        await api.uploadSource(appId, file)
        toast.success('Files uploaded')
      } catch (err) {
        toast.error('Upload failed')
      } finally {
        setUploading(false)
      }
    }}
    className="hidden"
  />
</label>
```

**Important:** The upload should happen AFTER the app is created (since we need the app ID). So the create flow should:
1. Create the app (POST /apps)
2. If source type is "upload", allow upload AFTER creation (app ID exists)
3. Trigger deployment (POST /apps/:id/deploy)

The frontend flow needs to be adjusted: after `CreateApp`, redirect to the app detail page where upload and deploy buttons exist.

---

## Phase 4: Add nixpacks to Install Script

### Problem
`scripts/install.sh` doesn't install nixpacks. The agent expects `nixpacks` binary in PATH for "auto" and "nixpacks" build strategies.

### Changes

#### `scripts/install.sh` — After "Building Go binaries" section (~line 268)

Add nixpacks installation:

```bash
# Install nixpacks for build support
if ! command -v nixpacks &> /dev/null; then
    log_info "Installing nixpacks (build tool)..."
    if ! command -v cargo &> /dev/null; then
        log_info "Installing Rust toolchain..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        source "$HOME/.cargo/env"
    fi
    "$HOME/.cargo/bin/cargo" install nixpacks
    cp "$HOME/.cargo/bin/nixpacks" /usr/local/bin/nixpacks
    chmod +x /usr/local/bin/nixpacks
    chown root:root /usr/local/bin/nixpacks
fi
```

---

## Phase 5: Commit and Deploy

### Steps
1. Commit `apps.go` changes (error persistence, logging, upload-type build path)
2. Commit upload handler + route registration
3. Commit frontend upload UI changes
4. Commit install script changes
5. Push all to GitHub
6. Build backend on server, restart `juvia-api`
7. Rebuild frontend, restart `juvia-ui`
8. Test: create upload-type app, upload files, deploy

---

## Files Changed

| File | Phase | Change |
|------|-------|--------|
| `backend/internal/handlers/apps/apps.go` | 1, 2 | Upload build path, UploadSource handler |
| `backend/cmd/api/main.go` | 2 | Register `/apps/:id/upload` route |
| `frontend/src/lib/api.ts` | 3 | Add `uploadSource()` method |
| `frontend/src/app/apps/new/page.tsx` | 3 | Wire up file input |
| `scripts/install.sh` | 4 | Install nixpacks |

## Testing

1. **Git source with valid repo** — should clone, detect runtime, build with nixpacks, start container
2. **Upload source with files** — should build from uploaded directory, start container
3. **Upload source without files** — should fail with clear error "No source files uploaded"
4. **nixpacks missing** — install script should install it automatically
5. **Deployment error visibility** — error message should appear in API response `build_logs` field
