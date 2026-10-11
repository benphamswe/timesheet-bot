// Package config loads non-secret settings from YAML and secrets from the environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
)

// Config holds the settings read from the YAML file. The fields tagged
// yaml:"-" are derived from the YAML fields by [Load] and are valid only on a
// Config that Load returned.
type Config struct {
	Timezone        string            `yaml:"timezone"`
	JiraAccountID   string            `yaml:"jira_account_id"`
	SlackUserID     string            `yaml:"slack_user_id"`
	DailyCapMinutes int               `yaml:"daily_cap_minutes"`
	PerKeyMinutes   int               `yaml:"per_key_minutes"`
	TempoBaseURL    string            `yaml:"tempo_base_url"`
	ChannelDefaults map[string]string `yaml:"channel_defaults"` // channel name (no #) -> issue key
	SubmitTargets   []string          `yaml:"submit_targets"`   // TargetTempo, TargetMavenlink
	Mavenlink       Mavenlink         `yaml:"mavenlink"`
	Activity        Activity          `yaml:"activity"`
	Allocation      Allocation        `yaml:"allocation"`
	Report          Report            `yaml:"report"`

	Location *time.Location             `yaml:"-"`
	DailyCap kernel.Duration            `yaml:"-"`
	PerKey   kernel.Duration            `yaml:"-"`
	Channels map[string]kernel.IssueKey `yaml:"-"`
}

// Allocation modes accepted in allocation.mode.
const (
	ModeJiraKeys = "jira_keys" // 45m per Jira issue key per day, from any message (needs Jira)
	ModeReport   = "report"    // the owner's own Slack timesheet summaries, no Jira (ADR-0012)
)

// Allocation is the `allocation` block.
type Allocation struct {
	Mode string `yaml:"mode"` // ModeJiraKeys (default) or ModeReport
}

// Report is the `report` block: where the owner writes timesheet summaries.
// The allocation-agent collector reads the same keys to choose what it
// collects, with the same rules: an absent include_self_dm is false, and the
// channel loses a leading '#' and must match [a-z0-9._-]+.
type Report struct {
	Channel       string `yaml:"channel"`         // summary channel name without '#'
	IncludeSelfDM bool   `yaml:"include_self_dm"` // also the owner's direct messages to themselves; absent means false
	Parser        string `yaml:"parser"`          // ParserManual (default) or ParserLLM
}

// Report parsers accepted in report.parser. The `llm` block (provider, model,
// base_url, timeout_seconds, max_retries) is read by the allocation-agent
// collector only; Load ignores it like any key it does not know.
const (
	ParserManual = "manual" // the hand-written reportparser.Parser, in Go
	ParserLLM    = "llm"    // report_lines the collector's LLM wrote into the activity file (version 2)
)

// ReportParserLLM reports whether report mode uses the collector's parsed lines.
func (c *Config) ReportParserLLM() bool { return c.ReportMode() && c.Report.Parser == ParserLLM }

var reportChannelRe = regexp.MustCompile(`^[a-z0-9._-]+$`)

// ReportMode reports whether allocation.mode is ModeReport.
func (c *Config) ReportMode() bool { return c.Allocation.Mode == ModeReport }

// OwnerID is the id that owns timesheets: jira_account_id in jira_keys mode,
// where it is also the Tempo author, and slack_user_id in report mode, which
// has no Jira account.
func (c *Config) OwnerID() string {
	if c.ReportMode() {
		return c.SlackUserID
	}
	return c.JiraAccountID
}

// Activity sources accepted in activity.source.
const (
	ActivitySlack = "slack" // Slack search.messages with SLACK_USER_TOKEN
	ActivityFile  = "file"  // week files written by the allocation-agent collector
)

// DefaultActivityFileDir is activity.file_dir when unset. Like every relative
// file_dir it is resolved against the config file's directory, so with the
// usual services/timesheet-core/config/config.yaml it is the repository
// root's data/activity, and in the container (config at /app/config) it is
// /data/activity.
const DefaultActivityFileDir = "../../../data/activity"

// Activity is the `activity` block: where the week's work evidence comes from.
type Activity struct {
	Source  string `yaml:"source"`   // ActivitySlack (default) or ActivityFile
	FileDir string `yaml:"file_dir"` // absolute, or relative to the config file's directory
	Dir     string `yaml:"-"`        // FileDir resolved by Load, absolute
}

// Submit targets accepted in submit_targets.
const (
	TargetTempo     = "tempo"
	TargetMavenlink = "mavenlink"
)

// Mavenlink auth modes accepted in mavenlink.auth.
const (
	AuthBearer = "bearer"
	AuthCookie = "cookie"
)

// Defaults for mavenlink.auth: cookie, where the session cookie is valid only
// on the account's own host.
const (
	DefaultMavenlinkInstanceURL = "https://27global.mavenlink.com"
	DefaultMavenlinkCookieBase  = DefaultMavenlinkInstanceURL + "/api/v1"
	DefaultMavenlinkCSRFPage    = DefaultMavenlinkInstanceURL + "/timesheets"
)

