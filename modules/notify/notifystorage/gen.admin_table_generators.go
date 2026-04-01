package notifystorage

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
		"webhook": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("notify\".\"webhooks").SetTitle("Webhook").SetDescription("Webhook")
			formList.SetTable("notify\".\"webhooks").SetTitle("Webhook").SetDescription("Webhook")
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
			info.AddField("URL", "url", db.Text)
			info.FieldSortable()
			formList.AddField("URL", "url", db.Text, form.Text)
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
			info.AddField("Events", "events", db.Text)
			info.FieldSortable()
			formList.AddField("Events", "events", db.Text, form.Text)
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
			info.AddField("Secret", "secret", db.Text)
			info.FieldSortable()
			formList.AddField("Secret", "secret", db.Text, form.Text)
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
			info.AddField("Enabled", "enabled", db.Bool)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Enabled", "enabled", db.Bool, form.Text)
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
			info.AddField("UpdatedAt", "updated_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("UpdatedAt", "updated_at", db.Timestamp, form.Datetime)
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
		"notification_channel": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("notify\".\"notification_channels").SetTitle("NotificationChannel").SetDescription("NotificationChannel")
			formList.SetTable("notify\".\"notification_channels").SetTitle("NotificationChannel").SetDescription("NotificationChannel")
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
			info.AddField("Name", "name", db.Text)
			info.FieldSortable()
			formList.AddField("Name", "name", db.Text, form.Text)
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
			info.AddField("ChannelType", "channel_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "slack", Text: "Slack"},
					{Value: "email", Text: "Email"},
					{Value: "telegram", Text: "Telegram"},
				},
			},
			)
			formList.AddField("ChannelType", "channel_type", db.Enum, form.SelectSingle)
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
				{Value: "slack", Text: "Slack"},
				{Value: "email", Text: "Email"},
				{Value: "telegram", Text: "Telegram"},
			})
			info.AddField("Config", "config", db.Text)
			info.FieldSortable()
			formList.AddField("Config", "config", db.Text, form.Text)
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
			info.AddField("Enabled", "enabled", db.Bool)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Enabled", "enabled", db.Bool, form.Text)
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
			info.AddField("UpdatedAt", "updated_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("UpdatedAt", "updated_at", db.Timestamp, form.Datetime)
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
		"notification_rule": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("notify\".\"notification_rules").SetTitle("NotificationRule").SetDescription("NotificationRule")
			formList.SetTable("notify\".\"notification_rules").SetTitle("NotificationRule").SetDescription("NotificationRule")
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
			info.AddField("ChannelID", "channel_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ChannelID", "channel_id", db.UUID, form.Text)
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
			info.AddField("ConnectionID", "connection_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ConnectionID", "connection_id", db.UUID, form.Text)
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
			info.AddField("Condition", "condition", db.Enum)
			info.FieldSortable()
			formList.AddField("Condition", "condition", db.Enum, form.SelectSingle)
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
				{Value: "on_failure", Text: "OnFailure"},
				{Value: "on_consecutive_failures", Text: "OnConsecutiveFailures"},
				{Value: "on_zero_records", Text: "OnZeroRecords"},
			})
			info.AddField("ConditionValue", "condition_value", db.Int)
			info.FieldSortable()
			formList.AddField("ConditionValue", "condition_value", db.Int, form.Text)
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
			info.AddField("Enabled", "enabled", db.Bool)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Enabled", "enabled", db.Bool, form.Text)
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
			info.AddField("UpdatedAt", "updated_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("UpdatedAt", "updated_at", db.Timestamp, form.Datetime)
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
