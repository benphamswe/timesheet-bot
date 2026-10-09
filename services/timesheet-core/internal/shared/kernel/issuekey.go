package kernel

import (
	"fmt"
	"regexp"
)

// IssueKey is a Jira issue key such as GCM-123.
type IssueKey string

var (
	issueKeyRe   = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	issueKeyScan = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
)

// NewIssueKey returns s as an IssueKey. It returns an error unless s is a whole
// key: an uppercase project key of at least two characters starting with a
// letter, a hyphen, and a number (GCM-123).
func NewIssueKey(s string) (IssueKey, error) {
	if !issueKeyRe.MatchString(s) {
		return "", fmt.Errorf("invalid issue key %q", s)
	}
	return IssueKey(s), nil
}

func (k IssueKey) String() string { return string(k) }

// ExtractIssueKeys returns the distinct issue keys in text, in order of first appearance.
func ExtractIssueKeys(text string) []IssueKey {
	var keys []IssueKey
	seen := map[string]bool{}
	for _, m := range issueKeyScan.FindAllString(text, -1) {
		if !seen[m] {
			seen[m] = true
			keys = append(keys, IssueKey(m))
		}
	}
	return keys
}
