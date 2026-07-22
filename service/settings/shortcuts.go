package settings

import (
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/key"
)

type ShortcutID string

const (
	ShortcutToggleFileExplorer ShortcutID = "workspace.toggleFileExplorer"
	ShortcutToggleConsole      ShortcutID = "workspace.toggleConsole"
	ShortcutToggleAssistant    ShortcutID = "workspace.toggleAssistant"
	ShortcutTogglePreview      ShortcutID = "workspace.togglePreview"
	ShortcutSave               ShortcutID = "editor.save"
	ShortcutFind               ShortcutID = "editor.find"
	ShortcutToggleReadOnly     ShortcutID = "editor.toggleReadOnly"
	ShortcutToggleWrap         ShortcutID = "editor.toggleWrap"
	ShortcutEditorCopy         ShortcutID = "editor.copy"
	ShortcutEditorCut          ShortcutID = "editor.cut"
	ShortcutEditorPaste        ShortcutID = "editor.paste"
	ShortcutCompletion         ShortcutID = "editor.completion"
	ShortcutFileCopy           ShortcutID = "files.copy"
	ShortcutFileCut            ShortcutID = "files.cut"
	ShortcutFilePaste          ShortcutID = "files.paste"
	ShortcutSendPrompt         ShortcutID = "assistant.sendPrompt"
)

type ShortcutBinding struct {
	Name      key.Name      `json:"name"`
	Modifiers key.Modifiers `json:"modifiers,omitempty"`
}

type ShortcutOverride struct {
	Binding  *ShortcutBinding `json:"binding,omitempty"`
	Disabled bool             `json:"disabled,omitempty"`
}

type ShortcutDefinition struct {
	ID          ShortcutID
	Category    string
	Name        string
	Description string
	Default     ShortcutBinding
}

var shortcutDefinitions = []ShortcutDefinition{
	{ShortcutToggleFileExplorer, "Workspace", "Toggle file explorer", "Show or hide the project files.", ShortcutBinding{"D", key.ModShortcut}},
	{ShortcutToggleConsole, "Workspace", "Toggle console", "Show or hide compiler and application output.", ShortcutBinding{"K", key.ModShortcut}},
	{ShortcutToggleAssistant, "Workspace", "Toggle assistant", "Open or close the AI assistant.", ShortcutBinding{"L", key.ModShortcut}},
	{ShortcutTogglePreview, "Workspace", "Toggle preview", "Show or hide the document preview.", ShortcutBinding{"P", key.ModShortcut}},
	{ShortcutSave, "Editor", "Save file", "Save the active document.", ShortcutBinding{"S", key.ModShortcut}},
	{ShortcutFind, "Editor", "Find and replace", "Search in the active document.", ShortcutBinding{"F", key.ModShortcut}},
	{ShortcutToggleReadOnly, "Editor", "Toggle read-only mode", "Lock or unlock editing for the active document.", ShortcutBinding{"L", key.ModShortcut}},
	{ShortcutToggleWrap, "Editor", "Toggle line wrapping", "Wrap or unwrap long editor lines.", ShortcutBinding{"W", key.ModShortcut}},
	{ShortcutEditorCopy, "Editor", "Copy text", "Copy the editor selection.", ShortcutBinding{"C", key.ModShortcut}},
	{ShortcutEditorCut, "Editor", "Cut text", "Cut the editor selection.", ShortcutBinding{"X", key.ModShortcut}},
	{ShortcutEditorPaste, "Editor", "Paste text", "Paste clipboard text into the editor.", ShortcutBinding{"V", key.ModShortcut}},
	{ShortcutCompletion, "Editor", "Show completions", "Open code and Typst suggestions.", ShortcutBinding{"P", key.ModShortcut}},
	{ShortcutFileCopy, "Files", "Copy file", "Copy the selected file or folder.", ShortcutBinding{"C", key.ModShortcut}},
	{ShortcutFileCut, "Files", "Cut file", "Cut the selected file or folder.", ShortcutBinding{"X", key.ModShortcut}},
	{ShortcutFilePaste, "Files", "Paste file", "Paste files into the selected folder.", ShortcutBinding{"V", key.ModShortcut}},
	{ShortcutSendPrompt, "Assistant", "Send prompt", "Send the current assistant message.", ShortcutBinding{key.NameEnter, key.ModShift}},
}

func ShortcutDefinitions() []ShortcutDefinition {
	return append([]ShortcutDefinition(nil), shortcutDefinitions...)
}

func ShortcutDefinitionFor(id ShortcutID) (ShortcutDefinition, bool) {
	for _, definition := range shortcutDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return ShortcutDefinition{}, false
}

func EffectiveShortcut(config *EditorSettings, id ShortcutID) (ShortcutBinding, bool) {
	definition, ok := ShortcutDefinitionFor(id)
	if !ok {
		return ShortcutBinding{}, false
	}
	if config != nil {
		if override, exists := config.ShortcutOverrides[id]; exists {
			if override.Disabled {
				return ShortcutBinding{}, false
			}
			if override.Binding != nil {
				return *override.Binding, true
			}
		}
	}
	return definition.Default, true
}

func ShortcutFilters(config *EditorSettings, focus event.Tag, ids ...ShortcutID) []event.Filter {
	var filters []event.Filter
	for _, id := range ids {
		if binding, enabled := EffectiveShortcut(config, id); enabled {
			filters = append(filters, binding.Filters(focus)...)
		}
	}
	return filters
}

func ShortcutMatches(config *EditorSettings, id ShortcutID, event key.Event) bool {
	binding, enabled := EffectiveShortcut(config, id)
	return enabled && event.State == key.Press && binding.Matches(event)
}

