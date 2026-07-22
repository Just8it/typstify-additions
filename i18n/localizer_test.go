package i18n

import "testing"

func TestStudentInterfaceTranslations(t *testing.T) {
	t.Cleanup(func() { _ = SetLocale("en-us") })

	if err := SetLocale("de"); err != nil {
		t.Fatal(err)
	}
	if got := Translate("Preview width"); got != "Vorschaubreite" {
		t.Fatalf("German preview label = %q", got)
	}
	if got := Translate("Choose how much editor space the preview uses when it opens. You can still drag the divider while editing."); got != "Wähle, wie viel Platz die Vorschau beim Öffnen im Editor verwendet. Du kannst den Trenner beim Bearbeiten weiterhin ziehen." {
		t.Fatalf("German preview description = %q", got)
	}
	if got := Translate("Paste asset: %s", "Fehler"); got != "Asset einfügen: Fehler" {
		t.Fatalf("German paste error = %q", got)
	}
	if got := Translate("Shortcuts"); got != "Tastenkürzel" {
		t.Fatalf("German shortcuts label = %q", got)
	}

	if err := SetLocale("zh-cn"); err != nil {
		t.Fatal(err)
	}
	if got := Translate("New chat"); got != "新建聊天" {
		t.Fatalf("Chinese new-chat label = %q", got)
	}
}
