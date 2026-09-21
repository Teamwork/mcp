package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/teamwork/mcp/internal/twprojects"
)

// jobRolesSuite walks the Job Role tools, focusing on the user-membership side
// added by twprojects-assign_jobrole / unassign_jobrole: it creates a job role,
// assigns real users to it, promotes one to primary, reads the membership back
// through get_jobrole, then removes it again and confirms the role is empty. The
// assign and unassign tools share a path and differ only in HTTP verb, so this
// exercises that both reach the right endpoint end-to-end.
//
// Job roles are site-level, so this suite does not touch the project named by
// PROJECT_ID; that value is only used to keep the created role's name unique.
type jobRolesSuite struct {
	r *runner

	jobRoleID int64
	userIDs   []int64 // users assigned during the run, in the order picked
}

func newJobRolesSuite(r *runner) suite {
	return &jobRolesSuite{r: r}
}

func (s *jobRolesSuite) steps() []step {
	return []step{
		{"LIST users via MCP — pick assignees", s.stepPickUsers},
		{"CREATE job role via MCP", s.stepCreateJobRole},
		{"ASSIGN users to job role via MCP", s.stepAssignUsers},
		{"GET job role via MCP — verify membership (POST reached the wire)", s.stepVerifyAssigned},
		{"ASSIGN first user as PRIMARY via MCP", s.stepAssignPrimary},
		{"GET job role via MCP — verify primary membership", s.stepVerifyPrimary},
		{"UNASSIGN users from job role via MCP", s.stepUnassignUsers},
		{"GET job role via MCP — verify membership cleared (DELETE reached the wire)", s.stepVerifyUnassigned},
		{"NEGATIVE: assign with empty user_ids should error clearly", s.stepNegativeEmptyUsers},
	}
}

func (s *jobRolesSuite) artefacts() []string {
	var artefacts []string
	if s.jobRoleID != 0 {
		artefacts = append(artefacts, fmt.Sprintf("jobRoleID = %d", s.jobRoleID))
	}
	if len(s.userIDs) > 0 {
		artefacts = append(artefacts, fmt.Sprintf("assigned  = %v (removed unless a step failed)", s.userIDs))
	}
	return artefacts
}

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

// stepPickUsers lists users and captures up to two IDs to use as assignees, so
// the suite works against whatever accounts the site actually has.
func (s *jobRolesSuite) stepPickUsers(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "list_users",
		twprojects.UserList(s.r.engine), map[string]any{
			"verbose": false,
		})
	if err != nil {
		return err
	}
	var listResp struct {
		People []struct {
			ID        int64  `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"people"`
	}
	if err := json.Unmarshal([]byte(text), &listResp); err != nil {
		return fmt.Errorf("decode users list: %w", err)
	}
	for _, person := range listResp.People {
		if person.ID == 0 {
			continue
		}
		s.userIDs = append(s.userIDs, person.ID)
		fmt.Printf("  → assignee %d (%s %s)\n", person.ID, person.FirstName, person.LastName)
		if len(s.userIDs) == 2 {
			break
		}
	}
	if len(s.userIDs) == 0 {
		return fmt.Errorf("no users found on the site to assign")
	}
	return nil
}

func (s *jobRolesSuite) stepCreateJobRole(ctx context.Context) error {
	name := fmt.Sprintf("mcp-test-jobrole-%d", s.r.projectID)
	text, err := s.r.callToolExpectOK(ctx, "create_jobrole",
		twprojects.JobRoleCreate(s.r.engine), map[string]any{
			"name": name,
		})
	if err != nil {
		return err
	}
	s.jobRoleID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract job role id: %w", err)
	}
	fmt.Printf("  → captured jobRoleID=%d\n", s.jobRoleID)
	return nil
}

func (s *jobRolesSuite) stepAssignUsers(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "assign_jobrole",
		twprojects.JobRoleAssignUsers(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleID,
			"user_ids":    asAnyInts(s.userIDs),
			"is_primary":  false,
		})
	return err
}

