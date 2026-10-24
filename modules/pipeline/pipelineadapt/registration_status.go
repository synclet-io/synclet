package pipelineadapt

import (
	"context"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
)

// RegistrationStatus reports through the auth module whether sign-up is open.
type RegistrationStatus struct {
	registrationOpen *authservice.RegistrationOpen
}

func NewRegistrationStatus(registrationOpen *authservice.RegistrationOpen) *RegistrationStatus {
	return &RegistrationStatus{registrationOpen: registrationOpen}
}

var _ pipelineconnect.RegistrationStatus = (*RegistrationStatus)(nil)

func (a *RegistrationStatus) RegistrationOpen(ctx context.Context) (bool, error) {
	return a.registrationOpen.Execute(ctx)
}
