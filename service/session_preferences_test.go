package service

import (
	"testing"

	"github.com/coder/acp-go-sdk"
)

func TestPreferredSessionConfig(t *testing.T) {
	modelCategory := acp.SessionConfigOptionCategoryModel
	grouped := acp.SessionConfigSelectOptionsGrouped{{
		Name: "Reasoning",
		Options: []acp.SessionConfigSelectOption{
			{Name: "Low", Value: "low"},
			{Name: "High", Value: "high"},
		},
	}}
	models := acp.SessionConfigSelectOptionsUngrouped{
		{Name: "Small", Value: "small"},
		{Name: "Large", Value: "large"},
	}
	options := []acp.SessionConfigOption{
		{Select: &acp.SessionConfigOptionSelect{
			Id: "model", Category: &modelCategory, CurrentValue: "small",
			Options: acp.SessionConfigSelectOptions{Ungrouped: &models},
		}},
		{Select: &acp.SessionConfigOptionSelect{
			Id: "reasoning_effort", CurrentValue: "low",
			Options: acp.SessionConfigSelectOptions{Grouped: &grouped},
		}},
	}

	if id, ok := preferredSessionConfig(options, acp.SessionConfigOptionCategoryModel, "large"); !ok || id != "model" {
		t.Fatalf("model config = %q, %t", id, ok)
	}
	if id, ok := preferredSessionConfig(options, acp.SessionConfigOptionCategoryThoughtLevel, "high"); !ok || id != "reasoning_effort" {
		t.Fatalf("reasoning config = %q, %t", id, ok)
	}
	if _, ok := preferredSessionConfig(options, acp.SessionConfigOptionCategoryModel, "missing"); ok {
		t.Fatal("unavailable model should not be applied")
	}
	if _, ok := preferredSessionConfig(options, acp.SessionConfigOptionCategoryModel, "small"); ok {
		t.Fatal("current model should not be applied again")
	}
}
