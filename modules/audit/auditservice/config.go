package auditservice

// Config bounds how long audit events are kept per workspace.
type Config struct {
	// DefaultRetentionDays applies to workspaces without a stored setting.
	DefaultRetentionDays int
	// MaxRetentionDays is the ceiling no workspace setting may exceed.
	MaxRetentionDays int
}

// EffectiveRetentionDays returns the retention window for a workspace: the
// stored setting when present, otherwise the default, never above the max.
func (c Config) EffectiveRetentionDays(setting *AuditRetentionSetting) int {
	days := c.DefaultRetentionDays
	if setting != nil {
		days = setting.RetentionDays
	}

	return min(days, c.MaxRetentionDays)
}
