package tenancy

import (
	"context"
	"testing"
	"time"

	"github.com/parama/booking/internal/domain"
)

type memBiz struct {
	b *domain.Business
}

func (m *memBiz) Create(_ context.Context, b *domain.Business) error {
	if m.b != nil {
		return domain.ErrConflict
	}
	m.b = b
	return nil
}

func (m *memBiz) Get(_ context.Context, id string) (*domain.Business, error) {
	if m.b == nil || m.b.ID != id {
		return nil, domain.ErrNotFound
	}
	return m.b, nil
}

func (m *memBiz) Save(_ context.Context, b *domain.Business) error {
	m.b = b
	return nil
}

func TestRegisterBusiness_AndPatch(t *testing.T) {
	repo := &memBiz{}
	app := &Application{Businesses: repo, Now: func() time.Time {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}}

	b, err := app.RegisterBusiness(context.Background(), RegisterBusinessInput{
		Name: "Salon", Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == "" || b.Name != "Salon" {
		t.Fatalf("%+v", b)
	}

	patched, err := app.PatchBusiness(context.Background(), b.ID, PatchBusinessInput{
		Name: ptr("Salon 2"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Name != "Salon 2" {
		t.Fatal(patched.Name)
	}
}

func ptr(s string) *string { return &s }
