# next-version

Computes the next release version from the changes since the last tag, so the
bump is derived from what shipped rather than picked by hand.

```bash
go run ./cmd/next-version                 # report the next version
go run ./cmd/next-version -bump=minor     # force a bump level
go run ./cmd/next-version -from=v1.28.0   # diff from an explicit tag
go run ./cmd/next-version -no-pr-lookup   # classify commit subjects only
```

## How a change is classified

Each commit since the last tag is resolved to the pull request it merged
through, and the bump comes from that pull request's title. All commits of one
pull request count once, so a rebase-merged branch is judged by its title, not
by commits like `Fix comment`. A commit with no pull request falls back to its
own subject.

| Title | Bump |
| --- | --- |
| `Feature:`, `Feat:` | minor |
| `Fix:`, `Enhancement:`, `Chore:`, `Docs:`, `Test:`, `Refactor:`, `Perf:`, `Build:`, `CI:`, `Style:`, `Revert:` | patch |
| any prefix with `!` (`Feature!:`), or a `BREAKING CHANGE:` body | major |
| anything else | patch, **reported as unclassified** |

The release takes the highest bump any single change asks for. Prefixes are
case-insensitive and may carry a scope: `Chore(deps):` is a patch.

`Enhancement:` is a patch because every Enhancement-only release so far shipped
as one (v1.25.4, v1.27.5, v1.28.4).

## Unclassified changes

A title with no known prefix counts as a patch and is listed as unclassified,
in the terminal and as a warning in the workflow summary. That is not a
formality: v1.27.4, v1.28.1 and v1.28.3 each shipped a feature under a patch
tag, and every one of those features had an unprefixed title. **Read the
unclassified list before releasing** — if one of them is a feature, re-run with
`bump: minor`.

Commits the release process lands on `main` itself (the Homebrew formula
update, which merges after the tag it belongs to) are skipped, so they never
show up as unclassified.

## Where it runs

`.github/workflows/release.yaml` calls it on the `workflow_dispatch` path,
where it writes `version`, `previous_tag`, `bump` and `unclassified` to
`$GITHUB_OUTPUT` and a per-change table to the run summary. The workflow then
creates that tag and releases it.

Use `-bump` to override the computation, and the workflow's `dry_run` input to
see the version and the table without tagging anything.

Locally the pull-request lookup needs `GH_TOKEN` or `GITHUB_TOKEN`
(`GH_TOKEN=$(gh auth token)`); without one it falls back to commit subjects and
says so.
