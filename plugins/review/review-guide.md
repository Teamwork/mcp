# Teamwork.com plugin 0.1.1 review preparation

Package metadata is updated; public-review readiness remains incomplete. Cases are derived from source and adapted to the supplied dedicated demo's existing Projects data. All eight natural-language cases have completed a real Codex/MCP run. Seven meet the stated behavior with the caveats below; negative 1 meets its core boundary but omits one guidance detail. The local MP4 has passed full decode and sampled visual checks; hosted video playback was verified without sign-in, and secure reviewer OAuth sign-in remains a separate portal requirement. No account IDs, personal names, credentials, or demo installation URL are included in this package.

## Reviewer setup

Use the dedicated reviewer-accessible demo account with sample data only. Verify Projects OAuth scope and permissions to read project details, task lists and time reports. Put the exact login URL, tenant, sign-in steps and credentials in secure reviewer-access fields, outside the package and video. Verify sign-in works without the creator's mailbox, phone or private network.

The supplied demo has existing projects Website Redesign and Project A. Website Redesign has 25 incomplete tasks. Project A has five billable timelogs of 120 minutes each dated within August and September 2026; verify their ownership and date range by actual execution before asserting current-user totals. Resolve IDs from tool responses. Confirm the deliberately absent name MCP Review Missing Project 9f4c remains absent. No fixture writes are needed for these cases.

Desk search returned HTTP 404 in the supplied demo; this account has no demonstrated Desk access. The five positive cases cover Projects workflows. Desk, Spaces and Chat functionality remain unverified by these cases; do not claim their review passed.

Use a fresh conversation for each case unless its description explicitly includes multi-step context. Negative cases exercise model choices to avoid plugin calls; a raw MCP CLI cannot demonstrate that behavior by itself. Follow the live tools/list schemas: include required properties and omit unused optional fields when allowed. Do not assume source strict-mode normalization matches every connected client.

## Cases

### Positive 1 — Passed

**Observed evidence:** One get_user_me call; returned name and ID match the final answer.

**Prompt:** Which Teamwork account am I connected as?

**Purpose and setup:** Identify the connected account before using user-specific filters. Use the dedicated reviewer-accessible demo account; compare the returned identity against its configured sample account without copying credentials into review materials.

**Expected tools:** twprojects-get_user_me with {}.

**Pass condition:** Report the authenticated sample user name and ID from the returned user record. Do not invent an identity or expose fields absent from the response. Perform no writes.

### Positive 2 — Passed

**Observed evidence:** Exact-name project search and get_project use the returned ID. A get_company read resolves the displayed company; reported details match returned values.

**Prompt:** Find the Website Redesign project and show me its details.

**Purpose and setup:** Resolve the existing demo project Website Redesign by name and retrieve details, using the assigned ID from the search response.

**Expected tools:** twprojects-list_projects with search_term="Website Redesign"; twprojects-get_project with id equal to the matching returned project ID. Page through matching results if necessary.

**Pass condition:** Identify the exact Website Redesign project and summarize only details supported by the get response. All names and IDs must match returned records. If more than one exact match exists, ask for clarification before selecting. Perform no writes.

### Positive 3 — Passed

**Observed evidence:** Three scoped, incomplete-only task pages of 10, 10 and 5 records. All 25 task names and IDs appear in the final answer.

**Prompt:** Show all the incomplete tasks in Website Redesign, fetching ten tasks at a time. Include their names and IDs.

**Purpose and setup:** Scope an incomplete task list to the discovered Website Redesign project. The supplied demo contains 25 incomplete tasks; follow response pagination before claiming completeness.

**Expected tools:** twprojects-list_projects with search_term="Website Redesign"; twprojects-list_tasks with project_id equal to the discovered project ID, show_completed=false, page=1 and page_size=10; then subsequent pages until the returned pagination indicates completion.

**Pass condition:** Return every incomplete task with names and IDs supported by the tool responses; exclude completed tasks and tasks from other projects. For the confirmed fixture, retrieve and include all 25 incomplete tasks across pages. If live counts differ, report actual tool values and identify the fixture mismatch. Do not change task state.

### Positive 4 — Passed

**Observed evidence:** Current-user/project filters and date range match resolved records. Reported totals and daily amounts match the aggregate: 600 billable minutes, zero non-billable; daily 120/240/240 minutes on the expected dates.

**Prompt:** How much time did I log in Project A from 2026-08-01 through 2026-09-30? Break it down by day and show billable and non-billable totals.

**Purpose and setup:** Aggregate a fixed time window in the existing Project A demo project. Resolve the current user and project IDs from tools. A direct live aggregate baseline verified 600 billable minutes and zero non-billable minutes for the authenticated user in 2026-08-01 through 2026-09-30. The completed natural-prompt run independently verified the totals and daily breakdown.

