package twprojects

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// TaskSkillsAndRolesPrompt returns the prompt that helps the LLM to identify
// all skills and job roles of a task.
func TaskSkillsAndRolesPrompt(engine *twapi.Engine) toolsets.ServerPrompt {
	return toolsets.ServerPrompt{
		Prompt: &mcp.Prompt{
			Name:  "twprojects_task_skills_and_roles",
			Title: "Teamwork.com Task Skills and Job Roles Analysis",
			Description: "Analyze the details of a task in Teamwork.com and suggest the most suitable skills and job roles " +
				"that align with the task requirements and context within the project.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:  "task_id",
					Title: "Task ID",
					Description: "The ID of the task to analyse. You can identify the desire task by using the " +
						string(MethodTaskList) + " method or in the Teamwork.com website.",
					Required: true,
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			if request.Params.Arguments == nil {
				return nil, fmt.Errorf("arguments are required")
			}

			taskIDStr := request.Params.Arguments["task_id"]
			taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid task ID format: %w", err)
			}
			if taskID <= 0 {
				return nil, fmt.Errorf("task ID must be a positive integer")
			}

			taskResponse, err := projects.TaskGet(ctx, engine, projects.NewTaskGetRequest(taskID))
			if err != nil {
				return nil, fmt.Errorf("failed to get task: %w", err)
			}

			tasklistResponse, err := projects.TasklistGet(ctx, engine,
				projects.NewTasklistGetRequest(taskResponse.Task.Tasklist.ID))
			if err != nil {
				return nil, fmt.Errorf("failed to get tasklist: %w", err)
			}

			projectResponse, err := projects.ProjectGet(ctx, engine,
				projects.NewProjectGetRequest(tasklistResponse.Tasklist.Project.ID))
			if err != nil {
				return nil, fmt.Errorf("failed to get project: %w", err)
			}

			skillsNext, err := twapi.Iterate[projects.SkillListRequest, *projects.SkillListResponse](
				ctx,
				engine,
				projects.NewSkillListRequest(),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to build skills iterator: %w", err)
			}

			var skills []string
			for {
				skillsResponse, hasSkillsNext, err := skillsNext()
				if err != nil {
					return nil, fmt.Errorf("failed to list skills: %w", err)
				}
				if skillsResponse == nil {
					break
				}
				for _, skill := range skillsResponse.Skills {
					skills = append(skills, skill.Name)
				}
				if !hasSkillsNext {
					break
				}
			}

			jobRolesNext, err := twapi.Iterate[projects.JobRoleListRequest, *projects.JobRoleListResponse](
				ctx,
				engine,
				projects.NewJobRoleListRequest(),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to build job roles iterator: %w", err)
			}

			var jobRoles []string
			for {
				jobRolesResponse, hasJobRolesNext, err := jobRolesNext()
				if err != nil {
					return nil, fmt.Errorf("failed to list job roles: %w", err)
				}
				if jobRolesResponse == nil {
					break
				}
				for _, jobRole := range jobRolesResponse.JobRoles {
					jobRoles = append(jobRoles, jobRole.Name)
				}
				if !hasJobRolesNext {
					break
				}
			}

			if len(skills) == 0 && len(jobRoles) == 0 {
				return nil, fmt.Errorf("no skills or job roles found in the organization")
			}

			return &mcp.GetPromptResult{
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: taskSkillsAndRolesSystemPrompt,
						},
					},
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(taskSkillsAndRolesUserPrompt,
								func() string {
									if len(skills) == 0 {
										return "No skills available in the organization."
									}
									return strings.Join(skills, ",")
								}(),
								func() string {
									if len(jobRoles) == 0 {
										return "No job roles available in the organization."
									}
									return strings.Join(jobRoles, ",")
								}(),
								taskResponse.Task.Name,
								func() string {
									if taskResponse.Task.Description == nil {
										return ""
									}
									return *taskResponse.Task.Description
								}(),
								tasklistResponse.Tasklist.Name,
								projectResponse.Project.Name,
								func() string {
									if projectResponse.Project.Description == nil {
										return ""
									}
									return *projectResponse.Project.Description
								}(),
							),
						},
					},
				},
			}, nil
		},
	}
}

const taskSkillsAndRolesSystemPrompt = `
You are a project manager expert using the Teamwork.com platform. Your objective is to analyse the task details and
identify what skills and job roles can have better chances to work on it. The chosen skills and/or job roles should
align with the task requirements and context within the project. Only provide skills and job roles that are relevant to
the task and exist in the organization.

Please send back a JSON object with the skills and job role IDs. The format MUST be:

{
  "skillIds": [1, 2],
  "jobRoleIds": [3, 4]
  "reasoning": "The reasoning behind the suggestions"
}

Here is the JSON schema for the response:

{
  "type": "object",
  "properties": {
    "skillIds": {
      "type": "array",
      "items": {
        "type": "integer"
      },
			"minItems": 0,
			"uniqueItems": true,
      "description": "List of suggested skill IDs"
    },
    "jobRoleIds": {
      "type": "array",
      "items": {
        "type": "integer"
      },
			"minItems": 0,
			"uniqueItems": true,
      "description": "List of suggested job role IDs"
    },
    "reasoning": {
      "type": "string",
      "description": "Explanation behind the suggestions"
    }
  },
  "required": ["skillIds", "jobRoleIds", "reasoning"],
  "additionalProperties": false
}

You MUST NOT send anything else, just the JSON object. If there are no skills or job roles, send an empty array. Do not
allucinate or make up any skills or job roles.
`

const taskSkillsAndRolesUserPrompt = `
Here are the available skills in the organization:
---
%s
---

Here are the available job roles in the organization:
---
%s
---

Here are the details of the task to analyse:
---
Task Name: %s
---
Task Description: %s
---
Tasklist Name: %s
---
Project Name: %s
---
Project Description: %s
---
`
