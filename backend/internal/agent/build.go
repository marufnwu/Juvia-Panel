package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BuildManager handles Docker image builds
type BuildManager struct {
	dataDir string
}

// NewBuildManager creates a new BuildManager
func NewBuildManager() *BuildManager {
	return &BuildManager{
		dataDir: "/var/panel",
	}
}

// SetDataDir sets the data directory for builds
func (bm *BuildManager) SetDataDir(dir string) {
	bm.dataDir = dir
}

// Build executes the build pipeline: clone → detect runtime → build image
func (bm *BuildManager) Build(ctx context.Context, params BuildParams) (*BuildResult, error) {
	startTime := time.Now()
	result := &BuildResult{
		BuildLogs: make([]LogLine, 0),
	}

	// Generate commit SHA from params or use timestamp
	commitSHA := params.Commit
	if commitSHA == "" {
		commitSHA = fmt.Sprintf("%d", time.Now().UnixNano())[:12]
	}
	result.CommitSHA = commitSHA

	// Generate image name
	sanitizedName := SanitizeAppName(params.AppName)
	result.ImageName = fmt.Sprintf("panel-app-%s:%s", sanitizedName, commitSHA)

	// Determine build directory
	buildDir := params.BuildPath
	if buildDir == "" {
		buildDir = GetBuildContextDir(bm.dataDir, params.AppID, commitSHA)
	}

	// Step 1: Clone repository
	bm.addLog(result, "info", "Starting build process...")

	if params.RepoURL != "" {
		bm.addLog(result, "info", fmt.Sprintf("Cloning repository from %s...", params.RepoURL))
		repoDir, err := bm.CloneRepo(ctx, params.AppID, params.RepoURL, params.Branch)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to clone repository: %v", err)
			bm.addLog(result, "error", result.Error)
			return result, nil
		}
		bm.addLog(result, "info", "Repository cloned successfully")
		buildDir = repoDir
	}

	// Step 2: Detect runtime
	bm.addLog(result, "info", "Detecting application runtime...")
	runtime, err := bm.DetectRuntime(buildDir)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to detect runtime: %v", err)
		bm.addLog(result, "error", result.Error)
		return result, nil
	}
	bm.addLog(result, "info", fmt.Sprintf("Detected runtime: %s", runtime))

	// Step 3: Build image
	bm.addLog(result, "info", "Building Docker image...")
	imageTag, err := bm.BuildImage(ctx, result, params, buildDir, runtime)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("build failed: %v", err)
		bm.addLog(result, "error", result.Error)
		return result, nil
	}

	result.ImageName = imageTag
	result.Success = true
	result.Duration = int(time.Since(startTime).Seconds())
	bm.addLog(result, "info", fmt.Sprintf("Build completed successfully in %d seconds", result.Duration))

	return result, nil
}

// CloneRepo clones a Git repository to the specified path
func (bm *BuildManager) CloneRepo(ctx context.Context, appID, repoURL, branch string) (string, error) {
	if branch == "" {
		branch = "main"
	}

	// Create temp directory for the repo
	repoDir := GetRepoDir(bm.dataDir, appID)

	// Clean up existing directory
	os.RemoveAll(repoDir)

	// Clone the repository
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", branch, repoURL, repoDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}

	return repoDir, nil
}

// DetectRuntime inspects the repository to determine the runtime
func (bm *BuildManager) DetectRuntime(repoPath string) (string, error) {
	// Check for Dockerfile first (explicit docker strategy)
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		return "docker", nil
	}

	// Check for package.json (Node.js)
	packageJSON := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJSON); err == nil {
		// Check if it's a Node.js project (has "scripts" or "dependencies")
		data, err := os.ReadFile(packageJSON)
		if err == nil {
			var pkg struct {
				Scripts      map[string]string `json:"scripts"`
				Dependencies map[string]string `json:"dependencies"`
			}
			if json.Unmarshal(data, &pkg); err == nil {
				if len(pkg.Scripts) > 0 || len(pkg.Dependencies) > 0 {
					return "nodejs", nil
				}
			}
		}
	}

	// Check for requirements.txt (Python)
	requirements := filepath.Join(repoPath, "requirements.txt")
	if _, err := os.Stat(requirements); err == nil {
		return "python", nil
	}

	// Check for go.mod (Go)
	goMod := filepath.Join(repoPath, "go.mod")
	if _, err := os.Stat(goMod); err == nil {
		return "go", nil
	}

	// Check for Gemfile (Ruby)
	gemfile := filepath.Join(repoPath, "Gemfile")
	if _, err := os.Stat(gemfile); err == nil {
		return "ruby", nil
	}

	// Check for composer.json (PHP)
	composer := filepath.Join(repoPath, "composer.json")
	if _, err := os.Stat(composer); err == nil {
		return "php", nil
	}

	// Check for static files (index.html)
	indexHTML := filepath.Join(repoPath, "index.html")
	if _, err := os.Stat(indexHTML); err == nil {
		return "static", nil
	}

	// Check for Cargo.toml (Rust)
	cargoToml := filepath.Join(repoPath, "Cargo.toml")
	if _, err := os.Stat(cargoToml); err == nil {
		return "rust", nil
	}

	// Default to static
	return "static", nil
}

