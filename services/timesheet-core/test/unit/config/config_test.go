package config_test

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"timesheet-bot/internal/config"
	"timesheet-bot/internal/shared/kernel"
)

// writeConfig writes body to a config file in a per-test directory that
// t.TempDir removes when the test ends.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadAppliesDefaultsAndStripsChannelHash(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
jira_account_id: acc-1
slack_user_id: U1
channel_defaults:
  "#proj-hub": HUB-1
  dev: GCM-7
`)

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if got, want := c.Location.String(), "Asia/Ho_Chi_Minh"; got != want {
		t.Errorf("Load().Location = %q, want %q", got, want)
	}
	if got, want := c.DailyCap, kernel.MustDuration(480); got != want {
		t.Errorf("Load().DailyCap = %v, want %v", got, want)
	}
	if got, want := c.PerKey, kernel.MustDuration(45); got != want {
		t.Errorf("Load().PerKey = %v, want %v", got, want)
	}
	wantChannels := map[string]kernel.IssueKey{"proj-hub": "HUB-1", "dev": "GCM-7"}
	if !maps.Equal(c.Channels, wantChannels) {
		t.Errorf("Load().Channels = %v, want %v", c.Channels, wantChannels)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
timezone: UTC
jira_account_id: acc-1
slack_user_id: U1
daily_cap_minutes: 420
per_key_minutes: 30
`)

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if got, want := c.Location.String(), "UTC"; got != want {
		t.Errorf("Load().Location = %q, want %q", got, want)
	}
	if got, want := c.DailyCap, kernel.MustDuration(420); got != want {
		t.Errorf("Load().DailyCap = %v, want %v", got, want)
	}
	if got, want := c.PerKey, kernel.MustDuration(30); got != want {
		t.Errorf("Load().PerKey = %v, want %v", got, want)
	}
}

// Load joins every validation problem into one error string and has no
// sentinels, so these rows check that the message names each bad field.
func TestLoadReportsEveryInvalidField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		body   string
		fields []string
	}{
		{
			name: "missing ids, bad per key and bad channel key",
			body: `
per_key_minutes: 40
channel_defaults:
  dev: not-a-key
`,
			fields: []string{"jira_account_id", "slack_user_id", "per_key_minutes", "channel_defaults.dev"},
		},
		{
			name: "zero daily cap",
			body: `
jira_account_id: acc-1
slack_user_id: U1
daily_cap_minutes: 0
`,
			fields: []string{"daily_cap_minutes"},
		},
		{
			name: "unknown timezone",
			body: `
timezone: Mars/Olympus_Mons
jira_account_id: acc-1
slack_user_id: U1
`,
			fields: []string{"Mars/Olympus_Mons"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeConfig(t, tt.body)
			_, err := config.Load(path)
			if err == nil {
				t.Fatalf("Load(%q) error = nil, want an error naming %v", path, tt.fields)
			}
			for _, field := range tt.fields {
				if !strings.Contains(err.Error(), field) {
					t.Errorf("Load(%q) error = %q, want it to name %q", path, err, field)
				}
			}
		})
	}
}

func TestLoadMissingFileIsNotExist(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "absent.yaml")
	if _, err := config.Load(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load(%q) error = %v, want fs.ErrNotExist", path, err)
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: [unclosed\n")
	c, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) = %+v, nil, want a parse error", path, c)
	}
	if c != nil {
		t.Errorf("Load(%q) config = %+v, want nil on error", path, c)
	}
}

// The Env tests use t.Setenv, so they must not call t.Parallel: the testing
// package panics when Setenv runs in a parallel test.

func TestEnvReturnsValuesWhenAllSet(t *testing.T) {
	t.Setenv("TSBOT_TEST_A", "tok-a")
	t.Setenv("TSBOT_TEST_B", "tok-b")

	got, err := config.Env("TSBOT_TEST_A", "TSBOT_TEST_B")
	if err != nil {
		t.Fatalf("Env() error = %v, want nil", err)
	}
	want := map[string]string{"TSBOT_TEST_A": "tok-a", "TSBOT_TEST_B": "tok-b"}
	if !maps.Equal(got, want) {
		t.Errorf("Env() = %v, want %v", got, want)
	}
}

