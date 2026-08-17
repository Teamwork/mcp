// next-version computes the next semantic version for a release from the
// changes merged since the last tag, so the bump is derived from what shipped
// instead of chosen by hand.
//
// Usage:
//
//	go run ./cmd/next-version                 # report the next version
//	go run ./cmd/next-version -bump=minor     # force a bump level
//	go run ./cmd/next-version -from=v1.28.0   # diff from an explicit tag
//	go run ./cmd/next-version -no-pr-lookup   # classify commit subjects only
//
// Each change is classified by the prefix of its pull-request title, falling
// back to the commit subject when a commit has no pull request (rebase-merged
// branches put commits like "Fix comment" on main; they inherit their pull
// request's prefix instead of counting on their own):
//
//	Feature: / Feat:                       -> minor
//	Fix: / Enhancement: / Chore: / Docs:   -> patch
//	any prefix with ! , or BREAKING CHANGE -> major
//
// A change with no recognisable prefix counts as a patch and is reported as
// unclassified: v1.27.4, v1.28.1 and v1.28.3 all shipped a feature under a
// patch tag, and every one of them was an unprefixed title.
//
// Under GitHub Actions it appends version, previous_tag, bump and unclassified
// to $GITHUB_OUTPUT, and a markdown table to $GITHUB_STEP_SUMMARY.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	bumpFlag := flag.String("bump", "auto", "bump to apply: auto, patch, minor or major")
	from := flag.String("from", "", "tag to diff from (default: highest semver tag reachable from -to)")
	to := flag.String("to", "HEAD", "revision to release")
	repo := flag.String("repo", os.Getenv("GITHUB_REPOSITORY"), "owner/name used for pull-request lookups")
	noPRLookup := flag.Bool("no-pr-lookup", false, "classify commit subjects only, without resolving pull requests")
	outputPath := flag.String("output", os.Getenv("GITHUB_OUTPUT"), "file to append key=value outputs to")
	summaryPath := flag.String("summary", os.Getenv("GITHUB_STEP_SUMMARY"), "file to append a markdown summary to")
	flag.Parse()

	forced, err := parseBump(*bumpFlag)
	if err != nil {
		fail("%v", err)
	}

	rep, err := buildReport(*from, *to, *repo, forced, !*noPRLookup)
	if err != nil {
		fail("%v", err)
	}

	fmt.Print(rep.text())
	for _, c := range rep.Changes {
		if !c.Classified {
			fmt.Printf("::warning title=Unclassified change::%s %q has no recognisable prefix; counted as patch\n", c.Ref, c.Title)
		}
	}

	if *outputPath != "" {
		if err := appendFile(*outputPath, rep.outputs()); err != nil {
			fail("write outputs: %v", err)
		}
	}
	if *summaryPath != "" {
		if err := appendFile(*summaryPath, rep.markdown()); err != nil {
			fail("write summary: %v", err)
		}
	}
}

// bumpLevel is a semantic-version component, ordered so that the highest level
// across a set of changes is their max.
type bumpLevel int

const (
	bumpNone bumpLevel = iota
	bumpPatch
	bumpMinor
	bumpMajor
)

func (b bumpLevel) String() string {
	switch b {
	case bumpMajor:
		return "major"
	case bumpMinor:
		return "minor"
	case bumpPatch:
		return "patch"
	default:
		return "none"
	}
}

// parseBump reads the -bump flag. "auto" is bumpNone: no forced level, so the
// changes decide.
func parseBump(s string) (bumpLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return bumpNone, nil
	case "patch":
		return bumpPatch, nil
	case "minor":
		return bumpMinor, nil
	case "major":
		return bumpMajor, nil
	}
	return bumpNone, fmt.Errorf("unknown bump %q: want auto, patch, minor or major", s)
}

type version struct {
	major, minor, patch int
}

func (v version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) less(o version) bool {
	switch {
	case v.major != o.major:
		return v.major < o.major
	case v.minor != o.minor:
		return v.minor < o.minor
	default:
		return v.patch < o.patch
	}
}

func (v version) next(l bumpLevel) version {
	switch l {
	case bumpMajor:
		return version{v.major + 1, 0, 0}
	case bumpMinor:
		return version{v.major, v.minor + 1, 0}
	case bumpPatch:
		return version{v.major, v.minor, v.patch + 1}
	default:
		return v
	}
}

