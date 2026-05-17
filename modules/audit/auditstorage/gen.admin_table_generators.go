package auditstorage

import (
	url "net/url"
	strings "strings"

	context "github.com/GoAdminGroup/go-admin/context"
	db "github.com/GoAdminGroup/go-admin/modules/db"
	form1 "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	table "github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	types "github.com/GoAdminGroup/go-admin/template/types"
	form "github.com/GoAdminGroup/go-admin/template/types/form"
	// user code 'imports'
	// end user code 'imports'
)

func NewTableGenerators() table.GeneratorList {
	return map[string]table.Generator{
		"audit_event": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("audit\".\"audit_events").SetTitle("AuditEvent").SetDescription("AuditEvent")
			formList.SetTable("audit\".\"audit_events").SetTitle("AuditEvent").SetDescription("AuditEvent")
			info.AddField("ID", "id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ID", "id", db.UUID, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("WorkspaceID", "workspace_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("WorkspaceID", "workspace_id", db.UUID, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("ActorType", "actor_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "user", Text: "User"},
					{Value: "api_key", Text: "APIKey"},
					{Value: "system", Text: "System"},
				},
			},
			)
			formList.AddField("ActorType", "actor_type", db.Enum, form.SelectSingle)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}

			formList.FieldOptions(types.FieldOptions{
				{Value: "user", Text: "User"},
				{Value: "api_key", Text: "APIKey"},
				{Value: "system", Text: "System"},
			})
			info.AddField("ActorID", "actor_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ActorID", "actor_id", db.UUID, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("ActorLabel", "actor_label", db.Text)
			info.FieldSortable()
			formList.AddField("ActorLabel", "actor_label", db.Text, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("Action", "action", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "create", Text: "Create"},
					{Value: "update", Text: "Update"},
					{Value: "delete", Text: "Delete"},
					{Value: "enable", Text: "Enable"},
					{Value: "disable", Text: "Disable"},
					{Value: "test", Text: "Test"},
					{Value: "sync", Text: "Sync"},
					{Value: "login", Text: "Login"},
					{Value: "logout", Text: "Logout"},
				},
			},
			)
			formList.AddField("Action", "action", db.Enum, form.SelectSingle)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}

			formList.FieldOptions(types.FieldOptions{
				{Value: "create", Text: "Create"},
				{Value: "update", Text: "Update"},
				{Value: "delete", Text: "Delete"},
				{Value: "enable", Text: "Enable"},
				{Value: "disable", Text: "Disable"},
				{Value: "test", Text: "Test"},
				{Value: "sync", Text: "Sync"},
				{Value: "login", Text: "Login"},
				{Value: "logout", Text: "Logout"},
			})
			info.AddField("ResourceType", "resource_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "source", Text: "Source"},
					{Value: "destination", Text: "Destination"},
					{Value: "connection", Text: "Connection"},
					{Value: "connector", Text: "Connector"},
					{Value: "repository", Text: "Repository"},
					{Value: "notification_channel", Text: "NotificationChannel"},
					{Value: "notification_rule", Text: "NotificationRule"},
					{Value: "webhook", Text: "Webhook"},
					{Value: "workspace", Text: "Workspace"},
					{Value: "workspace_member", Text: "WorkspaceMember"},
					{Value: "workspace_invite", Text: "WorkspaceInvite"},
					{Value: "api_key", Text: "APIKey"},
					{Value: "user", Text: "User"},
				},
			},
			)
			formList.AddField("ResourceType", "resource_type", db.Enum, form.SelectSingle)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}

			formList.FieldOptions(types.FieldOptions{
				{Value: "source", Text: "Source"},
				{Value: "destination", Text: "Destination"},
				{Value: "connection", Text: "Connection"},
				{Value: "connector", Text: "Connector"},
				{Value: "repository", Text: "Repository"},
				{Value: "notification_channel", Text: "NotificationChannel"},
				{Value: "notification_rule", Text: "NotificationRule"},
				{Value: "webhook", Text: "Webhook"},
				{Value: "workspace", Text: "Workspace"},
				{Value: "workspace_member", Text: "WorkspaceMember"},
				{Value: "workspace_invite", Text: "WorkspaceInvite"},
				{Value: "api_key", Text: "APIKey"},
				{Value: "user", Text: "User"},
			})
			info.AddField("ResourceID", "resource_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ResourceID", "resource_id", db.UUID, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("ResourceLabel", "resource_label", db.Text)
			info.FieldSortable()
			formList.AddField("ResourceLabel", "resource_label", db.Text, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("DiffJSON", "diff_json", db.Text)
			info.FieldSortable()
			formList.AddField("DiffJSON", "diff_json", db.Text, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("DiffTruncated", "diff_truncated", db.Bool)
			info.FieldSortable()
			formList.AddField("DiffTruncated", "diff_truncated", db.Bool, form.Text)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}
			info.AddField("CreatedAt", "created_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
			formList.AddField("CreatedAt", "created_at", db.Timestamp, form.Datetime)
			formList.PreProcessFn = func(values form1.Values) form1.Values {
				for k, v := range values {
					for i, v := range v {
						if strings.Contains(v, "%") {
							if newV, err := url.QueryUnescape(v); err == nil {
								values[k][i] = newV
							}
						}
					}
				}
				return values
			}

			return table
		},
	}
}
