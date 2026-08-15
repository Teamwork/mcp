# Teamwork MCP — Tool Reference

Auto-generated from the registered toolsets by `cmd/docs-gen`. Do not edit by hand — run `go run ./cmd/docs-gen` to regenerate.

This reflects the tools a client actually receives from the shipped servers (`cmd/mcp-http`, `cmd/mcp-stdio`) with writes enabled. **Delete operations are intentionally omitted**: they exist in the codebase but are gated behind an `allowDelete` flag that no shipped server enables, so no client can invoke them. Running a server with `-read-only` removes the write tools, leaving the Get/List operations plus any read-only entries under "Other actions" (e.g. `search`, `summarize_timelogs`, `users_workload`).

## Projects

### Content — `twprojects-content`

Comments, notebooks, milestones, tags, and activity feeds in Teamwork.com.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Activity | — | — | ✓ | — |
| Comment | ✓ | ✓ | ✓ | ✓ |
| Milestone | ✓ | ✓ | ✓ | ✓ |
| Notebook | ✓ | ✓ | ✓ | ✓ |
| Tag | ✓ | ✓ | ✓ | ✓ |
| Message | ✓ | ✓ | ✓ | ✓ |
| Message Reply | ✓ | ✓ | ✓ | ✓ |
| Link | ✓ | ✓ | ✓ | ✓ |

**Other actions:** `search`

### People — `twprojects-people`

Users, companies, teams, skills, job roles, and workload management in Teamwork.com.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Company | ✓ | ✓ | ✓ | ✓ |
| Industry | — | — | ✓ | — |
| Job Role | ✓ | ✓ | ✓ | ✓ |
| Skill | ✓ | ✓ | ✓ | ✓ |
| Team | ✓ | ✓ | ✓ | ✓ |
| User | ✓ | ✓ | ✓ | ✓ |
| Current User (me) | — | ✓ | — | — |

**Other actions:** `users_workload`

### Projects — `twprojects-projects`

Project, category, template, member, custom field, and custom item (user-defined entity types like Contracts, Leads, Deals) management in Teamwork.com.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Project Category | ✓ | ✓ | ✓ | ✓ |
| Project | ✓ | ✓ | ✓ | ✓ |
| Project Template | ✓ | — | ✓ | — |
| Custom Field | ✓ | ✓ | ✓ | ✓ |
| Custom Field Value | ✓ | ✓ | ✓ | ✓ |
| Custom Item | ✓ | ✓ | ✓ | ✓ |
| Custom Item Field | ✓ | ✓ | ✓ | ✓ |
| Custom Item Record | ✓ | ✓ | ✓ | ✓ |
| File | ✓ | — | — | — |

**Other actions:** `add_project_member`, `clone_project`

### Tasks — `twprojects-tasks`

Task, tasklist, and workflow management in Teamwork.com.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Task | ✓ | ✓ | ✓ | ✓ |
| Tasklist | ✓ | ✓ | ✓ | ✓ |
| Workflow | ✓ | ✓ | ✓ | ✓ |
| Workflow Stage | ✓ | ✓ | ✓ | ✓ |

**Other actions:** `complete_task`, `link_project_to_workflow`, `move_task_to_workflow_stage`, `move_tasks`

### Time — `twprojects-time`

Time tracking via timelogs, timers, calendars with time blocking, and budget reporting in Teamwork.com.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Calendar Event | — | — | ✓ | — |
| Calendar | — | — | ✓ | — |
| Project Budget | — | — | ✓ | — |
| Tasklist Budget | — | — | ✓ | — |
| Timelog | ✓ | ✓ | ✓ | ✓ |
| Timer | ✓ | ✓ | ✓ | ✓ |

**Other actions:** `complete_timer`, `pause_timer`, `resume_timer`, `summarize_timelogs`

## Desk

### Admin — `twdesk-admin`

Inbox configuration: priorities, statuses, types, and tags in Teamwork Desk.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Priority | ✓ | ✓ | ✓ | ✓ |
| Status | ✓ | ✓ | ✓ | ✓ |
| Tag | ✓ | ✓ | ✓ | ✓ |
| Ticket Type | ✓ | ✓ | ✓ | ✓ |

### Customers — `twdesk-customers`

Companies, customers, and user management in Teamwork Desk.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Company | ✓ | ✓ | ✓ | ✓ |
| Customer | ✓ | ✓ | ✓ | ✓ |
| User | — | ✓ | ✓ | — |

### Helpdocs — `twdesk-helpdocs`

Help doc articles and sites in Teamwork Desk.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Helpdoc Article | ✓ | ✓ | — | ✓ |
| Helpdoc Site | — | ✓ | ✓ | — |

**Other actions:** `search_helpdoc_articles`

### Tickets — `twdesk-tickets`

Tickets, messages, files, and inboxes in Teamwork Desk.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Inbox | — | ✓ | ✓ | — |
| Ticket | ✓ | ✓ | — | ✓ |
| File | ✓ | — | — | — |

**Other actions:** `link_task_to_ticket`, `reply_ticket`, `search_tickets`, `unlink_task_from_ticket`

## Spaces

### Content — `twspaces-content`

Comments, tags, categories, and search in Teamwork Spaces.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Comment | ✓ | ✓ | ✓ | ✓ |
| Tag | ✓ | ✓ | ✓ | ✓ |
| Category | ✓ | ✓ | ✓ | ✓ |

**Other actions:** `search`

### Pages — `twspaces-pages`

Page CRUD, homepage, and duplication in Teamwork Spaces.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Page | ✓ | ✓ | ✓ | ✓ |
| Homepage | — | ✓ | — | — |

**Other actions:** `duplicate_page`

### Spaces — `twspaces-spaces`

Space CRUD and collaborators in Teamwork Spaces.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Space | ✓ | ✓ | ✓ | ✓ |
| Space Collaborator | — | — | ✓ | — |

## Chat

### Chat — `twchat-chat`

Read conversations, messages, and people, and send messages in Teamwork Chat.

| Resource | Create | Get | List | Update |
|---|---|---|---|---|
| Current User | — | ✓ | — | — |
| Conversation | — | ✓ | ✓ | — |
| Message | — | — | ✓ | — |
| People | — | — | ✓ | — |

**Other actions:** `get_or_create_dm`, `send_dm`, `send_message`