// parseVersion accepts the plain vMAJOR.MINOR.PATCH this repo tags with.
// Anything else — a pre-release, a date, a name — is not a version, so it is
// ignored when picking the tag to diff from.
func parseVersion(tag string) (version, bool) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(tag), "v"), ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return version{}, false
		}
		switch i {
		case 0:
			v.major = n
		case 1:
			v.minor = n
		case 2:
			v.patch = n
		}
	}
	return v, true
}

// prefixPattern matches the conventional-commit style prefix CONTRIBUTING.md
// asks for: a word, an optional (scope), an optional ! and a colon.
var prefixPattern = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z-]*)\s*(?:\([^)]*\))?\s*(!?)\s*:`)

var breakingPattern = regexp.MustCompile(`(?mi)^BREAKING[ -]CHANGE:`)

// bumpByPrefix maps every prefix CONTRIBUTING.md lists, plus the
// conventional-commit spellings, to the component it moves. Enhancement is a
// patch: every Enhancement-only release so far (v1.25.4, v1.27.5, v1.28.4)
// shipped as one.
var bumpByPrefix = map[string]bumpLevel{
	"feature": bumpMinor,
	"feat":    bumpMinor,

	"bugfix":      bumpPatch,
	"build":       bumpPatch,
	"chore":       bumpPatch,
	"ci":          bumpPatch,
	"doc":         bumpPatch,
	"docs":        bumpPatch,
	"enhancement": bumpPatch,
	"fix":         bumpPatch,
	"perf":        bumpPatch,
	"performance": bumpPatch,
	"refactor":    bumpPatch,
	"revert":      bumpPatch,
	"style":       bumpPatch,
	"test":        bumpPatch,
	"tests":       bumpPatch,
}

// releaseChorePatterns match the commits the release process itself lands on
// main. The Homebrew formula pull request for one tag merges after that tag, so
// it always falls inside the next release's range; counting it would report an
// unclassified change on every release.
var releaseChorePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^tw-mcp v\d+\.\d+\.\d+\b`),
	regexp.MustCompile(`(?i)^chore: update Homebrew formula for v\d+\.\d+\.\d+\b`),
}

func isReleaseChore(title string) bool {
	title = strings.TrimSpace(title)
	for _, p := range releaseChorePatterns {
		if p.MatchString(title) {
			return true
		}
	}
	return false
}

// classify reads the bump out of a title. ok is false when no known prefix was
// found: the level is still patch, but the caller reports it.
func classify(title, body string) (prefix string, level bumpLevel, ok bool) {
	breaking := breakingPattern.MatchString(body)

	match := prefixPattern.FindStringSubmatch(title)
	if match == nil {
		if breaking {
			return "", bumpMajor, true
		}
		return "", bumpPatch, false
	}

	prefix = strings.ToLower(match[1])
	if match[2] == "!" || breaking {
		return prefix, bumpMajor, true
	}
	if level, ok = bumpByPrefix[prefix]; !ok {
		return prefix, bumpPatch, false
	}
	return prefix, level, true
}

// change is one releasable unit: a merged pull request, or a commit that has
// none.
type change struct {
	Ref        string // "#463", or the short sha for a commit with no pull request
	Title      string
	Prefix     string
	Level      bumpLevel
	Classified bool
}

type report struct {
	Previous version
	Next     version
	Level    bumpLevel
	Forced   bool
	Changes  []change
	Notes    []string
}

func (r report) unclassified() []change {
	var out []change
	for _, c := range r.Changes {
		if !c.Classified {
			out = append(out, c)
		}
	}
	return out
}

func buildReport(from, to, repo string, forced bumpLevel, lookupPRs bool) (report, error) {
	var rep report

	if from == "" {
		latest, err := latestTag(to)
		if err != nil {
			return rep, err
		}
		from = latest
		if from == "" {
			rep.Notes = append(rep.Notes, "no semver tag found; counting every commit from the start of history")
		}
	}
	if from != "" {
		prev, ok := parseVersion(from)
		if !ok {
			return rep, fmt.Errorf("previous tag %q is not vMAJOR.MINOR.PATCH", from)
		}
		rep.Previous = prev
	}

	commits, err := commitsIn(from, to)
	if err != nil {
		return rep, err
	}
	if len(commits) == 0 {
		return rep, fmt.Errorf("no commits between %s and %s", displayTag(from), to)
	}

	var resolve resolver
	if lookupPRs {
		client, err := newGHClient(repo)
		if err != nil {
			rep.Notes = append(rep.Notes, fmt.Sprintf("classifying commit subjects only: %v", err))
		} else {
			resolve = client.pullRequestFor
		}
	}

	rep.Changes, rep.Notes = collectChanges(commits, resolve, rep.Notes)
	if len(rep.Changes) == 0 {
		return rep, fmt.Errorf("nothing to release: every change since %s is a release chore", displayTag(from))
	}

	rep.Level = highestLevel(rep.Changes)
	if forced != bumpNone {
		rep.Forced = true
		rep.Level = forced
	}
	if rep.Level == bumpNone {
		return rep, fmt.Errorf("no change since %s moves the version", displayTag(from))
	}

	rep.Next = rep.Previous.next(rep.Level)
	if exists, err := tagExists(rep.Next.String()); err != nil {
		return rep, err
	} else if exists {
		return rep, fmt.Errorf("tag %s already exists; pass -bump to pick another level", rep.Next)
	}
	return rep, nil
}