func TestEnvNamesMissingWithoutValues(t *testing.T) {
	t.Setenv("TSBOT_TEST_SET", "secret-value")
	t.Setenv("TSBOT_TEST_EMPTY", "")
	// t.Setenv records the old value and restores it in cleanup, so the
	// variable can then be removed for this test only.
	t.Setenv("TSBOT_TEST_UNSET", "")
	if err := os.Unsetenv("TSBOT_TEST_UNSET"); err != nil {
		t.Fatalf("unset TSBOT_TEST_UNSET: %v", err)
	}

	got, err := config.Env("TSBOT_TEST_SET", "TSBOT_TEST_EMPTY", "TSBOT_TEST_UNSET")
	if err == nil {
		t.Fatalf("Env() = %v, nil, want an error", got)
	}
	if got != nil {
		t.Errorf("Env() values = %v, want nil on error", got)
	}
	for _, name := range []string{"TSBOT_TEST_EMPTY", "TSBOT_TEST_UNSET"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Env() error = %q, want it to name %s", err, name)
		}
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Errorf("Env() error = %q, want it to omit every value", err)
	}
	if strings.Contains(err.Error(), "TSBOT_TEST_SET") {
		t.Errorf("Env() error = %q, want it to omit variables that are set", err)
	}
}

func TestLoadSubmitTargetDefaults(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\n")

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if !slices.Equal(c.SubmitTargets, []string{config.TargetTempo}) {
		t.Errorf("Load().SubmitTargets = %v, want [tempo] (unchanged behavior)", c.SubmitTargets)
	}
	if c.Mavenlink.Auth != config.AuthBearer || !c.Mavenlink.SubmitForApproval || c.Mavenlink.BaseURL != "" {
		t.Errorf("Load().Mavenlink = %+v, want bearer auth, submit_for_approval true, empty base_url", c.Mavenlink)
	}
}

func TestLoadMavenlinkCookieDefaultsAndMappings(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
jira_account_id: acc-1
slack_user_id: U1
submit_targets: [mavenlink]
mavenlink:
  auth: cookie
  submit_for_approval: false
  default_workspace_id: 1
  mappings:
    - jira_project: GCM
      workspace_id: 7
      story_id: 70
      role_id: 5
    - issue_key: GCM-12
      workspace_id: 9
      billable: false
`)

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if !c.SubmitsTo(config.TargetMavenlink) || c.SubmitsTo(config.TargetTempo) {
		t.Errorf("Load().SubmitTargets = %v, want [mavenlink] only (the default is replaced, not appended to)", c.SubmitTargets)
	}
	m := c.Mavenlink
	if m.BaseURL != config.DefaultMavenlinkCookieBase || m.CSRFPageURL != config.DefaultMavenlinkCSRFPage || m.SubmitForApproval {
		t.Errorf("Load().Mavenlink = %+v, want the instance URLs for cookie auth and submit_for_approval false", m)
	}
	if len(m.Mappings) != 2 || m.Mappings[0].RoleID != 5 || m.Mappings[1].Billable == nil || *m.Mappings[1].Billable {
		t.Errorf("Load().Mavenlink.Mappings = %+v, want two rules, role 5 on the first, billable false on the second", m.Mappings)
	}
}

func TestLoadRejectsInvalidMavenlinkConfig(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
jira_account_id: acc-1
slack_user_id: U1
submit_targets: [tempo, mavenlink, mavenlink, jira]
mavenlink:
  auth: password
  mappings:
    - workspace_id: 7
    - issue_key: not-a-key
      jira_project: gcm
`)

	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) error = nil, want an invalid config error", path)
	}
	for _, field := range []string{
		`unknown target "jira"`, "mavenlink listed twice", `mavenlink.auth: unknown mode "password"`,
		"mavenlink.mappings[0]: set issue_key, jira_project or project", "mavenlink.mappings[1].issue_key",
		"mavenlink.mappings[1].jira_project", "mavenlink.mappings[1].workspace_id is required",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Load() error = %q, want it to mention %q", err, field)
		}
	}
}

func TestLoadIgnoresMavenlinkBlockWhenNotATarget(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nmavenlink:\n  auth: password\n")
	if _, err := config.Load(path); err != nil {
		t.Errorf("Load(%q) error = %v, want nil while mavenlink is not in submit_targets", path, err)
	}
}