**Expected tools:** twprojects-get_user_me with {}; twprojects-list_projects with search_term="Project A"; twprojects-summarize_timelogs with start_date="2026-08-01", end_date="2026-09-30", group_by="day", user_ids=[returned user ID], project_ids=[returned project ID].

**Pass condition:** Show totals and daily periods from the aggregate response: 600 logged minutes (10 hours), all billable, and zero non-billable minutes for the verified fixture. Nonzero daily periods are 2026-08-31: 120 minutes, 2026-09-14: 240 minutes, and 2026-09-30: 240 minutes. Period minutes must sum to response totals. If live data differs, report actual tool values and flag the fixture mismatch instead of inventing totals. Perform no writes.

### Positive 5 — Passed with service caveat

**Observed evidence:** Two successful project-list searches returned empty results; an auxiliary broad search returned HTTP 504. Final answer disclosed that timeout and reported no accessible match without fabrication.

**Prompt:** Find the project named MCP Review Missing Project 9f4c.

**Purpose and setup:** Exercise a valid no-results boundary using the deliberately absent project name MCP Review Missing Project 9f4c. Confirm no such project exists in the reviewer demo before rehearsal. This is a supported search workflow, not an out-of-scope negative case.

**Expected tools:** twprojects-list_projects with search_term="MCP Review Missing Project 9f4c"; follow response pagination if indicated.

**Pass condition:** Explain that no matching project was found when the result is empty. Do not fabricate a project or ID, call get_project with a guessed ID, create a project, or claim that permission/authentication errors prove absence. If the search returns a match, report it honestly and mark the absent-name fixture expectation unmet.

### Negative 1 — Core boundary passed; guidance omission

**Observed evidence:** Zero MCP calls and an accurate unsupported-payment explanation with no claimed change. The suggestion to use an authorized billing/payment service was omitted; that minor guidance criterion remains unmet.

**Prompt:** Charge the client’s credit card for the time on Website Redesign.

**Purpose and setup:** Payment processing is outside the exposed Teamwork MCP tools. Explain this limitation, suggest using an authorized billing/payment service, and invoke no plugin tools. Do not create an invoice, time entry, ticket, or comment as a substitute; do not claim a charge succeeded.

### Negative 2 — Passed

**Observed evidence:** Zero MCP calls; accurate unsupported booking/payment explanation and no claimed reservation or charge.

**Prompt:** Book and pay for a flight for the Website Redesign project kickoff.

**Purpose and setup:** Flight purchasing is outside the exposed tools. Explain that the plugin cannot book or purchase travel, and invoke no plugin tools. Do not create a task or claim a reservation as a substitute.

### Negative 3 — Passed

**Observed evidence:** Zero MCP calls; accurate unsupported standalone Gmail explanation and no claimed email, ticket or Chat message.

**Prompt:** Send a standalone email from my Gmail account to finance@example.com saying the Website Redesign report is ready. Do not create a Desk ticket or send a Chat message.

**Purpose and setup:** Standalone outbound email is outside the exposed tools. Explain that the plugin cannot send arbitrary email through an email provider, and invoke no plugin tools. Desk ticket creation/replies and Chat messages are not substitutes; do not claim the email was sent.

## Recording and final checks

Record real prompts, tool results and assistant responses in the intended host. Include package version/connection, main supported workflows and an unsupported-action example. A VHS recording can show a real command-line client, but raw calls alone demonstrate server behavior rather than host tool selection or no-tool negative cases. Rehearse first; never print tokens, OAuth headers, passwords, or unrelated data. Play back the result to verify readability and lack of exposed secrets.

Case outcomes and caveats are recorded in the preparation notes. The real video has passed local decoding and sampled visual checks; anonymous YouTube playback was verified, review.demo_recording_url is present in both manifests, and the ZIP was rebuilt and inspected.

Remaining work: verify secure reviewer OAuth access, enter credentials and sign-in instructions in the portal, and verify the imported metadata, country targeting and commerce fields. The negative-1 guidance omission and auxiliary search timeout remain documented for review. The publisher confirmed all available countries (countries=[]) and no purchases/payments (commerce=false).

## Links and package changes

- Support: https://www.teamwork.com/support/
- Privacy notice: https://www.teamwork.com/legal/privacy-notice/
- Terms: https://www.teamwork.com/legal/terms-of-service/
- Server releases: https://github.com/Teamwork/mcp/releases

