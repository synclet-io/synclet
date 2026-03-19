package pipelinestorage

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
		"managed_connector": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"managed_connectors").SetTitle("ManagedConnector").SetDescription("ManagedConnector")
			formList.SetTable("pipeline\".\"managed_connectors").SetTitle("ManagedConnector").SetDescription("ManagedConnector")
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
			info.AddField("DockerImage", "docker_image", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("DockerImage", "docker_image", db.Text, form.Text)
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
			info.AddField("DockerTag", "docker_tag", db.Text)
			info.FieldSortable()
			formList.AddField("DockerTag", "docker_tag", db.Text, form.Text)
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
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
			info.AddField("ConnectorType", "connector_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "source", Text: "Source"},
					{Value: "destination", Text: "Destination"},
				},
			},
			)
			formList.AddField("ConnectorType", "connector_type", db.Enum, form.SelectSingle)
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
			})
			info.AddField("Spec", "spec", db.Text)
			info.FieldSortable()
			formList.AddField("Spec", "spec", db.Text, form.Text)
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
			info.AddField("RepositoryID", "repository_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("RepositoryID", "repository_id", db.UUID, form.Text)
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
		"repository": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"repositories").SetTitle("Repository").SetDescription("Repository")
			formList.SetTable("pipeline\".\"repositories").SetTitle("Repository").SetDescription("Repository")
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
			info.AddField("AuthHeader", "auth_header", db.Text)
			info.FieldSortable()
			formList.AddField("AuthHeader", "auth_header", db.Text, form.Text)
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
			info.AddField("Status", "status", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "syncing", Text: "Syncing"},
					{Value: "synced", Text: "Synced"},
					{Value: "failed", Text: "Failed"},
				},
			},
			)
			formList.AddField("Status", "status", db.Enum, form.SelectSingle)
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
				{Value: "syncing", Text: "Syncing"},
				{Value: "synced", Text: "Synced"},
				{Value: "failed", Text: "Failed"},
			})
			info.AddField("LastSyncedAt", "last_synced_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("LastSyncedAt", "last_synced_at", db.Timestamp, form.Datetime)
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
			info.AddField("ConnectorCount", "connector_count", db.Int)
			info.FieldSortable()
			formList.AddField("ConnectorCount", "connector_count", db.Int, form.Text)
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
			info.AddField("LastError", "last_error", db.Text)
			info.FieldSortable()
			formList.AddField("LastError", "last_error", db.Text, form.Text)
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
		"repository_connector": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"repository_connectors").SetTitle("RepositoryConnector").SetDescription("RepositoryConnector")
			formList.SetTable("pipeline\".\"repository_connectors").SetTitle("RepositoryConnector").SetDescription("RepositoryConnector")
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
			info.AddField("RepositoryID", "repository_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("RepositoryID", "repository_id", db.UUID, form.Text)
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
			info.AddField("DockerRepository", "docker_repository", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("DockerRepository", "docker_repository", db.Text, form.Text)
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
			info.AddField("DockerImageTag", "docker_image_tag", db.Text)
			info.FieldSortable()
			formList.AddField("DockerImageTag", "docker_image_tag", db.Text, form.Text)
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
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
			info.AddField("ConnectorType", "connector_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "source", Text: "Source"},
					{Value: "destination", Text: "Destination"},
				},
			},
			)
			formList.AddField("ConnectorType", "connector_type", db.Enum, form.SelectSingle)
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
			})
			info.AddField("DocumentationURL", "documentation_url", db.Text)
			info.FieldSortable()
			formList.AddField("DocumentationURL", "documentation_url", db.Text, form.Text)
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
			info.AddField("ReleaseStage", "release_stage", db.Enum)
			info.FieldSortable()
			formList.AddField("ReleaseStage", "release_stage", db.Enum, form.SelectSingle)
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
				{Value: "generally_available", Text: "GenerallyAvailable"},
				{Value: "beta", Text: "Beta"},
				{Value: "alpha", Text: "Alpha"},
				{Value: "custom", Text: "Custom"},
				{Value: "unknown", Text: "Unknown"},
			})
			info.AddField("IconURL", "icon_url", db.Text)
			info.FieldSortable()
			formList.AddField("IconURL", "icon_url", db.Text, form.Text)
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
			info.AddField("Spec", "spec", db.Text)
			info.FieldSortable()
			formList.AddField("Spec", "spec", db.Text, form.Text)
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
			info.AddField("SupportLevel", "support_level", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "community", Text: "Community"},
					{Value: "certified", Text: "Certified"},
					{Value: "unknown", Text: "Unknown"},
				},
			},
			)
			formList.AddField("SupportLevel", "support_level", db.Enum, form.SelectSingle)
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
				{Value: "community", Text: "Community"},
				{Value: "certified", Text: "Certified"},
				{Value: "unknown", Text: "Unknown"},
			})
			info.AddField("License", "license", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("License", "license", db.Text, form.Text)
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
			info.AddField("SourceType", "source_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "api", Text: "API"},
					{Value: "database", Text: "Database"},
					{Value: "file", Text: "File"},
					{Value: "unknown", Text: "Unknown"},
				},
			},
			)
			formList.AddField("SourceType", "source_type", db.Enum, form.SelectSingle)
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
				{Value: "api", Text: "API"},
				{Value: "database", Text: "Database"},
				{Value: "file", Text: "File"},
				{Value: "unknown", Text: "Unknown"},
			})
			info.AddField("Metadata", "metadata", db.Text)
			info.FieldSortable()
			formList.AddField("Metadata", "metadata", db.Text, form.Text)
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
		"source": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"sources").SetTitle("Source").SetDescription("Source")
			formList.SetTable("pipeline\".\"sources").SetTitle("Source").SetDescription("Source")
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
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
			info.AddField("ManagedConnectorID", "managed_connector_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ManagedConnectorID", "managed_connector_id", db.UUID, form.Text)
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
			info.AddField("RuntimeConfig", "runtime_config", db.Text)
			info.FieldSortable()
			formList.AddField("RuntimeConfig", "runtime_config", db.Text, form.Text)
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
		"destination": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"destinations").SetTitle("Destination").SetDescription("Destination")
			formList.SetTable("pipeline\".\"destinations").SetTitle("Destination").SetDescription("Destination")
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
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
			info.AddField("ManagedConnectorID", "managed_connector_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("ManagedConnectorID", "managed_connector_id", db.UUID, form.Text)
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
			info.AddField("RuntimeConfig", "runtime_config", db.Text)
			info.FieldSortable()
			formList.AddField("RuntimeConfig", "runtime_config", db.Text, form.Text)
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
		"connection": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"connections").SetTitle("Connection").SetDescription("Connection")
			formList.SetTable("pipeline\".\"connections").SetTitle("Connection").SetDescription("Connection")
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
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
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
			info.AddField("Status", "status", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "active", Text: "Active"},
					{Value: "inactive", Text: "Inactive"},
					{Value: "paused", Text: "Paused"},
				},
			},
			)
			formList.AddField("Status", "status", db.Enum, form.SelectSingle)
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
				{Value: "active", Text: "Active"},
				{Value: "inactive", Text: "Inactive"},
				{Value: "paused", Text: "Paused"},
			})
			info.AddField("SourceID", "source_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("SourceID", "source_id", db.UUID, form.Text)
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
			info.AddField("DestinationID", "destination_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("DestinationID", "destination_id", db.UUID, form.Text)
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
			info.AddField("Schedule", "schedule", db.Text)
			info.FieldSortable()
			formList.AddField("Schedule", "schedule", db.Text, form.Text)
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
			info.AddField("SchemaChangePolicy", "schema_change_policy", db.Enum)
			info.FieldSortable()
			formList.AddField("SchemaChangePolicy", "schema_change_policy", db.Enum, form.SelectSingle)
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
				{Value: "propagate", Text: "Propagate"},
				{Value: "ignore", Text: "Ignore"},
				{Value: "pause", Text: "Pause"},
			})
			info.AddField("MaxAttempts", "max_attempts", db.Int)
			info.FieldSortable()
			formList.AddField("MaxAttempts", "max_attempts", db.Int, form.Text)
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
			info.AddField("NamespaceDefinition", "namespace_definition", db.Enum)
			info.FieldSortable()
			formList.AddField("NamespaceDefinition", "namespace_definition", db.Enum, form.SelectSingle)
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
				{Value: "custom", Text: "Custom"},
			})
			info.AddField("CustomNamespaceFormat", "custom_namespace_format", db.Text)
			info.FieldSortable()
			formList.AddField("CustomNamespaceFormat", "custom_namespace_format", db.Text, form.Text)
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
			info.AddField("StreamPrefix", "stream_prefix", db.Text)
			info.FieldSortable()
			formList.AddField("StreamPrefix", "stream_prefix", db.Text, form.Text)
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
			info.AddField("NextScheduledAt", "next_scheduled_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("NextScheduledAt", "next_scheduled_at", db.Timestamp, form.Datetime)
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
		"job": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"jobs").SetTitle("Job").SetDescription("Job")
			formList.SetTable("pipeline\".\"jobs").SetTitle("Job").SetDescription("Job")
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
			info.AddField("Status", "status", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "scheduled", Text: "Scheduled"},
					{Value: "starting", Text: "Starting"},
					{Value: "running", Text: "Running"},
					{Value: "completed", Text: "Completed"},
					{Value: "failed", Text: "Failed"},
					{Value: "cancelled", Text: "Cancelled"},
				},
			},
			)
			formList.AddField("Status", "status", db.Enum, form.SelectSingle)
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
				{Value: "scheduled", Text: "Scheduled"},
				{Value: "starting", Text: "Starting"},
				{Value: "running", Text: "Running"},
				{Value: "completed", Text: "Completed"},
				{Value: "failed", Text: "Failed"},
				{Value: "cancelled", Text: "Cancelled"},
			})
			info.AddField("JobType", "job_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "sync", Text: "Sync"},
					{Value: "discover", Text: "Discover"},
					{Value: "check", Text: "Check"},
				},
			},
			)
			formList.AddField("JobType", "job_type", db.Enum, form.SelectSingle)
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
				{Value: "sync", Text: "Sync"},
				{Value: "discover", Text: "Discover"},
				{Value: "check", Text: "Check"},
			})
			info.AddField("ScheduledAt", "scheduled_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("ScheduledAt", "scheduled_at", db.Timestamp, form.Datetime)
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
			info.AddField("StartedAt", "started_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
			formList.AddField("StartedAt", "started_at", db.Timestamp, form.Datetime)
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
			info.AddField("CompletedAt", "completed_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("CompletedAt", "completed_at", db.Timestamp, form.Datetime)
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
			info.AddField("Error", "error", db.Text)
			info.FieldSortable()
			formList.AddField("Error", "error", db.Text, form.Text)
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
			info.AddField("Attempt", "attempt", db.Int)
			info.FieldSortable()
			formList.AddField("Attempt", "attempt", db.Int, form.Text)
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
			info.AddField("MaxAttempts", "max_attempts", db.Int)
			info.FieldSortable()
			formList.AddField("MaxAttempts", "max_attempts", db.Int, form.Text)
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
			info.AddField("WorkerID", "worker_id", db.Text)
			info.FieldSortable()
			formList.AddField("WorkerID", "worker_id", db.Text, form.Text)
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
			info.AddField("HeartbeatAt", "heartbeat_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
			formList.AddField("HeartbeatAt", "heartbeat_at", db.Timestamp, form.Datetime)
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
			info.AddField("K8sJobName", "k8s_job_name", db.Text)
			info.FieldSortable()
			formList.AddField("K8sJobName", "k8s_job_name", db.Text, form.Text)
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
			info.AddField("FailureReason", "failure_reason", db.Text)
			info.FieldSortable()
			formList.AddField("FailureReason", "failure_reason", db.Text, form.Text)
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
		"job_attempt": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"job_attempts").SetTitle("JobAttempt").SetDescription("JobAttempt")
			formList.SetTable("pipeline\".\"job_attempts").SetTitle("JobAttempt").SetDescription("JobAttempt")
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
			info.AddField("JobID", "job_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("JobID", "job_id", db.UUID, form.Text)
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
			info.AddField("AttemptNumber", "attempt_number", db.Int)
			info.FieldSortable()
			formList.AddField("AttemptNumber", "attempt_number", db.Int, form.Text)
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
			info.AddField("StartedAt", "started_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("StartedAt", "started_at", db.Timestamp, form.Datetime)
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
			info.AddField("CompletedAt", "completed_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("CompletedAt", "completed_at", db.Timestamp, form.Datetime)
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
			info.AddField("Error", "error", db.Text)
			info.FieldSortable()
			formList.AddField("Error", "error", db.Text, form.Text)
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
			info.AddField("SyncStatsJSON", "sync_stats_json", db.Text)
			info.FieldSortable()
			formList.AddField("SyncStatsJSON", "sync_stats_json", db.Text, form.Text)
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
		"catalog_discovery": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"catalog_discoveries").SetTitle("CatalogDiscovery").SetDescription("CatalogDiscovery")
			formList.SetTable("pipeline\".\"catalog_discoveries").SetTitle("CatalogDiscovery").SetDescription("CatalogDiscovery")
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
			info.AddField("SourceID", "source_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("SourceID", "source_id", db.UUID, form.Text)
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
			info.AddField("Version", "version", db.Int)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("Version", "version", db.Int, form.Text)
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
			info.AddField("CatalogJSON", "catalog_json", db.Text)
			info.FieldSortable()
			formList.AddField("CatalogJSON", "catalog_json", db.Text, form.Text)
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
			info.AddField("DiscoveredAt", "discovered_at", db.Timestamp)
			info.FieldSortable()
			formList.AddField("DiscoveredAt", "discovered_at", db.Timestamp, form.Datetime)
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
		"configured_catalog": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"configured_catalogs").SetTitle("ConfiguredCatalog").SetDescription("ConfiguredCatalog")
			formList.SetTable("pipeline\".\"configured_catalogs").SetTitle("ConfiguredCatalog").SetDescription("ConfiguredCatalog")
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
			info.AddField("StreamsJSON", "streams_json", db.Text)
			info.FieldSortable()
			formList.AddField("StreamsJSON", "streams_json", db.Text, form.Text)
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
		"job_log": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.Int
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"job_logs").SetTitle("JobLog").SetDescription("JobLog")
			formList.SetTable("pipeline\".\"job_logs").SetTitle("JobLog").SetDescription("JobLog")
			info.AddField("ID", "id", db.Int)
			info.FieldSortable()
			formList.AddField("ID", "id", db.Int, form.Text)
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
			info.AddField("JobID", "job_id", db.UUID)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("JobID", "job_id", db.UUID, form.Text)
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
			info.AddField("LogLine", "log_line", db.Text)
			info.FieldSortable()
			formList.AddField("LogLine", "log_line", db.Text, form.Text)
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
		"connection_state": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "connection_id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"connection_state").SetTitle("ConnectionState").SetDescription("ConnectionState")
			formList.SetTable("pipeline\".\"connection_state").SetTitle("ConnectionState").SetDescription("ConnectionState")
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
			info.AddField("StateType", "state_type", db.Text)
			info.FieldSortable()
			formList.AddField("StateType", "state_type", db.Text, form.Text)
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
			info.AddField("StateBlob", "state_blob", db.Text)
			info.FieldSortable()
			formList.AddField("StateBlob", "state_blob", db.Text, form.Text)
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

		"workspace_settings": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "workspace_id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"workspace_settings").SetTitle("WorkspaceSettings").SetDescription("WorkspaceSettings")
			formList.SetTable("pipeline\".\"workspace_settings").SetTitle("WorkspaceSettings").SetDescription("WorkspaceSettings")
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
			info.AddField("MaxJobsPerWorkspace", "max_jobs_per_workspace", db.Int)
			info.FieldSortable()
			formList.AddField("MaxJobsPerWorkspace", "max_jobs_per_workspace", db.Int, form.Text)
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
		"connector_task": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"connector_tasks").SetTitle("ConnectorTask").SetDescription("ConnectorTask")
			formList.SetTable("pipeline\".\"connector_tasks").SetTitle("ConnectorTask").SetDescription("ConnectorTask")
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
			info.AddField("TaskType", "task_type", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "check", Text: "Check"},
					{Value: "spec", Text: "Spec"},
					{Value: "discover", Text: "Discover"},
				},
			},
			)
			formList.AddField("TaskType", "task_type", db.Enum, form.SelectSingle)
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
				{Value: "check", Text: "Check"},
				{Value: "spec", Text: "Spec"},
				{Value: "discover", Text: "Discover"},
			})
			info.AddField("Status", "status", db.Enum)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.SelectSingle,

				Options: types.FieldOptions{
					{Value: "pending", Text: "Pending"},
					{Value: "running", Text: "Running"},
					{Value: "completed", Text: "Completed"},
					{Value: "failed", Text: "Failed"},
				},
			},
			)
			formList.AddField("Status", "status", db.Enum, form.SelectSingle)
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
				{Value: "pending", Text: "Pending"},
				{Value: "running", Text: "Running"},
				{Value: "completed", Text: "Completed"},
				{Value: "failed", Text: "Failed"},
			})
			info.AddField("Payload", "payload", db.JSON)
			info.FieldSortable()
			formList.AddField("Payload", "payload", db.JSON, form.Code)
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
			info.AddField("Result", "result", db.JSON)
			info.FieldSortable()
			formList.AddField("Result", "result", db.JSON, form.Code)
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
			info.AddField("ErrorMessage", "error_message", db.Text)
			info.FieldSortable()
			formList.AddField("ErrorMessage", "error_message", db.Text, form.Text)
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
			info.AddField("WorkerID", "worker_id", db.Text)
			info.FieldSortable()
			formList.AddField("WorkerID", "worker_id", db.Text, form.Text)
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
			info.AddField("UpdatedAt", "updated_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
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
			info.AddField("CompletedAt", "completed_at", db.Timestamp)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Datetime,
			},
			)
			formList.AddField("CompletedAt", "completed_at", db.Timestamp, form.Datetime)
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
		"stream_generation": func(ctx *context.Context) table.Table {
			tableConfig := table.DefaultConfigWithDriver("postgresql")
			tableConfig.PrimaryKey.Type = db.UUID
			tableConfig.PrimaryKey.Name = "connection_id"
			table := table.NewDefaultTable(ctx, tableConfig)
			info := table.GetInfo()
			formList := table.GetForm()
			info.SetTable("pipeline\".\"stream_generations").SetTitle("StreamGeneration").SetDescription("StreamGeneration")
			formList.SetTable("pipeline\".\"stream_generations").SetTitle("StreamGeneration").SetDescription("StreamGeneration")
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
			info.AddField("StreamNamespace", "stream_namespace", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("StreamNamespace", "stream_namespace", db.Text, form.Text)
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
			info.AddField("StreamName", "stream_name", db.Text)
			info.FieldSortable()
			info.FieldFilterable(types.FilterType{
				FormType: form.Text,
			},
			)
			formList.AddField("StreamName", "stream_name", db.Text, form.Text)
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
			info.AddField("GenerationID", "generation_id", db.Int)
			info.FieldSortable()
			formList.AddField("GenerationID", "generation_id", db.Int, form.Text)
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