// highestLevel is the bump a set of changes earns: the largest any single
// change asks for.
func highestLevel(changes []change) bumpLevel {
	level := bumpNone
	for _, c := range changes {
		level = max(level, c.Level)
	}
	return level
}

// resolver maps a commit to the merged pull request it landed through, or nil
// when it has none.
type resolver func(sha string) (*pullRequest, error)

// collectChanges folds commits into changes, one per merged pull request. A
// pull request already seen is skipped, so a rebase-merged branch counts once
// under its own title.
func collectChanges(commits []commit, resolve resolver, notes []string) ([]change, []string) {
	var (
		changes  []change
		seenPRs  = map[int]bool{}
		lookupOK = resolve != nil
	)

	for _, c := range commits {
		title, ref, body := c.Subject, shortSHA(c.SHA), c.Body

		if lookupOK {
			pr, err := resolve(c.SHA)
			switch {
			case err != nil:
				notes = append(notes, fmt.Sprintf("pull-request lookup failed for %s, using its subject: %v", shortSHA(c.SHA), err))
				lookupOK = false
			case pr != nil && seenPRs[pr.Number]:
				continue
			case pr != nil:
				seenPRs[pr.Number] = true
				title, ref, body = pr.Title, fmt.Sprintf("#%d", pr.Number), pr.Body
			}
		}

		if isReleaseChore(title) {
			continue
		}

		prefix, level, ok := classify(title, body)
		changes = append(changes, change{
			Ref:        ref,
			Title:      title,
			Prefix:     prefix,
			Level:      level,
			Classified: ok,
		})
	}
	return changes, notes
}

func (r report) outputs() string {
	return fmt.Sprintf("version=%s\nprevious_tag=%s\nbump=%s\nunclassified=%d\n",
		r.Next, r.Previous, r.Level, len(r.unclassified()))
}

func (r report) text() string {
	var b strings.Builder

	forced := ""
	if r.Forced {
		forced = ", forced"
	}
	fmt.Fprintf(&b, "%s -> %s (%s%s, from %d change(s))\n\n", r.Previous, r.Next, r.Level, forced, len(r.Changes))

	for _, c := range r.Changes {
		mark := " "
		if !c.Classified {
			mark = "?"
		}
		fmt.Fprintf(&b, "  %s %-5s %-8s %s\n", mark, c.Level, c.Ref, c.Title)
	}

	if unclassified := r.unclassified(); len(unclassified) > 0 {
		fmt.Fprintf(&b, "\nWARNING: %d change(s) marked ? carry no recognisable prefix and\n", len(unclassified))
		fmt.Fprintf(&b, "counted as patch. Check none is a feature, and pass -bump=minor if one is.\n")
	}
	for _, note := range r.Notes {
		fmt.Fprintf(&b, "\nNote: %s\n", note)
	}
	return b.String()
}

