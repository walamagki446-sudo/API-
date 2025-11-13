package main

import (
	"testing"
	"time"
)

func TestStorageOperations(t *testing.T) {
	// Create temporary storage
	storage := NewStorage("test_subs.json")
	defer func() {
		// Clean up
		storage.RemoveSubscription(12345, "192.168.1.1")
		storage.Save()
	}()

	// Test adding subscription
	sub := &Subscription{
		UserID:        12345,
		Username:      "testuser",
		IP:            "192.168.1.1",
		TotalDays:     30,
		DaysRemaining: 30,
		StartDate:     time.Now(),
		LastChecked:   time.Now(),
		IsPaused:      false,
		Active:        true,
	}

	storage.AddSubscription(sub)

	// Test retrieving subscription
	retrieved := storage.GetSubscription(12345, "192.168.1.1")
	if retrieved == nil {
		t.Error("Failed to retrieve subscription")
	}
	if retrieved.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", retrieved.Username)
	}
	if retrieved.DaysRemaining != 30 {
		t.Errorf("Expected 30 days remaining, got %d", retrieved.DaysRemaining)
	}

	// Test pause
	retrieved.IsPaused = true
	retrieved.PausedAt = time.Now()
	retrieved.PausedDaysLeft = 30
	storage.UpdateSubscription(retrieved)

	paused := storage.GetSubscription(12345, "192.168.1.1")
	if !paused.IsPaused {
		t.Error("Subscription should be paused")
	}

	// Test resume
	paused.IsPaused = false
	paused.LastChecked = time.Now()
	storage.UpdateSubscription(paused)

	resumed := storage.GetSubscription(12345, "192.168.1.1")
	if resumed.IsPaused {
		t.Error("Subscription should not be paused after resume")
	}

	// Test getting all active subscriptions
	active := storage.GetAllActiveSubscriptions()
	if len(active) == 0 {
		t.Error("Should have at least one active subscription")
	}

	// Test removing subscription
	storage.RemoveSubscription(12345, "192.168.1.1")
	removed := storage.GetSubscription(12345, "192.168.1.1")
	if removed != nil {
		t.Error("Subscription should be removed")
	}
}

func TestDaysRemainingUpdate(t *testing.T) {
	// Initialize storage for the test
	storage = NewStorage("test_days.json")
	defer func() {
		storage = nil
	}()

	// Create a subscription that started 2 days ago
	sub := &Subscription{
		UserID:        12345,
		Username:      "testuser",
		IP:            "192.168.1.1",
		TotalDays:     30,
		DaysRemaining: 30,
		StartDate:     time.Now().Add(-48 * time.Hour),
		LastChecked:   time.Now().Add(-48 * time.Hour),
		IsPaused:      false,
		Active:        true,
	}

	// Update days remaining
	updateDaysRemaining(sub)

	// Should have 28 days remaining (30 - 2)
	if sub.DaysRemaining != 28 {
		t.Errorf("Expected 28 days remaining, got %d", sub.DaysRemaining)
	}

	// Test paused subscription doesn't update
	sub.IsPaused = true
	sub.DaysRemaining = 28
	sub.LastChecked = time.Now().Add(-24 * time.Hour)
	
	updateDaysRemaining(sub)
	
	// Should still have 28 days (paused)
	if sub.DaysRemaining != 28 {
		t.Errorf("Paused subscription should not update, expected 28 days remaining, got %d", sub.DaysRemaining)
	}
}

func TestHashString(t *testing.T) {
	// Test that hash is consistent
	hash1 := hashString("testuser")
	hash2 := hashString("testuser")
	
	if hash1 != hash2 {
		t.Error("Hash should be consistent for same input")
	}

	// Test that different strings have different hashes
	hash3 := hashString("differentuser")
	if hash1 == hash3 {
		t.Error("Different strings should produce different hashes")
	}

	// Test that hash is positive
	if hash1 < 0 {
		t.Error("Hash should be positive")
	}
}

func TestGetKey(t *testing.T) {
	storage := NewStorage("test.json")
	
	key1 := storage.GetKey(12345, "192.168.1.1")
	expected := "12345_192.168.1.1"
	
	if key1 != expected {
		t.Errorf("Expected key '%s', got '%s'", expected, key1)
	}

	// Test different IPs produce different keys
	key2 := storage.GetKey(12345, "192.168.1.2")
	if key1 == key2 {
		t.Error("Different IPs should produce different keys")
	}

	// Test different user IDs produce different keys
	key3 := storage.GetKey(54321, "192.168.1.1")
	if key1 == key3 {
		t.Error("Different user IDs should produce different keys")
	}
}

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test_text", "test\\_text"},
		{"test*bold*", "test\\*bold\\*"},
		{"test.com", "test\\.com"},
		{"test-dash", "test\\-dash"},
		{"normal", "normal"},
	}

	for _, test := range tests {
		result := escapeMarkdownV2(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", test.input, test.expected, result)
		}
	}
}
