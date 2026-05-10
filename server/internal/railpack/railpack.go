package railpack

import (
	"context"
	"io"
	"os/exec"
)

func Build(ctx context.Context, projectDir string, imageName string, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}

	cmd := exec.CommandContext(ctx, "railpack", "build", projectDir, "--name", imageName)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}
