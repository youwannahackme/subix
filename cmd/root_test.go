package cmd

import (
	"testing"
)

func TestUniqueFlagDefault(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("unique")
	if flag == nil {
		t.Fatal("flag --unique not registered on rootCmd")
	}

	if flag.DefValue != "true" {
		t.Fatalf("expected default value for --unique flag to be 'true', got '%s'", flag.DefValue)
	}

	// Test flag parsing with empty flags
	err := rootCmd.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	val, err := rootCmd.PersistentFlags().GetBool("unique")
	if err != nil {
		t.Fatalf("failed to get bool for --unique: %v", err)
	}
	if !val {
		t.Errorf("expected --unique to evaluate to true by default, got %v", val)
	}
	if !removeDuplicate {
		t.Errorf("expected removeDuplicate variable to be true by default, got %v", removeDuplicate)
	}

	// Test flag parsing with --unique=false
	err = rootCmd.ParseFlags([]string{"--unique=false"})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	val, err = rootCmd.PersistentFlags().GetBool("unique")
	if err != nil {
		t.Fatalf("failed to get bool for --unique: %v", err)
	}
	if val {
		t.Errorf("expected --unique to evaluate to false when passed --unique=false, got %v", val)
	}
	if removeDuplicate {
		t.Errorf("expected removeDuplicate variable to be false, got %v", removeDuplicate)
	}

	// Reset back to default
	_ = rootCmd.ParseFlags([]string{})
}
