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
		"password_reset_token": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"password_reset_tokens").SetTitle("PasswordResetToken").SetDescription("PasswordResetToken")
			formList.SetTable("auth\".\"password_reset_tokens").SetTitle("PasswordResetToken").SetDescription("PasswordResetToken")
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
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
			info.AddField("OAuthGrantID", "o_auth_grant_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("OAuthGrantID", "o_auth_grant_id", db.UUID, form.Text)
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
			info.AddField("OAuthLevel", "o_auth_level", db.Enum)
			info.FieldSortable()
			formList.AddField("OAuthLevel", "o_auth_level", db.Enum, form.SelectSingle)
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
				{Value: "read", Text: "Read"},
				{Value: "actions", Text: "Actions"},
				{Value: "write", Text: "Write"},
			})
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
		"o_auth_client": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oauth_clients").SetTitle("OAuthClient").SetDescription("OAuthClient")
			formList.SetTable("auth\".\"oauth_clients").SetTitle("OAuthClient").SetDescription("OAuthClient")
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
			info.AddField("ClientID", "client_id", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ClientID", "client_id", db.Text, form.Text)
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
			info.AddField("RedirectURIs", "redirect_ur_is", db.Text)
			info.FieldSortable()
			formList.AddField("RedirectURIs", "redirect_ur_is", db.Text, form.Code)
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
			info.AddField("LastGrantedAt", "last_granted_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("LastGrantedAt", "last_granted_at", db.Timestamp, form.Datetime)
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
		"o_auth_code": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oauth_codes").SetTitle("OAuthCode").SetDescription("OAuthCode")
			formList.SetTable("auth\".\"oauth_codes").SetTitle("OAuthCode").SetDescription("OAuthCode")
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
			info.AddField("CodeHash", "code_hash", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("CodeHash", "code_hash", db.Text, form.Text)
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
			info.AddField("ClientID", "client_id", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ClientID", "client_id", db.Text, form.Text)
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
			info.AddField("RedirectURI", "redirect_uri", db.Text)
			info.FieldSortable()
			formList.AddField("RedirectURI", "redirect_uri", db.Text, form.Text)
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
			info.AddField("CodeChallenge", "code_challenge", db.Text)
			info.FieldSortable()
			formList.AddField("CodeChallenge", "code_challenge", db.Text, form.Text)
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
			info.AddField("WorkspaceID", "workspace_id", db.UUID)
			info.FieldSortable()
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
			info.AddField("Level", "level", db.Enum)
			info.FieldSortable()
			formList.AddField("Level", "level", db.Enum, form.SelectSingle)
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
				{Value: "read", Text: "Read"},
				{Value: "actions", Text: "Actions"},
				{Value: "write", Text: "Write"},
			})
			info.AddField("ExpiresAt", "expires_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
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
		"o_auth_grant": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oauth_grants").SetTitle("OAuthGrant").SetDescription("OAuthGrant")
			formList.SetTable("auth\".\"oauth_grants").SetTitle("OAuthGrant").SetDescription("OAuthGrant")
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
			info.AddField("ClientID", "client_id", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ClientID", "client_id", db.Text, form.Text)
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
			info.AddField("Level", "level", db.Enum)
			info.FieldSortable()
			formList.AddField("Level", "level", db.Enum, form.SelectSingle)
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
				{Value: "read", Text: "Read"},
				{Value: "actions", Text: "Actions"},
				{Value: "write", Text: "Write"},
			})
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
		"o_auth_refresh_token": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("auth\".\"oauth_refresh_tokens").SetTitle("OAuthRefreshToken").SetDescription("OAuthRefreshToken")
			formList.SetTable("auth\".\"oauth_refresh_tokens").SetTitle("OAuthRefreshToken").SetDescription("OAuthRefreshToken")
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
			info.AddField("GrantID", "grant_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("GrantID", "grant_id", db.UUID, form.Text)
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
			info.AddField("UsedAt", "used_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("UsedAt", "used_at", db.Timestamp, form.Datetime)
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
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
