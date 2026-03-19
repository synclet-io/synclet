package pipelineadapt

import (
	"context"

	"github.com/synclet-io/synclet/pkg/container"
)

// ImagePullerAdapter adapts container.Runner to pipelineservice.ImagePuller.
type ImagePullerAdapter struct {
	runner container.Runner
}

// NewImagePullerAdapter creates a new ImagePullerAdapter.
func NewImagePullerAdapter(runner container.Runner) *ImagePullerAdapter {
	return &ImagePullerAdapter{runner: runner}
}

func (a *ImagePullerAdapter) Pull(ctx context.Context, image string) error {
	return a.runner.Pull(ctx, image)
}
