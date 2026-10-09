package domain

// Target is a time tracking system an approved timesheet is written to. The
// string values are used in config files.
type Target string

// Submit targets.
const (
	TargetTempo     Target = "tempo"
	TargetMavenlink Target = "mavenlink"
)
