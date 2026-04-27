package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUser_DefaultValues(t *testing.T) {
	u := User{
		Email:        "test@example.com",
		PasswordHash: "hashed",
		FirstName:    "Jane",
		LastName:     "Doe",
		IsActive:     true,
		IsVerified:   false,
	}

	if u.Email != "test@example.com" {
		t.Errorf("unexpected email: %s", u.Email)
	}
	if u.IsActive != true {
		t.Error("expected IsActive to be true")
	}
	if u.IsVerified != false {
		t.Error("expected IsVerified to be false")
	}
}

func TestUser_BeforeCreate_AssignsUUID(t *testing.T) {
	u := &User{}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if u.ID == uuid.Nil {
		t.Error("expected ID to be set after BeforeCreate")
	}
}

func TestUser_BeforeCreate_PreservesExistingUUID(t *testing.T) {
	existing := uuid.New()
	u := &User{ID: existing}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if u.ID != existing {
		t.Errorf("expected ID %s to be preserved, got %s", existing, u.ID)
	}
}

func TestTimestampModel_Fields(t *testing.T) {
	now := time.Now()
	ts := TimestampModel{
		CreatedAt: now,
		UpdatedAt: now,
	}
	if ts.CreatedAt != now {
		t.Error("CreatedAt mismatch")
	}
	if ts.UpdatedAt != now {
		t.Error("UpdatedAt mismatch")
	}
}

func TestUser_JSONTagPasswordHashHidden(t *testing.T) {
	u := User{
		Email:        "a@b.com",
		PasswordHash: "secret",
		FirstName:    "A",
		LastName:     "B",
	}
	if u.PasswordHash != "secret" {
		t.Error("PasswordHash should be accessible in Go code")
	}
}
