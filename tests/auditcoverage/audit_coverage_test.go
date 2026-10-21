package auditcoverage_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// readOnlyPrefixes marks RPCs whose names promise no state change.
var readOnlyPrefixes = []string{
	"Get", "List", "Search", "Is", "Check", "Preview", "Export", "Download",
	"Stream", "Validate", "Wait", "Refresh", "Heartbeat",
}

// skippedHandlers are handler types that serve machine-to-machine protocols rather than user configuration.
var skippedHandlers = map[string]bool{
	"pipelineconnect.ExecutorHandler": true,
}

// audited maps each mutating RPC to the repo-relative use-case files that record its audit event.
var audited = map[string][]string{
	"auditconnect.Handler.UpdateAuditRetention": {"modules/audit/auditservice/update_audit_retention.go"},

	"authconnect.Handler.ChangePassword":       {"modules/auth/authservice/change_password.go"},
	"authconnect.Handler.CreateAPIKey":         {"modules/auth/authservice/create_api_key.go"},
	"authconnect.Handler.Login":                {"modules/auth/authservice/login.go"},
	"authconnect.Handler.Logout":               {"modules/auth/authservice/logout.go"},
	"authconnect.Handler.Register":             {"modules/auth/authservice/register_and_login.go"},
	"authconnect.Handler.RequestPasswordReset": {"modules/auth/authservice/request_password_reset.go"},
	"authconnect.Handler.ResetPassword":        {"modules/auth/authservice/reset_password.go"},
	"authconnect.Handler.RevokeAPIKey":         {"modules/auth/authservice/revoke_api_key.go"},
	"authconnect.Handler.UpdateProfile":        {"modules/auth/authservice/update_profile.go"},

	"notifyconnect.Handler.CreateWebhook":                         {"modules/notify/notifyservice/create_webhook.go"},
	"notifyconnect.Handler.DeleteWebhook":                         {"modules/notify/notifyservice/delete_webhook.go"},
	"notifyconnect.Handler.TestWebhook":                           {"modules/notify/notifyservice/test_webhook.go"},
	"notifyconnect.Handler.UpdateWebhook":                         {"modules/notify/notifyservice/update_webhook.go"},
	"notifyconnect.NotificationHandler.CreateNotificationChannel": {"modules/notify/notifyservice/create_channel.go"},
	"notifyconnect.NotificationHandler.CreateNotificationRule":    {"modules/notify/notifyservice/create_notification_rule.go"},
	"notifyconnect.NotificationHandler.DeleteNotificationChannel": {"modules/notify/notifyservice/delete_channel.go"},
	"notifyconnect.NotificationHandler.DeleteNotificationRule":    {"modules/notify/notifyservice/delete_notification_rule.go"},
	"notifyconnect.NotificationHandler.TestNotificationChannel":   {"modules/notify/notifyservice/test_channel.go"},
	"notifyconnect.NotificationHandler.UpdateNotificationChannel": {"modules/notify/notifyservice/update_channel.go"},
	"notifyconnect.NotificationHandler.UpdateNotificationRule":    {"modules/notify/notifyservice/update_notification_rule.go"},

	"pipelineconnect.ConnectionHandler.ConfigureStreams":       {"modules/pipeline/pipelineservice/pipelinecatalog/configure_streams.go"},
	"pipelineconnect.ConnectionHandler.CreateConnection":       {"modules/pipeline/pipelineservice/pipelineconnections/create_connection.go"},
	"pipelineconnect.ConnectionHandler.DeleteConnection":       {"modules/pipeline/pipelineservice/pipelineconnections/delete_connection.go"},
	"pipelineconnect.ConnectionHandler.DiscoverSchema":         {"modules/pipeline/pipelineservice/pipelinetasks/create_discover_task.go"},
	"pipelineconnect.ConnectionHandler.DisableConnection":      {"modules/pipeline/pipelineservice/pipelineconnections/disable_connection.go"},
	"pipelineconnect.ConnectionHandler.EnableConnection":       {"modules/pipeline/pipelineservice/pipelineconnections/enable_connection.go"},
	"pipelineconnect.ConnectionHandler.ImportWorkspaceConfig":  {"modules/pipeline/pipelineservice/pipelineconfig/import_config.go"},
	"pipelineconnect.ConnectionHandler.ResetConnectionState":   {"modules/pipeline/pipelineservice/pipelinestate/reset_connection_state.go"},
	"pipelineconnect.ConnectionHandler.ResetStreamState":       {"modules/pipeline/pipelineservice/pipelinestate/reset_stream_state.go"},
	"pipelineconnect.ConnectionHandler.UpdateConnection":       {"modules/pipeline/pipelineservice/pipelineconnections/update_connection.go"},
	"pipelineconnect.ConnectionHandler.UpdatePipelineSettings": {"modules/pipeline/pipelineservice/pipelinesettings/update_workspace_settings.go"},
	"pipelineconnect.ConnectionHandler.UpdateStreamState":      {"modules/pipeline/pipelineservice/pipelinestate/update_stream_state.go"},
	"pipelineconnect.DestinationHandler.CreateDestination":     {"modules/pipeline/pipelineservice/pipelinedestinations/create_destination.go"},
	"pipelineconnect.DestinationHandler.DeleteDestination":     {"modules/pipeline/pipelineservice/pipelinedestinations/delete_destination.go"},
	"pipelineconnect.DestinationHandler.TestDestinationConnection": {
		"modules/pipeline/pipelineservice/pipelinetasks/create_check_task.go",
	},
	"pipelineconnect.DestinationHandler.UpdateDestination":   {"modules/pipeline/pipelineservice/pipelinedestinations/update_destination.go"},
	"pipelineconnect.JobHandler.CancelJob":                   {"modules/pipeline/pipelineservice/pipelinejobs/cancel_job_for_workspace.go"},
	"pipelineconnect.JobHandler.TriggerSync":                 {"modules/pipeline/pipelineservice/pipelinejobs/trigger_sync.go"},
	"pipelineconnect.WorkflowHandler.CreateWorkflow":         {"modules/pipeline/pipelineservice/pipelineworkflows/create_workflow.go"},
	"pipelineconnect.WorkflowHandler.UpdateWorkflow":         {"modules/pipeline/pipelineservice/pipelineworkflows/update_workflow.go"},
	"pipelineconnect.WorkflowHandler.DeleteWorkflow":         {"modules/pipeline/pipelineservice/pipelineworkflows/delete_workflow.go"},
	"pipelineconnect.WorkflowHandler.RunWorkflow":            {"modules/pipeline/pipelineservice/pipelineworkflows/start_workflow_run.go"},
	"pipelineconnect.WorkflowHandler.RerunWorkflowRun":       {"modules/pipeline/pipelineservice/pipelineworkflows/rerun_workflow_run.go"},
	"pipelineconnect.RegistryHandler.AddConnector":           {"modules/pipeline/pipelineservice/pipelineconnectors/add_connector.go"},
	"pipelineconnect.RegistryHandler.AddRepository":          {"modules/pipeline/pipelineservice/pipelinerepositories/add_repository.go"},
	"pipelineconnect.RegistryHandler.BatchUpdateConnectors":  {"modules/pipeline/pipelineservice/pipelineconnectors/update_connector.go"},
	"pipelineconnect.RegistryHandler.DeleteManagedConnector": {"modules/pipeline/pipelineservice/pipelineconnectors/delete_connector.go"},
	"pipelineconnect.RegistryHandler.DeleteRepository":       {"modules/pipeline/pipelineservice/pipelinerepositories/delete_repository.go"},
	"pipelineconnect.RegistryHandler.StarManagedConnector":   {"modules/pipeline/pipelineservice/pipelineconnectors/set_connector_starred.go"},
	"pipelineconnect.RegistryHandler.SyncRepository":         {"modules/pipeline/pipelineservice/pipelinerepositories/sync_repository.go"},
	"pipelineconnect.RegistryHandler.UnstarManagedConnector": {"modules/pipeline/pipelineservice/pipelineconnectors/set_connector_starred.go"},
	"pipelineconnect.RegistryHandler.UpdateManagedConnector": {"modules/pipeline/pipelineservice/pipelineconnectors/update_connector.go"},
	"pipelineconnect.SourceHandler.CreateSource":             {"modules/pipeline/pipelineservice/pipelinesources/create_source.go"},
	"pipelineconnect.SourceHandler.DeleteSource":             {"modules/pipeline/pipelineservice/pipelinesources/delete_source.go"},
	"pipelineconnect.SourceHandler.DiscoverSourceSchema":     {"modules/pipeline/pipelineservice/pipelinetasks/create_discover_task.go"},
	"pipelineconnect.SourceHandler.TestSourceConnection":     {"modules/pipeline/pipelineservice/pipelinetasks/create_check_task.go"},
	"pipelineconnect.SourceHandler.UpdateSource":             {"modules/pipeline/pipelineservice/pipelinesources/update_source.go"},

	"workspaceconnect.Handler.AcceptInvite":     {"modules/workspace/workspaceservice/accept_invite.go"},
	"workspaceconnect.Handler.CreateInvite":     {"modules/workspace/workspaceservice/create_invite.go"},
	"workspaceconnect.Handler.CreateWorkspace":  {"modules/workspace/workspaceservice/create_workspace.go"},
	"workspaceconnect.Handler.DeclineInvite":    {"modules/workspace/workspaceservice/decline_invite.go"},
	"workspaceconnect.Handler.DeleteWorkspace":  {"modules/workspace/workspaceservice/delete_workspace.go"},
	"workspaceconnect.Handler.RemoveMember":     {"modules/workspace/workspaceservice/remove_member.go"},
	"workspaceconnect.Handler.ResendInvite":     {"modules/workspace/workspaceservice/resend_invite.go"},
	"workspaceconnect.Handler.RevokeInvite":     {"modules/workspace/workspaceservice/revoke_invite.go"},
	"workspaceconnect.Handler.UpdateMemberRole": {"modules/workspace/workspaceservice/update_member_role.go"},
	"workspaceconnect.Handler.UpdateWorkspace":  {"modules/workspace/workspaceservice/update_workspace.go"},
}

