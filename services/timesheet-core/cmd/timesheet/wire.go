package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"

	fileactivity "timesheet-bot/internal/activity/infrastructure/file"
	slackactivity "timesheet-bot/internal/activity/infrastructure/slack"
	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/allocation/infrastructure/reportparser"
	"timesheet-bot/internal/config"
	"timesheet-bot/internal/interfaces/cli"
	"timesheet-bot/internal/platform/db"
	"timesheet-bot/internal/shared/httpx"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/domain"
	"timesheet-bot/internal/timesheet/infrastructure/jira"
	"timesheet-bot/internal/timesheet/infrastructure/mavenlink"
	"timesheet-bot/internal/timesheet/infrastructure/memory"
	"timesheet-bot/internal/timesheet/infrastructure/postgres"
	"timesheet-bot/internal/timesheet/infrastructure/tempo"
)

// buildApp is the composition root: the only place that knows concrete adapters.
func buildApp(ctx context.Context, configPath string, needs cli.Needs) (*cli.App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	var names []string
	if cfg.Activity.Source == config.ActivitySlack {
		names = append(names, config.EnvSlackUserToken)
	}
	if !cfg.ReportMode() {
		names = append(names, config.EnvJiraBaseURL, config.EnvJiraEmail, config.EnvJiraToken)
	}
	if needs.DB {
		names = append(names, config.EnvDatabaseURL)
	}
	if needs.Submit {
		names = append(names, submitEnv(cfg)...)
	}
	env, err := config.Env(names...)
	if err != nil {
		return nil, err
	}

	app := &cli.App{Location: cfg.Location, Close: func() {}}
	hc := httpx.NewClient(30 * time.Second)
	clock := kernel.SystemClock{Location: cfg.Location}

	var repo domain.Repository = memory.NewRepository()
	if needs.DB {
		pool, err := db.Open(ctx, env[config.EnvDatabaseURL])
		if err != nil {
			return nil, err
		}
		app.Close = pool.Close
		repo = postgres.NewRepository(pool)
	}

	source := activitySource(cfg, env, hc)
	app.Propose = &application.ProposeHandler{
		Repo:            repo,
		Source:          source,
		Policies:        []alloc.Policy{alloc.RulePolicy{}},
		Clock:           clock,
		UserAccountID:   cfg.OwnerID(),
		DailyCap:        cfg.DailyCap,
		PerKey:          cfg.PerKey,
		ChannelDefaults: cfg.Channels,
	}
	switch {
	case cfg.ReportParserLLM():
		reports, ok := source.(*fileactivity.Collector)
		if !ok { // config.Load already requires activity.source: file in report mode
			return nil, errors.New("report.parser: llm needs activity.source: file")
		}
		app.Propose.Reports = reports
		app.Propose.Policies = []alloc.Policy{alloc.ReportPolicy{Precomputed: true, Location: cfg.Location}}
	case cfg.ReportMode():
		app.Propose.Policies = []alloc.Policy{alloc.ReportPolicy{Parser: reportparser.New(), Location: cfg.Location}}
	default:
		app.Propose.Issues = jira.NewClient(env[config.EnvJiraBaseURL], env[config.EnvJiraEmail], env[config.EnvJiraToken], hc)
	}
	if cfg.SubmitsTo(config.TargetMavenlink) {
		app.Propose.Placements = mavenlinkMapping(cfg.Mavenlink)
	}
	app.Transition = &application.TransitionHandler{Repo: repo, Clock: clock, UserAccountID: cfg.OwnerID()}
	if needs.Submit {
		app.Submit = &application.SubmitHandler{Repo: repo, Clock: clock, UserAccountID: cfg.OwnerID()}
		if cfg.SubmitsTo(config.TargetTempo) {
			app.Submit.Gateway = tempo.NewClient(cfg.TempoBaseURL, env[config.EnvTempoToken], hc)
		}
		if cfg.SubmitsTo(config.TargetMavenlink) {
			app.Submit.Mavenlink = &application.MavenlinkSubmit{
				Gateway:           mavenlink.NewClient(cfg.Mavenlink.BaseURL, mavenlinkAuth(cfg.Mavenlink, env, hc), hc),
				Placements:        mavenlinkMapping(cfg.Mavenlink),
				SubmitForApproval: cfg.Mavenlink.SubmitForApproval,
			}
		}
	}
	return app, nil
}

var _ application.ReportLineSource = (*fileactivity.Collector)(nil)

// activitySource is the configured [application.ActivitySource]: Slack
// search, or the allocation-agent's week files.
func activitySource(cfg *config.Config, env map[string]string, hc *http.Client) application.ActivitySource {
	if cfg.Activity.Source == config.ActivityFile {
		return fileactivity.NewCollector(cfg.Activity.Dir, cfg.SlackUserID, cfg.Location)
	}
	api := slack.New(env[config.EnvSlackUserToken], slack.OptionHTTPClient(hc))
	return slackactivity.NewCollector(api, cfg.SlackUserID, cfg.Location)
}

// submitEnv names the secrets the configured submit targets need.
func submitEnv(cfg *config.Config) []string {
	var names []string
	if cfg.SubmitsTo(config.TargetTempo) {
		names = append(names, config.EnvTempoToken)
	}
	if cfg.SubmitsTo(config.TargetMavenlink) {
		if cfg.Mavenlink.Auth == config.AuthCookie {
			names = append(names, config.EnvMavenlinkCookie)
		} else {
			names = append(names, config.EnvMavenlinkToken)
		}
	}
	return names
}

func mavenlinkAuth(m config.Mavenlink, env map[string]string, hc *http.Client) mavenlink.Auth {
	if m.Auth == config.AuthCookie {
		return mavenlink.NewSessionCookie(env[config.EnvMavenlinkCookie], m.CSRFPageURL, hc)
	}
	return mavenlink.NewBearer(env[config.EnvMavenlinkToken])
}

func mavenlinkMapping(m config.Mavenlink) application.MavenlinkMapping {
	out := application.MavenlinkMapping{
		DefaultWorkspaceID: m.DefaultWorkspaceID,
		DefaultStoryID:     m.DefaultStoryID,
		DefaultRoleID:      m.DefaultRoleID,
	}
	for _, r := range m.Mappings {
		out.Rules = append(out.Rules, application.MavenlinkRule{
			IssueKey:    kernel.IssueKey(r.IssueKey), // validated by config.Load
			JiraProject: r.JiraProject,
			Project:     strings.TrimSpace(r.Project),
			WorkspaceID: r.WorkspaceID,
			StoryID:     r.StoryID,
			RoleID:      r.RoleID,
			Billable:    r.Billable,
		})
	}
	return out
}

func migrate(ctx context.Context, direction string) error {
	env, err := config.Env(config.EnvDatabaseURL)
	if err != nil {
		return err
	}
	return db.Migrate(ctx, env[config.EnvDatabaseURL], direction)
}
