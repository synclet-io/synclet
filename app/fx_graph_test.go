package app

import (
	"testing"

	"go.uber.org/fx"
)

// TestFxGraph validates the dependency graph for every deployment shape
// without starting the application, so a missing or mistyped provider fails
// here rather than at process start.
func TestFxGraph(t *testing.T) {
	cases := map[string]*RunAppOptions{
		"server": {
			RunPublicHTTPServer:   true,
			RunInternalHTTPServer: true,
		},
		"server standalone docker": {
			RunPublicHTTPServer:   true,
			RunInternalHTTPServer: true,
			RunJobs:               true,
			RunConsumers:          true,
			Standalone:            true,
			DockerExecutor:        true,
		},
		"server standalone k8s": {
			RunPublicHTTPServer:   true,
			RunInternalHTTPServer: true,
			RunJobs:               true,
			RunConsumers:          true,
			Standalone:            true,
			K8sExecutor:           true,
		},
		"jobs": {
			RunJobs:      true,
			RunConsumers: true,
		},
		"docker executor distributed": {
			DockerExecutor: true,
		},
		"k8s executor distributed": {
			K8sExecutor: true,
		},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			if err := fx.ValidateApp(NewFxAppOptions(options)); err != nil {
				t.Fatalf("fx graph invalid: %v", err)
			}
		})
	}
}
