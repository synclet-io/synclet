package pipelineconnect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	optionsv1 "github.com/synclet-io/synclet/gen/proto/synclet/options/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
)

func requiredRole(t *testing.T, file protoreflect.FileDescriptor, service, method protoreflect.Name) optionsv1.RequiredRole {
	t.Helper()

	svc := file.Services().ByName(service)
	require.NotNil(t, svc, "service %s", service)

	m := svc.Methods().ByName(method)
	require.NotNil(t, m, "method %s.%s", service, method)

	return proto.GetExtension(m.Options(), optionsv1.E_RequiredRole).(optionsv1.RequiredRole)
}

// TestStarRPCsRequireTheRoleThatCreatesSources asserts that whoever may create
// sources and destinations may star and unstar a connector.
func TestStarRPCsRequireTheRoleThatCreatesSources(t *testing.T) {
	pipelineFile := pipelinev1.File_synclet_publicapi_pipeline_v1_pipeline_proto
	createSource := requiredRole(t, pipelineFile, "SourceService", "CreateSource")
	require.Equal(t, createSource, requiredRole(t, pipelineFile, "DestinationService", "CreateDestination"))

	registryFile := registryv1.File_synclet_publicapi_registry_v1_registry_proto
	for _, method := range []protoreflect.Name{"StarManagedConnector", "UnstarManagedConnector"} {
		assert.Equal(t, createSource, requiredRole(t, registryFile, "ConnectorRegistryService", method), method)
	}
}
