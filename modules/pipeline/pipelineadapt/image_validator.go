package pipelineadapt

import (
	"context"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
)

// DBImageValidator validates connector images against repository connectors in the database.
type DBImageValidator struct {
	validateImage *pipelinerepositories.ValidateImage
}

// NewDBImageValidator creates a new DBImageValidator.
func NewDBImageValidator(validateImage *pipelinerepositories.ValidateImage) *DBImageValidator {
	return &DBImageValidator{validateImage: validateImage}
}

func (v *DBImageValidator) ValidateImage(ctx context.Context, dockerRepository string) error {
	return v.validateImage.Execute(ctx, pipelinerepositories.ValidateImageParams{
		DockerRepository: dockerRepository,
	})
}