func TestLoadActivityDefaults(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\n")

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if c.Activity.Source != config.ActivitySlack {
		t.Errorf("Load().Activity.Source = %q, want %q (unchanged behavior)", c.Activity.Source, config.ActivitySlack)
	}
	if want := filepath.Join(filepath.Dir(path), "..", "..", "..", "data", "activity"); c.Activity.Dir != want {
		t.Errorf("Load().Activity.Dir = %q, want %q (default resolved against the config file's directory)", c.Activity.Dir, want)
	}
}

func TestLoadActivityFileDirResolution(t *testing.T) {
	t.Parallel()
	abs := filepath.Join(t.TempDir(), "weeks")
	tests := []struct {
		name, fileDir string
		want          func(configDir string) string
	}{
		{name: "relative to the config file", fileDir: "activity", want: func(d string) string { return filepath.Join(d, "activity") }},
		{name: "absolute kept", fileDir: abs, want: func(string) string { return abs }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nactivity:\n  source: file\n  file_dir: "+tt.fileDir+"\n")
			c, err := config.Load(path)
			if err != nil {
				t.Fatalf("Load(%q) error = %v, want nil", path, err)
			}
			if want := tt.want(filepath.Dir(path)); c.Activity.Source != config.ActivityFile || c.Activity.Dir != want {
				t.Errorf("Load().Activity = %+v, want source file and Dir %q", c.Activity, want)
			}
		})
	}
}

func TestLoadRejectsInvalidActivity(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nactivity:\n  source: email\n  file_dir: \"\"\n")
	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) error = nil, want an invalid config error", path)
	}
	for _, field := range []string{`activity.source: unknown source "email"`, "activity.file_dir must not be empty"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Load() error = %q, want it to mention %q", err, field)
		}
	}
}