func (r report) markdown() string {
	var b strings.Builder

	forced := ""
	if r.Forced {
		forced = ", forced"
	}
	fmt.Fprintf(&b, "## Release %s\n\n", r.Next)
	fmt.Fprintf(&b, "`%s` → **`%s`** — %s bump%s, from %d change(s).\n\n", r.Previous, r.Next, r.Level, forced, len(r.Changes))

	if unclassified := r.unclassified(); len(unclassified) > 0 {
		fmt.Fprintf(&b, "> [!WARNING]\n")
		fmt.Fprintf(&b, "> %d change(s) carry no recognisable prefix and counted as a patch. If any of\n", len(unclassified))
		fmt.Fprintf(&b, "> them is a feature, this release is under-versioned: delete the tag and re-run\n")
		fmt.Fprintf(&b, "> with `bump: minor`.\n\n")
	}

	fmt.Fprintf(&b, "| Bump | Change | Title |\n| --- | --- | --- |\n")
	for _, c := range r.Changes {
		bump := c.Level.String()
		if !c.Classified {
			bump = "⚠️ " + bump
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", bump, c.Ref, escapePipes(c.Title))
	}

	for _, note := range r.Notes {
		fmt.Fprintf(&b, "\n_%s_\n", note)
	}
	return b.String()
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

type commit struct {
	SHA     string
	Subject string
	Body    string
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func displayTag(tag string) string {
	if tag == "" {
		return "the start of history"
	}
	return tag
}

// latestTag returns the highest semver tag reachable from rev, which is the one
// the last release shipped — not the most recent by date, and not one sitting
// on a branch that never merged.
func latestTag(rev string) (string, error) {
	out, err := git("tag", "--merged", rev, "--list", "v*")
	if err != nil {
		return "", err
	}
	return highestVersionTag(strings.Split(out, "\n")), nil
}

func highestVersionTag(tags []string) string {
	var (
		best     version
		bestName string
	)
	for _, line := range tags {
		tag := strings.TrimSpace(line)
		v, ok := parseVersion(tag)
		if !ok {
			continue
		}
		if bestName == "" || best.less(v) {
			best, bestName = v, tag
		}
	}
	return bestName
}

func tagExists(tag string) (bool, error) {
	if _, err := git("rev-parse", "-q", "--verify", "refs/tags/"+tag); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// commitsIn lists from..to oldest first, so the report reads like a changelog.
func commitsIn(from, to string) ([]commit, error) {
	rev := to
	if from != "" {
		rev = from + ".." + to
	}

	// \x1f separates fields and \x1e records, so a subject or body containing a
	// newline cannot be mistaken for the next commit.
	out, err := git("log", "--no-merges", "--reverse", "--format=%H%x1f%s%x1f%b%x1e", rev)
	if err != nil {
		return nil, err
	}

	var commits []commit
	for record := range strings.SplitSeq(out, "\x1e") {
		record = strings.Trim(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.Split(record, "\x1f")
		if len(fields) < 3 {
			return nil, fmt.Errorf("unexpected git log record %q", record)
		}
		commits = append(commits, commit{SHA: fields[0], Subject: fields[1], Body: fields[2]})
	}
	return commits, nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

type pullRequest struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	MergedAt string `json:"merged_at"`
}

type ghClient struct {
	repo   string
	token  string
	client *http.Client
}

// newGHClient needs a repo and a token. Both are present in Actions; locally
// the caller falls back to commit subjects, which is why the missing pieces are
// named in the error.
func newGHClient(repo string) (*ghClient, error) {
	if repo == "" {
		remote, err := git("remote", "get-url", "origin")
		if err != nil {
			return nil, fmt.Errorf("no -repo and no origin remote to derive it from")
		}
		repo = repoFromRemote(remote)
		if repo == "" {
			return nil, fmt.Errorf("cannot derive owner/name from origin %q", strings.TrimSpace(remote))
		}
	}

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("no GH_TOKEN or GITHUB_TOKEN for pull-request lookups")
	}

	return &ghClient{repo: repo, token: token, client: &http.Client{Timeout: 30 * time.Second}}, nil
}

// repoFromRemote pulls owner/name out of either remote form git uses.
func repoFromRemote(remote string) string {
	remote = strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	if _, after, ok := strings.Cut(remote, "github.com"); ok {
		return strings.Trim(after, ":/")
	}
	return ""
}

// pullRequestFor returns the merged pull request a commit landed through, or
// nil when it has none.
func (c *ghClient) pullRequestFor(sha string) (*pullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/pulls?per_page=100", c.repo, sha)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	var prs []pullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, fmt.Errorf("decode pull requests for %s: %w", shortSHA(sha), err)
	}
	return pickPullRequest(prs), nil
}

// pickPullRequest prefers a merged pull request: a commit can also be
// associated with open ones that merely contain it, and their titles say
// nothing about what shipped.
func pickPullRequest(prs []pullRequest) *pullRequest {
	for i := range prs {
		if prs[i].MergedAt != "" {
			return &prs[i]
		}
	}
	return nil
}

func appendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "next-version: "+format+"\n", args...)
	os.Exit(1)
}
