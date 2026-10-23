# Record the Teamwork.com review cases

This kit runs the eight natural-language cases in `../teamwork/plugin.json` through real, fresh Codex CLI agents connected to the same hosted Teamwork.com MCP endpoint. It loads the packaged workflow skill and records actual agent messages, MCP calls and results. Expected tools and outcomes are displayed for the human reviewer after execution; they are not provided to the agent. Negative cases receive the same available tools as positive cases.

**Status:** Python syntax and VHS tape syntax have been checked. Local preflight alone does not execute or pass any case. A successful process prints `EXECUTION COMPLETE / MANUAL VERIFICATION REQUIRED`; compare the actual calls, arguments and answers with the manifest before recording Passed or Failed in review notes. Empty results do not satisfy cases expecting existing records. The fifth positive case intentionally expects an empty project search. Denied access, missing required fixtures and unavailable required tools do not satisfy a positive case.

## Prerequisites

- `python3`, [Codex CLI](https://developers.openai.com/codex/cli/), [VHS](https://github.com/charmbracelet/vhs), `ffmpeg` and `ttyd` on PATH.
- Codex already signed in with an account allowed to run the CLI. The runner uses the existing CLI authentication without reading, copying or printing auth files. It ignores global configuration, does not select a model, and does not change global settings.
- A dedicated Teamwork demo account with synthetic data, permission for the manifest's Projects tools, and its bearer token supplied privately through `TW_MCP_BEARER_TOKEN`. Never put credentials in a tape, shell command argument, source file, screenshot or public recording.
- The manifest's fixtures: `Website Redesign` with its incomplete tasks (including enough tasks to demonstrate pagination), `Project A` with the connected user's time records during **2026-08-01 through 2026-09-30**, and no project named `MCP Review Missing Project 9f4c`. Read the manifest descriptions for exact setup and argument checks. Maintain the dated fixture for reviewer repeatability.

Use a token/account restricted to reading wherever supported. Positive cases are read-only, and the runner adds a no-mutations instruction for those cases. Negative cases receive no extra read-only instruction: the agent must identify the unsupported capability from the actual tools and packaged guidance. For every case the runner configures write-tool approvals to be refused; all available tool schemas remain discoverable. The shell sandbox does not by itself enforce restrictions on remote MCP writes. These controls depend on accurate MCP read-only annotations and account permissions.

## Run and record

From this directory, inspect the cases first:

```sh
python3 demo.py --list
```

Set `TW_MCP_BEARER_TOKEN` privately in the parent terminal **before** starting VHS. Confirm the dedicated account and synthetic data, then run:

```sh
export TEAMWORK_DEMO_CONFIRMED=1
python3 demo.py --preflight
vhs validate demo.tape
vhs demo.tape
# If VHS 0.12.0 prints Creating but produces no video:
python3 encode.py
```

The tape records `python3 demo.py` and waits for real completion or a blocked preflight by matching the visible screen, so a returned shell prompt cannot hide the completion marker. Each case has a ten-minute execution limit; the tape allows up to 90 minutes overall. Outputs are `teamwork-review.mp4` and `teamwork-review.webm` in this directory. The tape also preserves both real PNG streams in `teamwork-review-frames/`. VHS 0.12.0 has a confirmed video-export issue: it passes a canceled context to its encoder and can exit successfully without producing videos. If that occurs, run `python3 encode.py`. The helper verifies contiguous, matching, complete frame streams, encodes with the tape's settings, and checks both output files with ffprobe. If you change the tape's dimensions, frame rate, padding or theme, update the matching constants in the helper. Delete or move an old frames directory before recording again; VHS cannot rename onto an existing nonempty directory. Do not publish a recording containing `BLOCKED`, incomplete cases, failed expectations or sensitive data as successful evidence.

For a single case or a different package:

```sh
python3 demo.py --case positive-1 --manifest /absolute/path/to/teamwork/plugin.json
```

The default manifest can also be overridden with `TEAMWORK_REVIEW_MANIFEST`, useful when recording a staged package without changing the tape. `--timeout` controls the per-case limit in seconds. `--pause` controls the short reading pause after agent messages and each case (three seconds by default).

The runner stores complete credential-scrubbed event transcripts and stderr in a new private temporary directory outside the repository. Directory permissions are 700; files are 600. Terminal results are shortened for readability; use the transcripts to inspect complete outputs. Tool results may still contain demo account data. Review all footage and transcripts before sharing; automatic redaction cannot guarantee removal of every possible secret. Never commit recordings or transcripts by accident. The included `.gitignore` excludes MP4, WebM, preserved frames and Python cache files.

## Complete submission evidence

This CLI recording demonstrates live tool integration and agent tool selection with packaged workflow guidance. It does not install the plugin or demonstrate the host's plugin UI, icons, or OAuth sign-in. A supplementary capture of the packaged plugin in its intended host preview/development installation, including connection and a representative workflow, is recommended to demonstrate the submitted version and authentication flow. A CLI recording alone does not guarantee acceptance of the demo evidence. Desk access is outside this suite and remains unverified after an earlier HTTP 404 response.

Review the actual interactions against all five positive and three negative case criteria. Record truthful Passed, Failed, Blocked or Not run results with supporting evidence in preparation notes. Host the finished video at a reviewer-accessible URL and add the verified actual URL to `extensions.com.openai.review.demo_recording_url`; a local file or tape is not that URL. Enter reviewer credentials and login instructions only in the submission portal's secure review-access fields.