// Mavenlink is the `mavenlink` block, read when submit_targets has
// TargetMavenlink.
type Mavenlink struct {
	Auth               string             `yaml:"auth"`          // AuthBearer (default) or AuthCookie
	BaseURL            string             `yaml:"base_url"`      // empty: the default for Auth
	CSRFPageURL        string             `yaml:"csrf_page_url"` // cookie auth only
	SubmitForApproval  bool               `yaml:"submit_for_approval"`
	DefaultWorkspaceID int64              `yaml:"default_workspace_id"`
	DefaultStoryID     int64              `yaml:"default_story_id"`
	DefaultRoleID      int64              `yaml:"default_role_id"`
	Mappings           []MavenlinkMapping `yaml:"mappings"`
}

// MavenlinkMapping is one rule sending entries to a Mavenlink project. Set
// issue_key and/or jira_project for jira_keys mode, project for report mode.
type MavenlinkMapping struct {
	IssueKey    string `yaml:"issue_key"`
	JiraProject string `yaml:"jira_project"`
	Project     string `yaml:"project"` // report project name, matched case-insensitively
	WorkspaceID int64  `yaml:"workspace_id"`
	StoryID     int64  `yaml:"story_id"`
	RoleID      int64  `yaml:"role_id"`
	Billable    *bool  `yaml:"billable"`
}

// SubmitsTo reports whether submit_targets has target.
func (c *Config) SubmitsTo(target string) bool { return slices.Contains(c.SubmitTargets, target) }

