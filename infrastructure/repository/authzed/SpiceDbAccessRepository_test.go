package authzed

import (
	"authz/domain"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var container *LocalSpiceDbContainer

func TestMain(m *testing.M) {
	factory := NewLocalSpiceDbContainerFactory()
	var err error
	container, err = factory.CreateContainer()

	if err != nil {
		fmt.Printf("Error initializing Docker container: %s", err)
		os.Exit(-1)
	}

	result := m.Run()

	container.Close()
	os.Exit(result)
}

func TestCheckAccess(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	client, err := container.CreateClient()
	assert.NoError(t, err)

	cases := []struct {
		sub       domain.SubjectID
		operation string
		resource  domain.Resource
		expected  domain.AccessDecision
	}{
		{sub: "u1", operation: "access", resource: domain.Resource{Type: "license", ID: "o1/smarts"}, expected: true},
		{sub: "u1", operation: "access", resource: domain.Resource{Type: "license", ID: "o1/doesnotexist"}, expected: false},
		{sub: "doesnotexist", operation: "access", resource: domain.Resource{Type: "license", ID: "o1/smarts"}, expected: false},
	}

	for _, testcase := range cases {
		actual, err := client.CheckAccess(testcase.sub, testcase.operation, testcase.resource)
		assert.NoError(t, err, fmt.Sprintf("Error in case (subject: %s, operation: %s, resource: [%s, %s])", testcase.sub, testcase.operation, testcase.resource.Type, testcase.resource.ID))
		assert.Equal(t, testcase.expected, actual, "Unexpected result for case (subject: %s, operation: %s, resource: [%s, %s])", testcase.sub, testcase.operation, testcase.resource.Type, testcase.resource.ID)
	}
}

func TestGetLicense(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	lic, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, "o1", lic.OrgID)
	assert.Equal(t, "smarts", lic.ServiceID)
	assert.Equal(t, 10, lic.MaxSeats)
	assert.Equal(t, 1, lic.InUse)
}

func TestGetAssigned(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	assigned, err := client.GetAssigned("o1", "smarts")
	assert.NoError(t, err)

	assert.ElementsMatch(t, []domain.SubjectID{"u1"}, assigned)
}

func TestRapidAssignments(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	for i := 2; i <= 10; i++ {
		err = client.AssignSeats([]domain.SubjectID{domain.SubjectID(fmt.Sprintf("u%d", i))}, "o1", domain.Service{ID: "smarts"})
		assert.NoError(t, err)
	}

	lic, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, 10, lic.InUse)
}

func TestAssignBatch(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	// given
	subs := []domain.SubjectID{
		"u100", "u101",
	}
	oldLic, e1 := client.GetLicense("o1", "smarts")
	assert.NoError(t, e1)
	assert.Equal(t, 1, oldLic.InUse)

	// when
	err = client.AssignSeats(subs, "o1", domain.Service{ID: "smarts"})

	// then
	assert.NoError(t, err)
	newLic, e2 := client.GetLicense("o1", "smarts")
	assert.NoError(t, e2)
	assert.Equal(t, oldLic.InUse+len(subs), newLic.InUse)
}

func TestUnassignBatch(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	// given
	subs := []domain.SubjectID{
		"u1",
	}

	oldLic, e1 := client.GetLicense("o1", "smarts")
	assert.NoError(t, e1)
	assert.Equal(t, 1, oldLic.InUse)

	// when
	err = client.UnAssignSeats(subs, "o1", domain.Service{ID: "smarts"})

	// then
	assert.NoError(t, err)
	newLic, e2 := client.GetLicense("o1", "smarts")
	assert.NoError(t, e2)
	assert.Equal(t, oldLic.InUse-len(subs), newLic.InUse)
}

func TestAssignUnassign(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	err = client.AssignSeats([]domain.SubjectID{"u2"}, "o1", domain.Service{ID: "smarts"})
	assert.NoError(t, err)

	lic, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, 2, lic.InUse)

	err = client.UnAssignSeats([]domain.SubjectID{"u2"}, "o1", domain.Service{ID: "smarts"})
	assert.NoError(t, err)

	lic, err = client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, 1, lic.InUse)
}

func TestUnassignNotAssigned(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	licBefore, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	err = client.UnAssignSeats([]domain.SubjectID{"not_assigned"}, "o1", domain.Service{ID: "smarts"})
	assert.Error(t, err)

	licAfter, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, licBefore.InUse, licAfter.InUse)
}

func TestAssignAlreadyAssigned(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Parallel()

	client, err := container.CreateClient()
	assert.NoError(t, err)

	licBefore, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	err = client.AssignSeats([]domain.SubjectID{"u1"}, "o1", domain.Service{ID: "smarts"})
	assert.Error(t, err)

	licAfter, err := client.GetLicense("o1", "smarts")
	assert.NoError(t, err)

	assert.Equal(t, licBefore.InUse, licAfter.InUse)
}