// BuildImage builds a Docker image using the specified strategy
func (bm *BuildManager) BuildImage(ctx context.Context, result *BuildResult, params BuildParams, buildDir, runtime string) (string, error) {
	var imageTag string

	switch params.BuildStrategy {
	case "dockerfile":
		imageTag = result.ImageName
		if err := bm.buildWithDockerfile(ctx, result, buildDir, imageTag); err != nil {
			return "", err
		}
	case "static":
		imageTag = result.ImageName
		if err := bm.buildStatic(ctx, result, buildDir, imageTag, params.StartCommand); err != nil {
			return "", err
		}
	case "nixpacks", "":
		// Use nixpacks for auto-detection
		imageTag = result.ImageName
		if err := bm.buildWithNixpacks(ctx, result, buildDir, imageTag, params.BuildCommand, params.StartCommand); err != nil {
			return "", err
		}
	default:
		// Default to nixpacks
		imageTag = result.ImageName
		if err := bm.buildWithNixpacks(ctx, result, buildDir, imageTag, params.BuildCommand, params.StartCommand); err != nil {
			return "", err
		}
	}

	return imageTag, nil
}

// buildWithNixpacks builds an image using nixpacks
func (bm *BuildManager) buildWithNixpacks(ctx context.Context, result *BuildResult, buildDir, imageTag, buildCmd, startCmd string) error {
	bm.addLog(result, "info", "Building with Nixpacks...")

	// Create build context directory
	contextDir := filepath.Join(bm.dataDir, "tmp", "builds", result.CommitSHA)
	os.MkdirAll(contextDir, 0755)

	// Copy repo contents to build context
	if err := copyDirectory(buildDir, contextDir); err != nil {
		return fmt.Errorf("failed to copy build context: %w", err)
	}

	// Build arguments for nixpacks
	args := []string{"build", contextDir, "--name", imageTag}

	// Add custom build command if specified
	if buildCmd != "" {
		args = append(args, "--build-cmd", buildCmd)
	}

	// Add custom start command if specified
	if startCmd != "" {
		args = append(args, "--start-cmd", startCmd)
	}

	bm.addLog(result, "info", fmt.Sprintf("Running: nixpacks %v", strings.Join(args, " ")))

	// Run nixpacks
	cmd := exec.CommandContext(ctx, "nixpacks", args...)
	cmd.Dir = contextDir

	// Capture output in real-time
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nixpacks command failed: %w", err)
	}

	// Read output streams
	go readStream(result, stdout, "stdout")
	go readStream(result, stderr, "stderr")

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("nixpacks build failed: %w", err)
	}

	return nil
}

// buildWithDockerfile builds an image using an existing Dockerfile
func (bm *BuildManager) buildWithDockerfile(ctx context.Context, result *BuildResult, buildDir, imageTag string) error {
	bm.addLog(result, "info", "Building with Dockerfile...")

	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err != nil {
		return fmt.Errorf("Dockerfile not found in %s", buildDir)
	}

	// Run docker build
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, "-f", dockerfilePath, buildDir)
	cmd.Dir = buildDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.BuildLogs = append(result.BuildLogs, LogLine{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "error",
			Message:   string(output),
		})
		return fmt.Errorf("docker build failed: %w", err)
	}

	bm.addLog(result, "info", "Docker image built successfully")
	return nil
}

// buildStatic builds a simple static file image
func (bm *BuildManager) buildStatic(ctx context.Context, result *BuildResult, buildDir, imageTag, startCmd string) error {
	bm.addLog(result, "info", "Building static site...")

	// Use nginx for static serving by default
	if startCmd == "" {
		startCmd = "nginx -g 'daemon off;'"
	}

	// Create a simple Dockerfile for static files
	dockerfile := fmt.Sprintf(`FROM nginx:alpine
COPY . /usr/share/nginx/html
CMD %s`, startCmd)

	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to create Dockerfile: %w", err)
	}

	return bm.buildWithDockerfile(ctx, result, buildDir, imageTag)
}

