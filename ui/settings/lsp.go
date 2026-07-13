package settings

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/lsp"
	"looz.ws/typstify/service/settings"
)

type LspSettingsView struct {
	setting                    *settings.LspSettings
	lspVersion                 string
	openInBrowser              widget.Bool
	enableLspLogs              widget.Bool
	enablePowerSaving          widget.Bool
	enablePartialRenderPreview widget.Bool
	isInitialized              bool
	lastErr                    error
}

func (l *LspSettingsView) Layout(gtx C, th *theme.Theme) D {

	if !l.isInitialized {
		l.lspVersion = lsp.Version()
		l.openInBrowser = widget.Bool{Value: l.setting.OpenPreviewInBrowser != 0}
		l.enableLspLogs = widget.Bool{Value: l.setting.EnableLSPLogs != 0}
		l.enablePowerSaving = widget.Bool{Value: l.setting.EnablePowerSaving != 0}
		l.enablePartialRenderPreview = widget.Bool{Value: l.setting.EnablePartialRenderPreview}

		l.isInitialized = true
	} else {
		var doUpdate bool

		if l.enableLspLogs.Update(gtx) {
			if l.enableLspLogs.Value {
				l.setting.EnableLSPLogs = 1
			} else {
				l.setting.EnableLSPLogs = 0
			}
			doUpdate = true
		}

		if l.enablePowerSaving.Update(gtx) {
			if l.enablePowerSaving.Value {
				l.setting.EnablePowerSaving = 1
			} else {
				l.setting.EnablePowerSaving = 0
			}
			doUpdate = true
		}

		if l.openInBrowser.Update(gtx) {
			if l.openInBrowser.Value {
				l.setting.OpenPreviewInBrowser = 1
			} else {
				l.setting.OpenPreviewInBrowser = 0
			}

			doUpdate = true
		}

		if l.enablePartialRenderPreview.Update(gtx) {
			l.setting.EnablePartialRenderPreview = l.enablePartialRenderPreview.Value
			doUpdate = true
		}

		if doUpdate {
			l.lastErr = l.setting.Save()
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if l.lastErr != nil {
				return misc.LayoutErrorLabel(gtx, th, l.lastErr)

			} else {
				return layout.Dimensions{}
			}
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Version"),
				"",
				func(gtx C) D {
					return material.Label(th.Theme, th.TextSize, l.lspVersion).Layout(gtx)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Preview In Browser"),
				i18n.Translate("When checked, document preview will be opening in your default browser. Otherwise the preview will use built-in previewer."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &l.openInBrowser, "Open in browser").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Enable Partial Preview"),
				i18n.Translate("The previewer will only render pages inside the viewport if enabled. This improves performance especially for large document."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &l.enablePartialRenderPreview, "Enable partial rendering").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Debug Log"),
				i18n.Translate("When checked, logs from the built-in LSP (Language Server Procotol) server is written to the console panel. It needs to restart or reload to take effect."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &l.enableLspLogs, "Enable debug log").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Power Saving"),
				i18n.Translate(`When checked, LSP server runs in power saving mode, only basic syntax checking and code completion are avaliable, diagnostics and previewing will not work. It needs to restart or reload to take effect.`),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &l.enablePowerSaving, "Enable power saving").Layout),
					)
				})
		}),
	)
}

func (l *LspSettingsView) Title() string { return i18n.Translate("Language Server") }
