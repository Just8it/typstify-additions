package settings

import (
	"testing"

	config "looz.ws/typstify/service/settings"
)

func TestShortcutDefinitionsHaveLocalizedLabels(t *testing.T) {
	for _, definition := range config.ShortcutDefinitions() {
		if shortcutCategoryLabel(definition.Category) == "" ||
			shortcutNameLabel(definition.ID) == string(definition.ID) ||
			shortcutDescriptionLabel(definition.ID) == "" {
			t.Fatalf("shortcut %q is missing localized copy", definition.ID)
		}
	}
}
