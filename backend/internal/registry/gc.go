package registry

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GCMode controls how garbage collection is executed.
type GCMode string

const (
	GCModeLocal  GCMode = "local"
	GCModeRemote GCMode = "remote" // not available for plain Docker Registry V2
)

type GCResult struct {
	Supported           bool   `json:"supported"`
	DryRun              bool   `json:"dry_run"`
	Message             string `json:"message"`
	ReclaimableBlobs    int    `json:"reclaimable_blobs,omitempty"`
	ReclaimableBytes    int64  `json:"reclaimable_bytes,omitempty"`
	ReclaimableHuman    string `json:"reclaimable_human,omitempty"`
	Output              string `json:"output,omitempty"`
}

// LocalGCRunner runs controlled registry garbage-collect via docker exec.
// NEVER accepts arbitrary user-supplied shell commands.
type LocalGCRunner struct {
	ContainerName string
}

var safeContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func (r *LocalGCRunner) Run(ctx context.Context, dryRun bool) (*GCResult, error) {
	if r.ContainerName == "" {
		return &GCResult{
			Supported: false,
			DryRun:    dryRun,
			Message:   "Local GC is not configured. Set REGISTRY_CONTAINER_NAME to the Docker registry container name, and mount the Docker socket if needed.",
		}, nil
	}
	if !safeContainerName.MatchString(r.ContainerName) {
		return nil, fmt.Errorf("invalid REGISTRY_CONTAINER_NAME")
	}

	args := []string{"exec", r.ContainerName, "registry", "garbage-collect", "/etc/docker/registry/config.yml"}
	if dryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String() + stderr.String()

	if err != nil {
		return &GCResult{
			Supported: true,
			DryRun:    dryRun,
			Message:   "Garbage collection failed. Ensure the registry container is running and Docker socket access is available.",
			Output:    truncate(out, 4000),
		}, fmt.Errorf("gc failed: %w", err)
	}

	blobs, bytesReclaim := parseGCOutput(out)
	return &GCResult{
		Supported:        true,
		DryRun:           dryRun,
		Message:          "Garbage collection completed",
		ReclaimableBlobs: blobs,
		ReclaimableBytes: bytesReclaim,
		ReclaimableHuman: formatBytes(bytesReclaim),
		Output:           truncate(out, 4000),
	}, nil
}

func parseGCOutput(out string) (blobs int, bytes int64) {
	// Best-effort parse of registry garbage-collect dry-run output
	reBlob := regexp.MustCompile(`(?i)blob\s+unreferenced|marking blob|delete blob`)
	for _, line := range strings.Split(out, "\n") {
		if reBlob.MatchString(line) {
			blobs++
		}
		if m := regexp.MustCompile(`(?i)(\d+)\s*bytes`).FindStringSubmatch(line); len(m) == 2 {
			if n, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				bytes += n
			}
		}
	}
	return blobs, bytes
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "unknown"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func RemoteGCUnsupported() *GCResult {
	return &GCResult{
		Supported: false,
		Message:   "Remote API garbage collection is not available for Docker Registry HTTP API V2. Configure Local Registry GC mode with REGISTRY_CONTAINER_NAME.",
	}
}