func (s *jobRolesSuite) stepVerifyAssigned(ctx context.Context) error {
	members, _, err := s.getMembership(ctx)
	if err != nil {
		return err
	}
	for _, id := range s.userIDs {
		if !contains(members, id) {
			return fmt.Errorf("expected user %d in job role membership %v after assign", id, members)
		}
	}
	fmt.Printf("  ✓ all assigned users present in membership: %v\n", members)
	return nil
}

func (s *jobRolesSuite) stepAssignPrimary(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "assign_jobrole (primary)",
		twprojects.JobRoleAssignUsers(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleID,
			"user_ids":    asAnyInts(s.userIDs[:1]),
			"is_primary":  true,
		})
	return err
}

func (s *jobRolesSuite) stepVerifyPrimary(ctx context.Context) error {
	_, primary, err := s.getMembership(ctx)
	if err != nil {
		return err
	}
	if !contains(primary, s.userIDs[0]) {
		return fmt.Errorf("expected user %d in primaryUsers %v after primary assign", s.userIDs[0], primary)
	}
	fmt.Printf("  ✓ user %d is now a primary holder: %v\n", s.userIDs[0], primary)
	return nil
}

func (s *jobRolesSuite) stepUnassignUsers(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "unassign_jobrole",
		twprojects.JobRoleUnassignUsers(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleID,
			"user_ids":    asAnyInts(s.userIDs),
		})
	return err
}

func (s *jobRolesSuite) stepVerifyUnassigned(ctx context.Context) error {
	members, _, err := s.getMembership(ctx)
	if err != nil {
		return err
	}
	for _, id := range s.userIDs {
		if contains(members, id) {
			return fmt.Errorf("user %d still in membership %v after unassign", id, members)
		}
	}
	fmt.Printf("  ✓ membership cleared after unassign: %v\n", members)
	return nil
}

func (s *jobRolesSuite) stepNegativeEmptyUsers(ctx context.Context) error {
	text, isError, err := s.r.callTool(ctx, twprojects.JobRoleAssignUsers(s.r.engine), map[string]any{
		"job_role_id": s.jobRoleID,
		"user_ids":    []any{},
	})
	if err != nil {
		return fmt.Errorf("negative test: %w", err)
	}
	if !isError {
		return fmt.Errorf("expected an error result for empty user_ids, got success: %s", text)
	}
	fmt.Printf("  ✓ empty user_ids error surfaced: %s\n", strings.TrimSpace(text))
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getMembership reads the job role and returns the user IDs under users and
// primaryUsers. The get handler marshals the JobRoleGetResponse, so membership
// lives under jobRole.users / jobRole.primaryUsers as {id,type} references.
func (s *jobRolesSuite) getMembership(ctx context.Context) (members, primary []int64, err error) {
	text, err := s.r.callToolExpectOK(ctx, "get_jobrole",
		twprojects.JobRoleGet(s.r.engine), map[string]any{
			"id": s.jobRoleID,
		})
	if err != nil {
		return nil, nil, err
	}
	var getResp struct {
		JobRole struct {
			Users []struct {
				ID int64 `json:"id"`
			} `json:"users"`
			PrimaryUsers []struct {
				ID int64 `json:"id"`
			} `json:"primaryUsers"`
		} `json:"jobRole"`
	}
	if err := json.Unmarshal([]byte(text), &getResp); err != nil {
		return nil, nil, fmt.Errorf("decode job role: %w", err)
	}
	for _, user := range getResp.JobRole.Users {
		members = append(members, user.ID)
	}
	for _, user := range getResp.JobRole.PrimaryUsers {
		primary = append(primary, user.ID)
	}
	return members, primary, nil
}

func contains(ids []int64, want int64) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

// cleanup deletes the job role, which also removes any membership it still
// carries, so a failed run mid-way does not strand assignments.
func (s *jobRolesSuite) cleanup(ctx context.Context) {
	if s.jobRoleID == 0 {
		return
	}
	fmt.Printf("  delete job role %d via MCP\n", s.jobRoleID)
	s.r.callToolIgnoreError(ctx, fmt.Sprintf("delete job role %d", s.jobRoleID),
		twprojects.JobRoleDelete(s.r.engine), map[string]any{
			"id": s.jobRoleID,
		})
}
