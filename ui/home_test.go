package ui

import (
	"math"
	"testing"
)

func TestEditorRatioForPreviewWidth(t *testing.T) {
	tests := []struct {
		previewWidth int
		want         float32
	}{
		{previewWidth: 25, want: 0.75},
		{previewWidth: 50, want: 0.50},
		{previewWidth: 90, want: 0.70},
	}
	for _, test := range tests {
		if got := editorRatioForPreviewWidth(test.previewWidth); math.Abs(float64(got-test.want)) > 0.001 {
			t.Fatalf("editor ratio for preview width %d = %.2f, want %.2f", test.previewWidth, got, test.want)
		}
	}
}
