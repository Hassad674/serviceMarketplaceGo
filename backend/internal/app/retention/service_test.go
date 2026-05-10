package retention_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retentionapp "marketplace-backend/internal/app/retention"
	"marketplace-backend/internal/domain/retention"
)

// fakeRepo is a hand-rolled implementation of repository.RetentionRepository
// per the project's "mocks live next to the test" convention. Each
// policy maps to a slice of "what to return on each successive call",
// so a test can simulate "first batch deletes 3, second deletes 0"
// without juggling counters.
type fakeRepo struct {
	calls map[string]int
	plan  map[string][]int    // per-policy plan: rows-affected per call
	errs  map[string]error    // per-policy error injected on every call
	hits  map[string][]string // observed call order, for assertions
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		calls: map[string]int{},
		plan:  map[string][]int{},
		errs:  map[string]error{},
		hits:  map[string][]string{},
	}
}

func (f *fakeRepo) Sweep(_ context.Context, p retention.Policy) (int, error) {
	idx := f.calls[p.Name]
	f.calls[p.Name] = idx + 1
	f.hits["all"] = append(f.hits["all"], p.Name)
	if err, ok := f.errs[p.Name]; ok && err != nil {
		return 0, err
	}
	plan := f.plan[p.Name]
	if idx < len(plan) {
		return plan[idx], nil
	}
	return 0, nil
}

func validPolicy(name string) retention.Policy {
	return retention.Policy{
		Name:      name,
		Table:     name,
		AgeColumn: "created_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyDelete,
		BatchSize: 10,
	}
}

func TestNewService_RejectsNilRepo(t *testing.T) {
	_, err := retentionapp.NewService(nil, []retention.Policy{validPolicy("a")})
	require.Error(t, err)
}

func TestNewService_RejectsEmptyPolicySet(t *testing.T) {
	_, err := retentionapp.NewService(newFakeRepo(), nil)
	require.Error(t, err)
}

func TestNewService_RejectsInvalidPolicy(t *testing.T) {
	_, err := retentionapp.NewService(newFakeRepo(), []retention.Policy{{Name: ""}})
	require.Error(t, err)
}

func TestService_Run_LoopsUntilEmpty(t *testing.T) {
	repo := newFakeRepo()
	repo.plan["messages"] = []int{5, 5, 5, 0}
	svc, err := retentionapp.NewService(repo, []retention.Policy{validPolicy("messages")})
	require.NoError(t, err)

	results, errs := svc.Run(context.Background())
	require.Empty(t, errs)
	require.Len(t, results, 1)
	assert.Equal(t, 15, results[0].Affected, "expect 5+5+5 swept rows")
	assert.Equal(t, 4, results[0].Batches, "expect 4 calls (3 productive + 1 zero terminator)")
}

func TestService_Run_RespectsMaxBatchesPerRun(t *testing.T) {
	repo := newFakeRepo()
	// Always-1 plan would loop forever without a cap — this is the
	// load-bearing safety net.
	for i := 0; i < 1000; i++ {
		repo.plan["messages"] = append(repo.plan["messages"], 1)
	}
	svc, err := retentionapp.NewService(repo, []retention.Policy{validPolicy("messages")})
	require.NoError(t, err)
	svc = svc.WithMaxBatchesPerRun(5)

	results, errs := svc.Run(context.Background())
	require.Empty(t, errs)
	assert.Equal(t, 5, results[0].Batches)
	assert.Equal(t, 5, results[0].Affected)
}

func TestService_Run_AllPoliciesAreCalled(t *testing.T) {
	repo := newFakeRepo()
	policies := []retention.Policy{
		validPolicy("a"),
		validPolicy("b"),
		validPolicy("c"),
	}
	svc, err := retentionapp.NewService(repo, policies)
	require.NoError(t, err)

	results, errs := svc.Run(context.Background())
	require.Empty(t, errs)
	require.Len(t, results, 3)
	for _, name := range []string{"a", "b", "c"} {
		assert.GreaterOrEqual(t, repo.calls[name], 1, "policy %q never called", name)
	}
}

func TestService_Run_OnePolicyErrorDoesNotAbortLoop(t *testing.T) {
	repo := newFakeRepo()
	repo.errs["b"] = errors.New("boom")
	repo.plan["a"] = []int{2, 0}
	repo.plan["c"] = []int{3, 0}

	policies := []retention.Policy{
		validPolicy("a"),
		validPolicy("b"),
		validPolicy("c"),
	}
	svc, err := retentionapp.NewService(repo, policies)
	require.NoError(t, err)

	results, errs := svc.Run(context.Background())
	require.Len(t, errs, 1, "exactly the failing policy should report an error")
	require.Len(t, results, 3, "every policy still gets a Result entry")

	byName := map[string]retention.Result{}
	for _, r := range results {
		byName[r.Policy] = r
	}
	assert.Equal(t, 2, byName["a"].Affected)
	assert.Equal(t, 0, byName["b"].Affected, "failing policy reports zero")
	assert.Equal(t, 3, byName["c"].Affected, "policy after the failing one still ran")
}

func TestService_Run_RespectsCancellation(t *testing.T) {
	repo := newFakeRepo()
	// Long plan; we cancel the context before it can drain.
	for i := 0; i < 1000; i++ {
		repo.plan["messages"] = append(repo.plan["messages"], 1)
	}
	svc, err := retentionapp.NewService(repo, []retention.Policy{validPolicy("messages")})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel — first batch must still try, but the inter-batch sleep should bail out fast

	_, errs := svc.Run(ctx)
	// Pre-cancelled context: either Run reports the cancel error, or
	// the per-policy cancel hits during the inter-batch sleep. Either
	// outcome is acceptable; the failure mode we want to rule out is
	// "ran to completion of the 1000-batch plan".
	assert.LessOrEqual(t, repo.calls["messages"], retention.MaxBatchesPerRun)
	if len(errs) > 0 {
		// At least one error should be the cancellation propagation.
		hasCancel := false
		for _, e := range errs {
			if errors.Is(e, context.Canceled) {
				hasCancel = true
				break
			}
		}
		assert.True(t, hasCancel, "expected a cancelled-context error in: %v", errs)
	}
}

func TestService_Run_EmptyTickIsCheapAndOK(t *testing.T) {
	repo := newFakeRepo()
	// Empty plan = first call returns 0 = service exits immediately.
	svc, err := retentionapp.NewService(repo, []retention.Policy{validPolicy("messages")})
	require.NoError(t, err)
	results, errs := svc.Run(context.Background())
	require.Empty(t, errs)
	require.Len(t, results, 1)
	assert.Equal(t, 0, results[0].Affected)
	assert.Equal(t, 1, results[0].Batches, "exactly one terminating call expected")
}