// exempt maps mutating RPCs that intentionally record no audit event to the reason why.
var exempt = map[string]string{
	"authconnect.Handler.AuthorizeOAuth": "issues a one-time code that expires in 10 minutes; the API key it is exchanged for is audited by ExchangeOAuthCode",
}

func TestMutatingRPCsAreAudited(t *testing.T) {
	root := repoRoot(t)
	rpcs := collectRPCs(t, root)

	var mutating []string

	for _, key := range rpcs {
		if !isReadOnly(key) {
			mutating = append(mutating, key)
		}
	}

	for _, key := range mutating {
		_, isAudited := audited[key]
		_, isExempt := exempt[key]

		switch {
		case isAudited && isExempt:
			t.Errorf("%s is listed in both audited and exempt; keep it in exactly one", key)
		case !isAudited && !isExempt:
			t.Errorf("%s is a mutating RPC without an audit mapping: call audit.Record in its use case and add the "+
				"use-case file to audited, or add a justified entry to exempt", key)
		}
	}

	for _, key := range sortedKeys(audited) {
		if !slices.Contains(mutating, key) {
			t.Errorf("audited lists %s, which is not a mutating RPC handler; remove or rename the entry", key)

			continue
		}

		if len(audited[key]) == 0 {
			t.Errorf("audited lists %s without any use-case file", key)
		}

		for _, rel := range audited[key] {
			assertRecordsAudit(t, root, key, rel)
		}
	}

	for _, key := range sortedKeys(exempt) {
		if !slices.Contains(mutating, key) {
			t.Errorf("exempt lists %s, which is not a mutating RPC handler; remove or rename the entry", key)
		}

		if strings.TrimSpace(exempt[key]) == "" {
			t.Errorf("exempt lists %s without a reason", key)
		}
	}
}