func TestLoadReportMode(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
slack_user_id: U1
allocation:
  mode: report
report:
  channel: "#timesheet"
activity:
  source: file
submit_targets: [mavenlink]
mavenlink:
  mappings:
    - project: Atlas
      workspace_id: 7
`)

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil (jira_account_id is not needed in report mode)", path, err)
	}
	if !c.ReportMode() || c.Report.Channel != "timesheet" || c.Report.IncludeSelfDM {
		t.Errorf("Load() = mode %q, report %+v, want report mode, channel without '#', include_self_dm false when absent", c.Allocation.Mode, c.Report)
	}
	if c.OwnerID() != "U1" || c.Mavenlink.Mappings[0].Project != "Atlas" {
		t.Errorf("Load() owner %q, mapping %+v, want the Slack user as owner and the Atlas project rule", c.OwnerID(), c.Mavenlink.Mappings[0])
	}
}

func TestLoadDefaultsToJiraKeysMode(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\n")
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v, want nil", path, err)
	}
	if c.ReportMode() || c.Allocation.Mode != config.ModeJiraKeys || c.OwnerID() != "acc-1" {
		t.Errorf("Load() mode %q, owner %q, want jira_keys and the Jira account as owner", c.Allocation.Mode, c.OwnerID())
	}
}

func TestLoadRejectsInvalidReportMode(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
slack_user_id: U1
allocation:
  mode: report
report:
  include_self_dm: false
submit_targets: [tempo, mavenlink]
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) error = nil, want an invalid config error", path)
	}
	for _, field := range []string{
		"report: set report.channel, include_self_dm, or both",
		"submit_targets: tempo needs Jira issues",
		"activity.source: report mode reads the summaries",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Load() error = %q, want it to mention %q", err, field)
		}
	}
	if strings.Contains(err.Error(), "jira_account_id") {
		t.Errorf("Load() error = %q, want no jira_account_id error in report mode", err)
	}
}

// An absent include_self_dm is false, as in the allocation-agent collector, so
// a report block with neither source is an error.
func TestLoadReportModeNeedsASource(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "slack_user_id: U1\nallocation:\n  mode: report\nactivity:\n  source: file\nsubmit_targets: [mavenlink]\n")
	if _, err := config.Load(path); err == nil || !strings.Contains(err.Error(), "report: set report.channel, include_self_dm, or both") {
		t.Errorf("Load() error = %v, want the no-report-source error", err)
	}
}

func TestLoadReportChannelNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		channel, want string
		ok            bool
	}{
		{channel: "timesheet", want: "timesheet", ok: true},
		{channel: "#team.time_sheet-2", want: "team.time_sheet-2", ok: true},
		{channel: "Timesheet", ok: false},
		{channel: "time sheet", ok: false},
		{channel: "#", want: "", ok: true}, // stripped to empty: no channel; the self-DM is the source
	}
	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			t.Parallel()
			path := writeConfig(t, "slack_user_id: U1\nallocation:\n  mode: report\nreport:\n  channel: \""+tt.channel+
				"\"\n  include_self_dm: true\nactivity:\n  source: file\nsubmit_targets: [mavenlink]\n")
			c, err := config.Load(path)
			if tt.ok {
				if err != nil || c.Report.Channel != tt.want {
					t.Errorf("Load(channel %q) = %+v, %v, want channel %q", tt.channel, c, err, tt.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "report.channel") {
				t.Errorf("Load(channel %q) error = %v, want a report.channel error", tt.channel, err)
			}
		})
	}
}

func TestLoadRejectsUnknownAllocationMode(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nallocation:\n  mode: hours\n")
	if _, err := config.Load(path); err == nil || !strings.Contains(err.Error(), `allocation.mode: unknown mode "hours"`) {
		t.Errorf("Load() error = %v, want an unknown allocation.mode error", err)
	}
}

func TestLoadReportParser(t *testing.T) {
	t.Parallel()
	report := "slack_user_id: U1\nallocation:\n  mode: report\nreport:\n  include_self_dm: true\n%sactivity:\n  source: file\nsubmit_targets: [mavenlink]\n"
	tests := []struct {
		name, extra string
		wantLLM     bool
	}{
		{name: "default manual", extra: ""},
		{name: "llm with the collector's llm block", wantLLM: true,
			extra: "  parser: llm\nllm:\n  provider: deepseek\n  model: deepseek-flash\n  base_url: https://api.deepseek.com\n  timeout_seconds: 60\n  max_retries: 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// The llm block is top level; it follows the report block's keys here only for the file layout.
			body := strings.Replace(report, "%s", tt.extra, 1)
			path := writeConfig(t, body)
			c, err := config.Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil (the llm block is the collector's and is ignored)", err)
			}
			if c.ReportParserLLM() != tt.wantLLM {
				t.Errorf("Load().ReportParserLLM() = %v, want %v (parser %q)", c.ReportParserLLM(), tt.wantLLM, c.Report.Parser)
			}
		})
	}
}

func TestLoadRejectsUnknownReportParser(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nreport:\n  parser: regex\n")
	if _, err := config.Load(path); err == nil || !strings.Contains(err.Error(), `report.parser: unknown parser "regex", want manual or llm`) {
		t.Errorf("Load() error = %v, want an unknown report.parser error", err)
	}
}

func TestLoadParserLLMOnlyMattersInReportMode(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "jira_account_id: acc-1\nslack_user_id: U1\nreport:\n  parser: llm\n")
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if c.ReportParserLLM() {
		t.Errorf("Load().ReportParserLLM() = true in jira_keys mode, want false")
	}
}

func TestLoadRejectsAmbiguousMavenlinkRules(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `
jira_account_id: acc-1
slack_user_id: U1
submit_targets: [mavenlink]
mavenlink:
  mappings:
    - project: Atlas
      workspace_id: 7
    - issue_key: GCM-12
      workspace_id: 8
    - project: "  atlas "
      workspace_id: 9
    - issue_key: gcm-12
      workspace_id: 10
    - project: Beacon
      jira_project: GCM
      workspace_id: 11
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) error = nil, want an ambiguous mapping error", path)
	}
	for _, want := range []string{
		`mavenlink.mappings[0] and [2]: both map project "  atlas "`,
		`mavenlink.mappings[1] and [3]: both map issue_key "gcm-12"`,
		"mavenlink.mappings[4]: project (report mode) cannot be combined with issue_key or jira_project",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to mention %q", err, want)
		}
	}
}