// Load reads and validates the YAML file at path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is the operator's own --config flag; reading the file they name is the purpose
	if err != nil {
		return nil, err
	}
	c := Config{
		Timezone: "Asia/Ho_Chi_Minh", DailyCapMinutes: 480, PerKeyMinutes: 45,
		SubmitTargets: []string{TargetTempo},
		Mavenlink:     Mavenlink{Auth: AuthBearer, SubmitForApproval: true},
		Activity:      Activity{Source: ActivitySlack, FileDir: DefaultActivityFileDir},
		Allocation:    Allocation{Mode: ModeJiraKeys},
		Report:        Report{Parser: ParserManual},
	}
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.resolve(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

// resolve validates c and fills the derived fields. configDir is the config
// file's directory, which relative paths are resolved against.
func (c *Config) resolve(configDir string) error {
	var errs []string
	var err error
	if c.Location, err = time.LoadLocation(c.Timezone); err != nil {
		errs = append(errs, err.Error())
	}
	if c.DailyCap, err = kernel.NewDuration(c.DailyCapMinutes); err != nil {
		errs = append(errs, "daily_cap_minutes: "+err.Error())
	}
	if c.PerKey, err = kernel.NewDuration(c.PerKeyMinutes); err != nil {
		errs = append(errs, "per_key_minutes: "+err.Error())
	}
	if c.JiraAccountID == "" && !c.ReportMode() {
		errs = append(errs, "jira_account_id is required")
	}
	if c.SlackUserID == "" {
		errs = append(errs, "slack_user_id is required")
	}
	c.Channels = map[string]kernel.IssueKey{}
	for ch, k := range c.ChannelDefaults {
		key, err := kernel.NewIssueKey(k)
		if err != nil {
			errs = append(errs, fmt.Sprintf("channel_defaults.%s: %v", ch, err))
			continue
		}
		c.Channels[strings.TrimPrefix(ch, "#")] = key
	}
	errs = append(errs, c.resolveTargets()...)
	errs = append(errs, c.resolveAllocation()...)
	switch c.Activity.Source {
	case ActivitySlack, ActivityFile:
	default:
		errs = append(errs, fmt.Sprintf("activity.source: unknown source %q, want slack or file", c.Activity.Source))
	}
	if c.Activity.FileDir == "" {
		errs = append(errs, "activity.file_dir must not be empty")
	}
	c.Activity.Dir = c.Activity.FileDir
	if !filepath.IsAbs(c.Activity.Dir) {
		c.Activity.Dir = filepath.Join(configDir, c.Activity.Dir)
	}
	// Absolute, so errors name the real location whatever the working directory.
	if abs, err := filepath.Abs(c.Activity.Dir); err == nil {
		c.Activity.Dir = abs
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("invalid config: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Config) resolveAllocation() []string {
	var errs []string
	c.Report.Channel = strings.TrimPrefix(c.Report.Channel, "#")
	if c.Report.Channel != "" && !reportChannelRe.MatchString(c.Report.Channel) {
		errs = append(errs, fmt.Sprintf("report.channel: invalid channel name %q, want lowercase letters, digits, '.', '_' or '-'", c.Report.Channel))
	}
	if c.Report.Parser != ParserManual && c.Report.Parser != ParserLLM {
		errs = append(errs, fmt.Sprintf("report.parser: unknown parser %q, want manual or llm", c.Report.Parser))
	}
	switch c.Allocation.Mode {
	case ModeJiraKeys:
		return errs
	case ModeReport:
	default:
		return append(errs, fmt.Sprintf("allocation.mode: unknown mode %q, want jira_keys or report", c.Allocation.Mode))
	}
	if c.Report.Channel == "" && !c.Report.IncludeSelfDM {
		errs = append(errs, "report: set report.channel, include_self_dm, or both, in report mode")
	}
	if c.SubmitsTo(TargetTempo) {
		errs = append(errs, "submit_targets: tempo needs Jira issues, which report mode does not have; use [mavenlink]")
	}
	if c.Activity.Source != ActivityFile {
		errs = append(errs, "activity.source: report mode reads the summaries the allocation-agent collector writes; set activity.source: file")
	}
	return errs
}

func (c *Config) resolveTargets() []string {
	var errs []string
	if len(c.SubmitTargets) == 0 {
		errs = append(errs, "submit_targets: want at least one of tempo, mavenlink")
	}
	seen := map[string]bool{}
	for _, t := range c.SubmitTargets {
		switch {
		case t != TargetTempo && t != TargetMavenlink:
			errs = append(errs, fmt.Sprintf("submit_targets: unknown target %q, want tempo or mavenlink", t))
		case seen[t]:
			errs = append(errs, fmt.Sprintf("submit_targets: %s listed twice", t))
		}
		seen[t] = true
	}
	if !c.SubmitsTo(TargetMavenlink) {
		return errs
	}

	m := &c.Mavenlink
	switch m.Auth {
	case AuthBearer:
	case AuthCookie:
		if m.BaseURL == "" {
			m.BaseURL = DefaultMavenlinkCookieBase
		}
		if m.CSRFPageURL == "" {
			m.CSRFPageURL = DefaultMavenlinkCSRFPage
		}
	default:
		errs = append(errs, fmt.Sprintf("mavenlink.auth: unknown mode %q, want bearer or cookie", m.Auth))
	}
	issueRules, projectRules := map[string]int{}, map[string]int{}
	for i, r := range m.Mappings {
		field := fmt.Sprintf("mavenlink.mappings[%d]", i)
		project := alloc.NormalizeReportText(r.Project)
		if r.IssueKey == "" && r.JiraProject == "" && project == "" {
			errs = append(errs, field+": set issue_key, jira_project or project")
		}
		if project != "" && (r.IssueKey != "" || r.JiraProject != "") {
			errs = append(errs, field+": project (report mode) cannot be combined with issue_key or jira_project")
		}
		if r.IssueKey != "" {
			if _, err := kernel.NewIssueKey(r.IssueKey); err != nil {
				errs = append(errs, fmt.Sprintf("%s.issue_key: %v", field, err))
			}
			k := strings.ToUpper(r.IssueKey)
			if first, ok := issueRules[k]; ok {
				errs = append(errs, fmt.Sprintf("mavenlink.mappings[%d] and [%d]: both map issue_key %q; keep one", first, i, r.IssueKey))
			} else {
				issueRules[k] = i
			}
		}
		if project != "" {
			if first, ok := projectRules[project]; ok {
				errs = append(errs, fmt.Sprintf("mavenlink.mappings[%d] and [%d]: both map project %q (names are compared ignoring case and spaces); keep one", first, i, r.Project))
			} else {
				projectRules[project] = i
			}
		}
		if r.JiraProject != "" && !jiraProjectRe.MatchString(r.JiraProject) {
			errs = append(errs, fmt.Sprintf("%s.jira_project: invalid project key %q", field, r.JiraProject))
		}
		if r.WorkspaceID <= 0 {
			errs = append(errs, field+".workspace_id is required")
		}
		if r.StoryID < 0 || r.RoleID < 0 {
			errs = append(errs, field+": story_id and role_id must not be negative")
		}
	}
	if m.DefaultWorkspaceID < 0 || m.DefaultStoryID < 0 || m.DefaultRoleID < 0 {
		errs = append(errs, "mavenlink: default ids must not be negative")
	}
	return errs
}

var jiraProjectRe = regexp.MustCompile(`^[A-Z][A-Z0-9]+$`)

// Secret environment variable names. Values are never logged or printed.
const (
	EnvSlackUserToken = "SLACK_USER_TOKEN"
	EnvTempoToken     = "TEMPO_API_TOKEN" //nolint:gosec // G101: an environment variable name, not a credential
	EnvJiraBaseURL    = "JIRA_BASE_URL"
	EnvJiraEmail      = "JIRA_EMAIL"
	EnvJiraToken      = "JIRA_API_TOKEN" //nolint:gosec // G101: an environment variable name, not a credential
	EnvDatabaseURL    = "DATABASE_URL"

	EnvMavenlinkToken  = "MAVENLINK_API_TOKEN"      //nolint:gosec // G101: an environment variable name, not a credential
	EnvMavenlinkCookie = "MAVENLINK_SESSION_COOKIE" //nolint:gosec // G101: an environment variable name, not a credential
)

// Env reads required variables, reporting every missing name (never values).
func Env(names ...string) (map[string]string, error) {
	vals := make(map[string]string, len(names))
	var missing []string
	for _, n := range names {
		v := os.Getenv(n)
		if v == "" {
			missing = append(missing, n)
		}
		vals[n] = v
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment variables: %s (export them in ~/.zshrc)", strings.Join(missing, ", "))
	}
	return vals, nil
}