func assertRecordsAudit(t *testing.T, root, key, rel string) {
	t.Helper()

	src, err := fs.ReadFile(os.DirFS(root), rel)
	if err != nil {
		t.Errorf("%s maps to %s, which cannot be read: %v; fix the path in audited", key, rel, err)

		return
	}

	if !strings.Contains(string(src), "audit.Record(") {
		t.Errorf("%s maps to %s, which does not call audit.Record(; instrument the use case or fix the mapping", key, rel)
	}
}

// collectRPCs returns sorted "<package>.<HandlerType>.<Method>" keys for every exported handler method
// whose second parameter is *connect.Request[...].
func collectRPCs(t *testing.T, root string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(root, "modules", "*", "*connect", "*.go"))
	if err != nil {
		t.Fatalf("globbing handler files: %v", err)
	}

	fset := token.NewFileSet()

	var keys []string

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() || !takesConnectRequest(fn) {
				continue
			}

			recv := receiverTypeName(fn.Recv.List[0].Type)
			if recv == "" || skippedHandlers[file.Name.Name+"."+recv] {
				continue
			}

			keys = append(keys, file.Name.Name+"."+recv+"."+fn.Name.Name)
		}
	}

	if len(keys) == 0 {
		t.Fatalf("no RPC handlers found under %s/modules; the handler layout changed", root)
	}

	slices.Sort(keys)

	return keys
}

func takesConnectRequest(fn *ast.FuncDecl) bool {
	var params []ast.Expr

	for _, field := range fn.Type.Params.List {
		count := max(len(field.Names), 1)
		for range count {
			params = append(params, field.Type)
		}
	}

	if len(params) < 2 {
		return false
	}

	star, ok := params[1].(*ast.StarExpr)
	if !ok {
		return false
	}

	index, ok := star.X.(*ast.IndexExpr)
	if !ok {
		return false
	}

	sel, ok := index.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)

	return ok && pkg.Name == "connect" && sel.Sel.Name == "Request"
}

func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

func isReadOnly(key string) bool {
	method := key[strings.LastIndex(key, ".")+1:]

	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}

	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file location")
	}

	dir := filepath.Dir(file)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", file)
		}

		dir = parent
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}
