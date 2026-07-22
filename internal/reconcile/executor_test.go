package reconcile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
)

type fakeRewriteClient struct {
	operations []string
	failOn     string
}

func (c *fakeRewriteClient) AddRewrite(
	_ context.Context,
	rewrite adguard.Rewrite,
) error {
	operation := "add:" + rewrite.Domain
	c.operations = append(c.operations, operation)

	if c.failOn == operation {
		return errors.New("forced addition failure")
	}

	return nil
}

func (c *fakeRewriteClient) UpdateRewrite(
	_ context.Context,
	current adguard.Rewrite,
	desired adguard.Rewrite,
) error {
	operation := "update:" + desired.Domain
	c.operations = append(c.operations, operation)

	if c.failOn == operation {
		return errors.New("forced update failure")
	}

	_ = current

	return nil
}

func (c *fakeRewriteClient) DeleteRewrite(
	_ context.Context,
	rewrite adguard.Rewrite,
) error {
	operation := "delete:" + rewrite.Domain
	c.operations = append(c.operations, operation)

	if c.failOn == operation {
		return errors.New("forced deletion failure")
	}

	return nil
}

func TestExecuteAppliesPlanInSafeOrder(t *testing.T) {
	client := &fakeRewriteClient{}

	plan := Plan{
		Add: []adguard.Rewrite{
			{
				Domain: "new.internal",
				Answer: "172.20.0.10",
			},
		},
		Update: []Change{
			{
				Current: adguard.Rewrite{
					Domain: "changed.internal",
					Answer: "172.20.0.99",
				},
				Desired: adguard.Rewrite{
					Domain: "changed.internal",
					Answer: "172.20.0.20",
				},
			},
		},
		Delete: []adguard.Rewrite{
			{
				Domain: "stale.internal",
				Answer: "172.20.0.50",
			},
		},
	}

	result, err := Execute(
		context.Background(),
		client,
		plan,
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned an unexpected error: %v",
			err,
		)
	}

	expectedOperations := []string{
		"add:new.internal",
		"update:changed.internal",
		"delete:stale.internal",
	}

	if len(client.operations) != len(expectedOperations) {
		t.Fatalf(
			"len(operations) = %d, expected %d",
			len(client.operations),
			len(expectedOperations),
		)
	}

	for index := range expectedOperations {
		if client.operations[index] !=
			expectedOperations[index] {
			t.Errorf(
				"operations[%d] = %q, expected %q",
				index,
				client.operations[index],
				expectedOperations[index],
			)
		}
	}

	if result.Added != 1 {
		t.Errorf(
			"Added = %d, expected 1",
			result.Added,
		)
	}

	if result.Updated != 1 {
		t.Errorf(
			"Updated = %d, expected 1",
			result.Updated,
		)
	}

	if result.Deleted != 1 {
		t.Errorf(
			"Deleted = %d, expected 1",
			result.Deleted,
		)
	}
}

func TestExecuteStopsAfterFailure(t *testing.T) {
	client := &fakeRewriteClient{
		failOn: "update:changed.internal",
	}

	plan := Plan{
		Add: []adguard.Rewrite{
			{
				Domain: "new.internal",
				Answer: "172.20.0.10",
			},
		},
		Update: []Change{
			{
				Current: adguard.Rewrite{
					Domain: "changed.internal",
					Answer: "172.20.0.99",
				},
				Desired: adguard.Rewrite{
					Domain: "changed.internal",
					Answer: "172.20.0.20",
				},
			},
		},
		Delete: []adguard.Rewrite{
			{
				Domain: "stale.internal",
				Answer: "172.20.0.50",
			},
		},
	}

	result, err := Execute(
		context.Background(),
		client,
		plan,
	)
	if err == nil {
		t.Fatal("Execute() returned nil error")
	}

	if !strings.Contains(
		err.Error(),
		"execute rewrite update",
	) {
		t.Errorf(
			"error = %q, expected update error",
			err,
		)
	}

	if result.Added != 1 {
		t.Errorf(
			"Added = %d, expected 1",
			result.Added,
		)
	}

	if result.Updated != 0 {
		t.Errorf(
			"Updated = %d, expected 0",
			result.Updated,
		)
	}

	if result.Deleted != 0 {
		t.Errorf(
			"Deleted = %d, expected 0",
			result.Deleted,
		)
	}

	for _, operation := range client.operations {
		if operation == "delete:stale.internal" {
			t.Error(
				"deletion was executed after update failure",
			)
		}
	}
}

func TestExecuteEmptyPlan(t *testing.T) {
	client := &fakeRewriteClient{}

	result, err := Execute(
		context.Background(),
		client,
		Plan{},
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned an unexpected error: %v",
			err,
		)
	}

	if result != (ExecutionResult{}) {
		t.Errorf(
			"result = %+v, expected zero result",
			result,
		)
	}

	if len(client.operations) != 0 {
		t.Errorf(
			"len(operations) = %d, expected 0",
			len(client.operations),
		)
	}
}