// BuildImageDirect builds a Docker image from a Dockerfile at the specified path
func (bm *BuildManager) BuildImageDirect(ctx context.Context, appID, dockerfile string) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Generate image name
	commitSHA := fmt.Sprintf("%d", time.Now().UnixNano())[:12]
	sanitizedName := SanitizeAppName(appID)
	imageTag := fmt.Sprintf("panel-app-%s:%s", sanitizedName, commitSHA)

	// Determine build directory
	buildDir := filepath.Join(bm.dataDir, "tmp", "builds", appID)

	// Ensure build directory exists
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}

	// Write Dockerfile if provided inline, otherwise use existing one
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if dockerfile != "" {
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			return "", fmt.Errorf("failed to write Dockerfile: %w", err)
		}
	}

	// Verify Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("Dockerfile not found at %s", dockerfilePath)
	}

	// Run docker build
	bm.addBuildLog(appID, "info", fmt.Sprintf("Building Docker image: %s", imageTag))

	args := []string{"build", "-t", imageTag, "-f", dockerfilePath, buildDir}
	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		bm.addBuildLog(appID, "error", fmt.Sprintf("Docker build failed: %s", errMsg))
		return "", fmt.Errorf("docker build failed: %w - %s", err, errMsg)
	}

	bm.addBuildLog(appID, "info", "Docker image built successfully")
	return imageTag, nil
}

// buildImageFromPath builds an image using an existing Dockerfile at the given path
func (bm *BuildManager) buildImageFromPath(ctx context.Context, buildDir, imageTag string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s", dockerfilePath)
	}

	args := []string{"build", "-t", imageTag, "-f", dockerfilePath, buildDir}
	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w - %s", err, string(output))
	}

	return nil
}

// buildImageWithNixpacks builds an image using nixpacks for auto-detection
func (bm *BuildManager) buildImageWithNixpacks(ctx context.Context, buildDir, imageTag, buildCmd, startCmd string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Create build context directory
	contextDir := filepath.Join(bm.dataDir, "tmp", "builds", fmt.Sprintf("nixpacks-%d", time.Now().UnixNano()))
	os.MkdirAll(contextDir, 0755)

	// Copy repo contents to build context
	if err := copyDirectory(buildDir, contextDir); err != nil {
		return fmt.Errorf("failed to copy build context: %w", err)
	}

	// Build arguments for nixpacks
	args := []string{"build", contextDir, "--name", imageTag}

	// Add custom build command if specified
	if buildCmd != "" {
		args = append(args, "--build-cmd", buildCmd)
	}

	// Add custom start command if specified
	if startCmd != "" {
		args = append(args, "--start-cmd", startCmd)
	}

	cmd := exec.CommandContext(ctx, "nixpacks", args...)
	cmd.Dir = contextDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nixpacks build failed: %w - %s", err, string(output))
	}

	return nil
}

// buildImageWithStatic builds a static site image using nginx
func (bm *BuildManager) buildImageWithStatic(ctx context.Context, buildDir, imageTag, startCmd string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Use nginx for static serving by default
	if startCmd == "" {
		startCmd = "nginx -g 'daemon off;'"
	}

	// Create a Dockerfile for static files
	dockerfile := fmt.Sprintf(`FROM nginx:alpine
COPY . /usr/share/nginx/html
CMD %s`, startCmd)

	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to create Dockerfile: %w", err)
	}

	return bm.buildImageFromPath(ctx, buildDir, imageTag)
}

// buildLogs stores build logs per deployment
var buildLogs = make(map[string][]LogLine)
var logsMu sync.Mutex

// addBuildLog adds a log entry for a build
func (bm *BuildManager) addBuildLog(deploymentID, level, message string) {
	logsMu.Lock()
	defer logsMu.Unlock()

	logs := buildLogs[deploymentID]
	logs = append(logs, LogLine{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	})
	buildLogs[deploymentID] = logs
}

// GetBuildLogs retrieves stored build logs for a deployment
func (bm *BuildManager) GetBuildLogs(deploymentID string) []LogLine {
	logsMu.Lock()
	defer logsMu.Unlock()

	return buildLogs[deploymentID]
}

// CleanupBuildLogs removes stored logs for a deployment
func (bm *BuildManager) CleanupBuildLogs(deploymentID string) {
	logsMu.Lock()
	defer logsMu.Unlock()

	delete(buildLogs, deploymentID)
}

// addLog adds a log entry to the result
func (bm *BuildManager) addLog(result *BuildResult, level, message string) {
	result.BuildLogs = append(result.BuildLogs, LogLine{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	})
}

// readStream reads from a stream and adds to build logs
func readStream(result *BuildResult, r io.Reader, streamType string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		result.BuildLogs = append(result.BuildLogs, LogLine{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "info",
			Message:   scanner.Text(),
		})
	}
}

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.Stat()
	if err != nil {
		return err
	}

	_, err = srcFile.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = dstFile.Seek(0, 0)
	if err != nil {
		return err
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, werr := dstFile.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err != nil {
			break
		}
	}

	return dstFile.Sync()
}

// CleanupBuild removes build artifacts
func (bm *BuildManager) CleanupBuild(deploymentID string) error {
	buildDir := filepath.Join(bm.dataDir, "tmp", "builds", deploymentID)
	return os.RemoveAll(buildDir)
}