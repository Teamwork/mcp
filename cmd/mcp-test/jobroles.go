package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/teamwork/mcp/internal/twprojects"
)

// jobRolesSuite walks the Job Role tools, focusing on the user-membership side
// added by twprojects-set_user_jobrole / clear_user_jobrole: it creates two job
// roles, sets users onto the first, reads the membership back through
// get_jobrole, then moves one user to the second role and confirms the first
// role loses them — the replacement invariant the set tool's description
// promises, and the one thing a wire-level test cannot see. It finishes by
// clearing every membership and confirming both roles are empty. The set and
// clear tools share a path and differ only in HTTP verb, so this exercises that
// both reach the right endpoint end-to-end.
//
// Job roles are site-level, so this suite does not touch the project named by
// PROJECT_ID; that value is only used to keep the created role names unique.
type jobRolesSuite struct {
	r *runner

	jobRoleIDA int64
	jobRoleIDB int64
	userIDs    []int64 // users assigned during the run, in the order picked
}

func newJobRolesSuite(r *runner) suite {
	return &jobRolesSuite{r: r}
}

func (s *jobRolesSuite) steps() []step {
	return []step{
		{"LIST users via MCP — pick assignees", s.stepPickUsers},
		{"CREATE job role A via MCP", s.stepCreateJobRoleA},
		{"CREATE job role B via MCP", s.stepCreateJobRoleB},
		{"SET users' job role to A via MCP", s.stepSetUsersToA},
		{"GET job role A via MCP — verify membership (POST reached the wire)", s.stepVerifyOnA},
		{"SET first user's job role to B via MCP", s.stepMoveFirstUserToB},
		{"GET job role A via MCP — verify first user moved off A", s.stepVerifyMovedOffA},
		{"GET job role B via MCP — verify first user now on B", s.stepVerifyOnB},
		{"CLEAR users from both job roles via MCP", s.stepClearUsers},
		{"GET both job roles via MCP — verify membership cleared (DELETE reached the wire)", s.stepVerifyCleared},
		{"NEGATIVE: set with empty user_ids should error clearly", s.stepNegativeEmptyUsers},
	}
}

func (s *jobRolesSuite) artefacts() []string {
	var artefacts []string
	if s.jobRoleIDA != 0 {
		artefacts = append(artefacts, fmt.Sprintf("jobRoleIDA = %d", s.jobRoleIDA))
	}
	if s.jobRoleIDB != 0 {
		artefacts = append(artefacts, fmt.Sprintf("jobRoleIDB = %d", s.jobRoleIDB))
	}
	if len(s.userIDs) > 0 {
		artefacts = append(artefacts, fmt.Sprintf("assigned  = %v (cleared unless a step failed)", s.userIDs))
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

func (s *jobRolesSuite) stepCreateJobRoleA(ctx context.Context) error {
	id, err := s.createJobRole(ctx, fmt.Sprintf("mcp-test-jobrole-a-%d", s.r.projectID))
	if err != nil {
		return err
	}
	s.jobRoleIDA = id
	fmt.Printf("  → captured jobRoleIDA=%d\n", s.jobRoleIDA)
	return nil
}

func (s *jobRolesSuite) stepCreateJobRoleB(ctx context.Context) error {
	id, err := s.createJobRole(ctx, fmt.Sprintf("mcp-test-jobrole-b-%d", s.r.projectID))
	if err != nil {
		return err
	}
	s.jobRoleIDB = id
	fmt.Printf("  → captured jobRoleIDB=%d\n", s.jobRoleIDB)
	return nil
}

func (s *jobRolesSuite) stepSetUsersToA(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "set_user_jobrole",
		twprojects.JobRoleSetUser(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleIDA,
			"user_ids":    asAnyInts(s.userIDs),
		})
	return err
}

func (s *jobRolesSuite) stepVerifyOnA(ctx context.Context) error {
	members, primary, err := s.getMembership(ctx, s.jobRoleIDA)
	if err != nil {
		return err
	}
	for _, id := range s.userIDs {
		if !contains(members, id) {
			return fmt.Errorf("expected user %d in job role A membership %v after set", id, members)
		}
		if !contains(primary, id) {
			return fmt.Errorf("expected user %d in job role A primaryUsers %v after set — "+
				"set should make the role their primary one", id, primary)
		}
	}
	fmt.Printf("  ✓ all users present in role A membership and primaryUsers: %v / %v\n", members, primary)
	return nil
}

// stepMoveFirstUserToB sets the first user's role to B. Because setting a user's
// role replaces the one they held, this is the move the set tool's description
// promises: it must remove the user from role A, which the next step asserts.
func (s *jobRolesSuite) stepMoveFirstUserToB(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "set_user_jobrole (move to B)",
		twprojects.JobRoleSetUser(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleIDB,
			"user_ids":    asAnyInts(s.userIDs[:1]),
		})
	return err
}

