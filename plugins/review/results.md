# Teamwork.com 0.1.1 live review results

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

Raw credential-scrubbed transcripts remain in the private temporary directory; this summary deliberately excludes account names, emails, actual IDs, and tenant URLs. The local MP4 was encoded from 5,043 matched, contiguous real VHS frame pairs: H.264, 1600 by 1000 pixels, 336.2 seconds. Full decoding completed without errors, and beginning, middle and final video samples were visually inspected for readable interactions and completion evidence. The creator uploaded the recording to YouTube: https://youtu.be/b6myj3Mpkfw. Anonymous playback was verified and the URL was added to the plugin manifests. Secure reviewer OAuth sign-in details remain a separate portal requirement.
