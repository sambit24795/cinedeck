package main

import (
	"testing"

	"github.com/sambit24795/cinedeck/internal/config"
)

func TestRegionForFlagBeatsConfig(t *testing.T) {
	root := newRootCmd()
	if err := root.PersistentFlags().Set("region", "us"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := regionFor(root, config.Config{Region: "IN"}); got != "US" {
		t.Errorf("regionFor = %q, want %q", got, "US")
	}
}

func TestRegionForConfigLowercased(t *testing.T) {
	root := newRootCmd()
	if got := regionFor(root, config.Config{Region: "gb"}); got != "GB" {
		t.Errorf("regionFor = %q, want %q", got, "GB")
	}
}

func TestRegionForConfig(t *testing.T) {
	root := newRootCmd()
	if got := regionFor(root, config.Config{Region: "IN"}); got != "IN" {
		t.Errorf("regionFor = %q, want %q", got, "IN")
	}
}

func TestRegionForBlankFlagFallsBack(t *testing.T) {
	root := newRootCmd()
	if err := root.PersistentFlags().Set("region", "  "); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := regionFor(root, config.Config{Region: "IN"}); got != "IN" {
		t.Errorf("regionFor = %q, want %q", got, "IN")
	}
}

func TestRegionForEmpty(t *testing.T) {
	root := newRootCmd()
	if got := regionFor(root, config.Config{Region: ""}); got != "" {
		t.Errorf("regionFor = %q, want %q", got, "")
	}
}