// stepVerifyMovedOffA is the invariant a wire-level test cannot reach: setting a
// user's role to B removes them from A. A tool that merely added B alongside A
// would leave the user in A here and pass every unit test unchanged.
func (s *jobRolesSuite) stepVerifyMovedOffA(ctx context.Context) error {
	members, _, err := s.getMembership(ctx, s.jobRoleIDA)
	if err != nil {
		return err
	}
	if contains(members, s.userIDs[0]) {
		return fmt.Errorf("user %d still in role A membership %v after being set to role B — set did not move them",
			s.userIDs[0], members)
	}
	fmt.Printf("  ✓ user %d removed from role A after move to B: %v\n", s.userIDs[0], members)
	return nil
}

func (s *jobRolesSuite) stepVerifyOnB(ctx context.Context) error {
	members, primary, err := s.getMembership(ctx, s.jobRoleIDB)
	if err != nil {
		return err
	}
	if !contains(members, s.userIDs[0]) {
		return fmt.Errorf("expected user %d in role B membership %v after move", s.userIDs[0], members)
	}
	if !contains(primary, s.userIDs[0]) {
		return fmt.Errorf("expected user %d in role B primaryUsers %v after move — "+
			"set should make the role their primary one", s.userIDs[0], primary)
	}
	fmt.Printf("  ✓ user %d now present in role B membership and primaryUsers: %v / %v\n",
		s.userIDs[0], members, primary)
	return nil
}

// stepClearUsers removes every assigned user from whichever role now holds them:
// the first user was moved to B, and any remaining users are still on A.
func (s *jobRolesSuite) stepClearUsers(ctx context.Context) error {
	if _, err := s.r.callToolExpectOK(ctx, "clear_user_jobrole (B)",
		twprojects.JobRoleClearUser(s.r.engine), map[string]any{
			"job_role_id": s.jobRoleIDB,
			"user_ids":    asAnyInts(s.userIDs[:1]),
		}); err != nil {
		return err
	}
	if len(s.userIDs) > 1 {
		if _, err := s.r.callToolExpectOK(ctx, "clear_user_jobrole (A)",
			twprojects.JobRoleClearUser(s.r.engine), map[string]any{
				"job_role_id": s.jobRoleIDA,
				"user_ids":    asAnyInts(s.userIDs[1:]),
			}); err != nil {
			return err
		}
	}
	return nil
}

func (s *jobRolesSuite) stepVerifyCleared(ctx context.Context) error {
	for _, roleID := range []int64{s.jobRoleIDA, s.jobRoleIDB} {
		members, _, err := s.getMembership(ctx, roleID)
		if err != nil {
			return err
		}
		for _, id := range s.userIDs {
			if contains(members, id) {
				return fmt.Errorf("user %d still in role %d membership %v after clear", id, roleID, members)
			}
		}
		fmt.Printf("  ✓ role %d membership cleared: %v\n", roleID, members)
	}
	return nil
}

func (s *jobRolesSuite) stepNegativeEmptyUsers(ctx context.Context) error {
	text, isError, err := s.r.callTool(ctx, twprojects.JobRoleSetUser(s.r.engine), map[string]any{
		"job_role_id": s.jobRoleIDA,
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

func (s *jobRolesSuite) createJobRole(ctx context.Context, name string) (int64, error) {
	text, err := s.r.callToolExpectOK(ctx, "create_jobrole",
		twprojects.JobRoleCreate(s.r.engine), map[string]any{
			"name": name,
		})
	if err != nil {
		return 0, err
	}
	id, err := extractTrailingID(text)
	if err != nil {
		return 0, fmt.Errorf("extract job role id: %w", err)
	}
	return id, nil
}

// getMembership reads the job role and returns the user IDs under users and
// primaryUsers. The get handler marshals the JobRoleGetResponse, so membership
// lives under jobRole.users / jobRole.primaryUsers as {id,type} references.
func (s *jobRolesSuite) getMembership(ctx context.Context, roleID int64) (members, primary []int64, err error) {
	text, err := s.r.callToolExpectOK(ctx, "get_jobrole",
		twprojects.JobRoleGet(s.r.engine), map[string]any{
			"id": roleID,
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

// cleanup deletes the job roles, which also removes any membership they still
// carry, so a failed run mid-way does not strand assignments.
func (s *jobRolesSuite) cleanup(ctx context.Context) {
	for _, roleID := range []int64{s.jobRoleIDA, s.jobRoleIDB} {
		if roleID == 0 {
			continue
		}
		fmt.Printf("  delete job role %d via MCP\n", roleID)
		s.r.callToolIgnoreError(ctx, fmt.Sprintf("delete job role %d", roleID),
			twprojects.JobRoleDelete(s.r.engine), map[string]any{
				"id": roleID,
			})
	}
}
