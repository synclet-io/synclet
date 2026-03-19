package authstorage

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
		"user": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"users").SetTitle("User").SetDescription("User")
			formList.SetTable("auth\".\"users").SetTitle("User").SetDescription("User")
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
			info.AddField("Email", "email", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Email", "email", db.Text, form.Text)
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
			info.AddField("PasswordHash", "password_hash", db.Text)
			info.FieldSortable()
			formList.AddField("PasswordHash", "password_hash", db.Text, form.Text)
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

			return table
		},
		"refresh_token": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"refresh_tokens").SetTitle("RefreshToken").SetDescription("RefreshToken")
			formList.SetTable("auth\".\"refresh_tokens").SetTitle("RefreshToken").SetDescription("RefreshToken")
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
			info.AddField("UserID", "user_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("UserID", "user_id", db.UUID, form.Text)
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
			info.AddField("TokenHash", "token_hash", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("TokenHash", "token_hash", db.Text, form.Text)
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
			info.AddField("ExpiresAt", "expires_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("ExpiresAt", "expires_at", db.Timestamp, form.Datetime)
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

			return table
		},
		"api_key": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"api_keys").SetTitle("APIKey").SetDescription("APIKey")
			formList.SetTable("auth\".\"api_keys").SetTitle("APIKey").SetDescription("APIKey")
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
			info.AddField("UserID", "user_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("UserID", "user_id", db.UUID, form.Text)
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
			info.AddField("KeyHash", "key_hash", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("KeyHash", "key_hash", db.Text, form.Text)
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
			info.AddField("LastUsedAt", "last_used_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("LastUsedAt", "last_used_at", db.Timestamp, form.Datetime)
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
			info.AddField("ExpiresAt", "expires_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("ExpiresAt", "expires_at", db.Timestamp, form.Datetime)
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

			return table
		},
		"oidc_identity": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oidc_identities").SetTitle("OIDCIdentity").SetDescription("OIDCIdentity")
			formList.SetTable("auth\".\"oidc_identities").SetTitle("OIDCIdentity").SetDescription("OIDCIdentity")
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
			info.AddField("UserID", "user_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("UserID", "user_id", db.UUID, form.Text)
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
			info.AddField("ProviderSlug", "provider_slug", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ProviderSlug", "provider_slug", db.Text, form.Text)
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
			info.AddField("Subject", "subject", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Subject", "subject", db.Text, form.Text)
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
			info.AddField("Email", "email", db.Text)
			info.FieldSortable()
			formList.AddField("Email", "email", db.Text, form.Text)
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
			info.AddField("LastLoginAt", "last_login_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("LastLoginAt", "last_login_at", db.Timestamp, form.Datetime)
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
		"oidc_state": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oidc_states").SetTitle("OIDCState").SetDescription("OIDCState")
			formList.SetTable("auth\".\"oidc_states").SetTitle("OIDCState").SetDescription("OIDCState")
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
			info.AddField("State", "state", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("State", "state", db.Text, form.Text)
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
			info.AddField("Verifier", "verifier", db.Text)
			info.FieldSortable()
			formList.AddField("Verifier", "verifier", db.Text, form.Text)
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
			info.AddField("ProviderSlug", "provider_slug", db.Text)
			info.FieldSortable()
			formList.AddField("ProviderSlug", "provider_slug", db.Text, form.Text)
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
			info.AddField("ExpiresAt", "expires_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("ExpiresAt", "expires_at", db.Timestamp, form.Datetime)
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

			return table
		},
	}
}
