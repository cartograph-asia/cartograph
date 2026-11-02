package countryscan

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// dockerImage runs osmium where it is not installed, as map/build.sh does.
const dockerImage = "debian:bookworm-slim"

// Cut writes to out the boundary=administrative relations of the extract in,
// with their ways and nodes, and its place=* nodes: what the scan reads,
// a small part of a large extract. It does nothing when out exists and is
// newer than in. osmium runs on the host if installed, else (or with
// docker) in a throwaway container.
func Cut(ctx context.Context, in, out string, docker bool) error {
	if fi, err := os.Stat(in); err != nil {
		return fmt.Errorf("countryscan: %w", err)
	} else if oi, err := os.Stat(out); err == nil && oi.ModTime().After(fi.ModTime()) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("countryscan: %w", err)
	}
	tmp := out + ".part"
	defer os.Remove(tmp)
	args := []string{"tags-filter", "--overwrite", "-f", "pbf", "-o"}
	var cmd *exec.Cmd
	if bin, err := exec.LookPath("osmium"); err == nil && !docker {
		cmd = exec.CommandContext(ctx, bin, append(args, tmp, in, "r/boundary=administrative", "n/place")...)
	} else {
		if _, err := exec.LookPath("docker"); err != nil {
			return fmt.Errorf("countryscan: neither osmium nor docker is installed (brew install osmium-tool)")
		}
		inDir, outDir := filepath.Dir(in), filepath.Dir(out)
		abs := func(p string) string { a, _ := filepath.Abs(p); return a }
		// The file names are arguments of the script ($1, $2), never part of it.
		script := `apt-get update -qq && apt-get install -y -qq --no-install-recommends osmium-tool >/dev/null && ` +
			`osmium tags-filter --overwrite -f pbf -o "/out/$2" "/in/$1" r/boundary=administrative n/place`
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "-v", abs(inDir)+":/in:ro", "-v", abs(outDir)+":/out", dockerImage,
			"sh", "-c", script, "sh", filepath.Base(in), filepath.Base(tmp))
	}
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("countryscan: cut %s: %w", in, err)
	}
	if err := os.Rename(tmp, out); err != nil {
		return fmt.Errorf("countryscan: %w", err)
	}
	return nil
}