func ShortcutDisplay(config *EditorSettings, id ShortcutID) string {
	binding, enabled := EffectiveShortcut(config, id)
	if !enabled {
		return ""
	}
	return binding.Display()
}

func ShortcutHasOverride(config *EditorSettings, id ShortcutID) bool {
	if config == nil {
		return false
	}
	_, ok := config.ShortcutOverrides[id]
	return ok
}

func ShortcutCustomBinding(config *EditorSettings, id ShortcutID) (ShortcutBinding, bool) {
	if config == nil {
		return ShortcutBinding{}, false
	}
	override, ok := config.ShortcutOverrides[id]
	if !ok || override.Binding == nil {
		return ShortcutBinding{}, false
	}
	return *override.Binding, true
}

func ShortcutDisabled(config *EditorSettings, id ShortcutID) bool {
	if config == nil {
		return false
	}
	return config.ShortcutOverrides[id].Disabled
}

func (config *EditorSettings) SetShortcut(id ShortcutID, binding ShortcutBinding) error {
	definition, ok := ShortcutDefinitionFor(id)
	if !ok {
		return nil
	}
	config.ensureShortcutOverrides()
	override := config.ShortcutOverrides[id]
	override.Disabled = false
	if binding.Equal(definition.Default) {
		override.Binding = nil
	} else {
		value := binding
		override.Binding = &value
	}
	config.storeShortcutOverride(id, override)
	return config.Save()
}

func (config *EditorSettings) SetShortcutEnabled(id ShortcutID, enabled bool) error {
	config.ensureShortcutOverrides()
	override := config.ShortcutOverrides[id]
	override.Disabled = !enabled
	config.storeShortcutOverride(id, override)
	return config.Save()
}

func (config *EditorSettings) ResetShortcut(id ShortcutID) error {
	delete(config.ShortcutOverrides, id)
	return config.Save()
}

func (config *EditorSettings) ResetShortcuts() error {
	config.ShortcutOverrides = make(map[ShortcutID]ShortcutOverride)
	return config.Save()
}

func (config *EditorSettings) ensureShortcutOverrides() {
	if config.ShortcutOverrides == nil {
		config.ShortcutOverrides = make(map[ShortcutID]ShortcutOverride)
	}
}

func (config *EditorSettings) storeShortcutOverride(id ShortcutID, override ShortcutOverride) {
	if !override.Disabled && override.Binding == nil {
		delete(config.ShortcutOverrides, id)
		return
	}
	config.ShortcutOverrides[id] = override
}

func ShortcutConflict(config *EditorSettings, id ShortcutID, binding ShortcutBinding) (ShortcutDefinition, bool) {
	definition, ok := ShortcutDefinitionFor(id)
	if !ok {
		return ShortcutDefinition{}, false
	}
	for _, other := range shortcutDefinitions {
		if other.ID == id || other.Category != definition.Category {
			continue
		}
		otherBinding, enabled := EffectiveShortcut(config, other.ID)
		if enabled && binding.Equal(otherBinding) {
			return other, true
		}
	}
	return ShortcutDefinition{}, false
}

func (binding ShortcutBinding) Filters(focus event.Tag) []event.Filter {
	keyFilters := binding.KeyFilters(focus)
	filters := make([]event.Filter, len(keyFilters))
	for index, filter := range keyFilters {
		filters[index] = filter
	}
	return filters
}

func (binding ShortcutBinding) KeyFilters(focus event.Tag) []key.Filter {
	if binding.Name == "" {
		return nil
	}
	names := []key.Name{binding.Name}
	if binding.Name == key.NameEnter || binding.Name == key.NameReturn {
		names = []key.Name{key.NameEnter, key.NameReturn}
	}
	filters := make([]key.Filter, 0, len(names))
	for _, name := range names {
		filters = append(filters, key.Filter{Focus: focus, Name: name, Required: binding.Modifiers})
	}
	return filters
}

func (binding ShortcutBinding) Matches(event key.Event) bool {
	return sameShortcutName(binding.Name, event.Name) && binding.Modifiers == event.Modifiers
}

func (binding ShortcutBinding) Equal(other ShortcutBinding) bool {
	return sameShortcutName(binding.Name, other.Name) && binding.Modifiers == other.Modifiers
}

func (binding ShortcutBinding) Display() string {
	parts := make([]string, 0, 6)
	for _, modifier := range []struct {
		value key.Modifiers
		label string
	}{
		{key.ModCtrl, "Ctrl"},
		{key.ModCommand, "Cmd"},
		{key.ModShift, "Shift"},
		{key.ModAlt, "Alt"},
		{key.ModSuper, "Super"},
	} {
		if binding.Modifiers.Contain(modifier.value) {
			parts = append(parts, modifier.label)
		}
	}
	parts = append(parts, shortcutName(binding.Name))
	return strings.Join(parts, "+")
}

func ShortcutBindingFromEvent(event key.Event) (ShortcutBinding, bool) {
	switch event.Name {
	case "", key.NameCtrl, key.NameCommand, key.NameShift, key.NameAlt, key.NameSuper:
		return ShortcutBinding{}, false
	default:
		return ShortcutBinding{Name: event.Name, Modifiers: event.Modifiers}, true
	}
}

func sameShortcutName(left, right key.Name) bool {
	if left == right {
		return true
	}
	return (left == key.NameEnter || left == key.NameReturn) && (right == key.NameEnter || right == key.NameReturn)
}

func shortcutName(name key.Name) string {
	switch name {
	case key.NameEnter, key.NameReturn:
		return "Enter"
	case key.NameEscape:
		return "Esc"
	case key.NameDeleteBackward:
		return "Backspace"
	case key.NameDeleteForward:
		return "Delete"
	default:
		return string(name)
	}
}
