package main

import (
"os"
"testing"
)

func TestGetEnvAsInt64Slice(t *testing.T) {
// Test empty string
os.Setenv("TEST_ADMIN_IDS", "")
result := getEnvAsInt64Slice("TEST_ADMIN_IDS", []int64{})
if len(result) != 0 {
t.Errorf("Expected empty slice, got %v", result)
}

// Test single admin
os.Setenv("TEST_ADMIN_IDS", "123456789")
result = getEnvAsInt64Slice("TEST_ADMIN_IDS", []int64{})
if len(result) != 1 || result[0] != 123456789 {
t.Errorf("Expected [123456789], got %v", result)
}

// Test multiple admins
os.Setenv("TEST_ADMIN_IDS", "123456789,987654321,555555555")
result = getEnvAsInt64Slice("TEST_ADMIN_IDS", []int64{})
if len(result) != 3 {
t.Errorf("Expected 3 admins, got %d", len(result))
}
if result[0] != 123456789 || result[1] != 987654321 || result[2] != 555555555 {
t.Errorf("Expected [123456789, 987654321, 555555555], got %v", result)
}

// Test with spaces
os.Setenv("TEST_ADMIN_IDS", "123456789, 987654321 , 555555555")
result = getEnvAsInt64Slice("TEST_ADMIN_IDS", []int64{})
if len(result) != 3 {
t.Errorf("Expected 3 admins with spaces, got %d", len(result))
}

os.Unsetenv("TEST_ADMIN_IDS")
}

func TestIsAdmin(t *testing.T) {
// Set up test config
config.AdminIDs = []int64{123456789, 987654321, 555555555}

// Test valid admin
if !isAdmin(123456789) {
t.Error("Expected 123456789 to be admin")
}
if !isAdmin(987654321) {
t.Error("Expected 987654321 to be admin")
}
if !isAdmin(555555555) {
t.Error("Expected 555555555 to be admin")
}

// Test non-admin
if isAdmin(111111111) {
t.Error("Expected 111111111 to NOT be admin")
}

// Test empty admin list
config.AdminIDs = []int64{}
if isAdmin(123456789) {
t.Error("Expected no admins with empty list")
}
}