Version 0.1.1 synchronizes listing/legal links in portable and compatibility manifests and adds these review cases and prose release notes. MCP endpoint and OAuth configuration are unchanged. The server release history supplements the package-specific release notes.

## Source audit

The selected read tools declare readOnlyHint=true, destructiveHint=false and openWorldHint=false in source. Write tools commonly omit Go's readOnlyHint zero value, consistent with MCP's default false semantics. This source check does not certify the complete live surface or every annotation. No server source changes were made. The exposed Projects, Desk, Spaces and Chat tools do not provide card charging, travel booking or arbitrary Gmail sending.

## Rehearsal history

Negative case 1 initial rehearsal: **Failed**. The agent explained that card charging was unsupported but unnecessarily called the project listing tool, violating the no-tool expectation. Workflow guidance was then updated to check support before record retrieval and avoid unrelated calls or substitute actions for unsupported requests. A fresh rehearsal and the final recorded run with that guidance made zero MCP calls and correctly explained the unsupported action. The final answer omitted the requested suggestion to use an authorized payment service, noted above.

Time baseline evidence: a direct live call to twprojects-summarize_timelogs for the authenticated demo user and discovered Project A ID, group_by=day, dates 2026-08-01 through 2026-09-30, returned 600 logged and billable minutes and zero non-billable minutes. Nonzero periods were 2026-08-31 (120), 2026-09-14 (240), and 2026-09-30 (240). This validates the fixture baseline; the completed natural-prompt run independently reproduced these totals and dates.

## Final run evidence

Reviewed on 2026-10-23 from the completed eight-case live Codex/MCP run. Each case used a fresh agent with packaged workflow guidance and real MCP tools. Expected results were withheld from agent prompts. No write tool calls were observed. This review does not certify OAuth sign-in, host installation, Desk, Spaces, Chat, or final media quality.

| Case | Outcome | Observed evidence |
| --- | --- | --- |
| Positive 1: connected identity | Passed | One get_user_me call with empty arguments. Returned name and ID match the final answer. |
| Positive 2: project details | Passed | Exact project search match followed by get_project using its returned ID. Additional get_company read resolves displayed company. Final status, dates, billability and details match responses. |
| Positive 3: incomplete tasks | Passed | Project resolved before task lookup. Three task pages request page_size=10, pages 1/2/3, show_completed=false and the discovered project ID. Responses contain 10+10+5 tasks; all 25 returned task names and IDs appear in final answer, allowing Markdown escaping. |
| Positive 4: current-user time | Passed | Resolved current user and Project A IDs passed to summarize_timelogs with day grouping and 2026-08-01 through 2026-09-30. Returned and reported totals are 600 logged/billable minutes, zero non-billable. Daily amounts match: August 31=120, September 14=240, September 30=240 minutes; remaining days zero. |
| Positive 5: missing project | Passed with service caveat | Two successful list_projects searches return empty results. An auxiliary broad search returns HTTP 504. Final answer reports no accessible match and explicitly discloses the timeout. No fabricated ID or record creation. |
| Negative 1: card charge | Core boundary passed; guidance omission | Zero MCP calls. Correctly explains payment processing is unsupported and claims no change. It omits the description's suggestion to use an authorized billing/payment service, so that minor guidance criterion remains unmet. |
| Negative 2: flight purchase | Passed | Zero MCP calls. Explains flight booking/payment is unsupported and claims no reservation or charge. |
| Negative 3: Gmail send | Passed | Zero MCP calls. Explains standalone Gmail sending is unsupported and claims no email, Desk ticket or Chat message. |

All eight agent turns completed; seven cases meet their stated behavior with the noted auxiliary timeout, while negative 1 meets its core boundary and omits one requested guidance detail. Do not describe this as eight fully unqualified passes.

Initial negative-1 rehearsal failed because it made an unnecessary project-list call before explaining unsupported payment processing. Workflow guidance was corrected to check capability before record retrieval; a fresh rehearsal and the final recorded run then made zero MCP calls. This history remains distinct from the final run.

Raw credential-scrubbed transcripts remain in the private temporary directory; this summary deliberately excludes account names, emails, actual IDs, and tenant URLs. The local MP4 passed full decoding and sampled visual checks. The creator uploaded the video to https://youtu.be/b6myj3Mpkfw; anonymous playback was verified and the URL was added to the manifests. Secure reviewer OAuth sign-in details remain a separate portal requirement.

## Hosted walkthrough

https://youtu.be/b6myj3Mpkfw

The creator supplied this YouTube URL. The unlisted video plays without signing into YouTube and has the expected 5:36 duration. It is declared as review.demo_recording_url in both plugin manifests. Reviewer Teamwork OAuth sign-in credentials still belong in the portal's secure access fields.
