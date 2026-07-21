# Student File Interface - Source of Truth

> **Non-negotiable:** The Student Library, contextual editor explorer, and redesigned navigation/menu are toggleable replacements. The Classic interface remains fully available and must not be deleted, degraded, or have its saved state overwritten.
>
> Update this document whenever a product or implementation decision changes. It takes precedence over `HANDOFF.md` and the visual prototype when they disagree.

## Goal

Make large study workspaces such as `_Uni` calm and navigable without recursively exposing every descendant. The filesystem remains the organization model; no database, tags, cloud model, or course metadata is required.

The original design study is the archive [`im currently wanting to implemt an goodnotes style file manager fo typstify, i w-handoff.zip`](<./im currently wanting to implemt an goodnotes style file manager fo typstify, i w-handoff.zip>). Its primary artifact is `project/Typstify Library.dc.html`. Option **2a** defines the Library interaction model and option **2b** defines the contextual editor sidebar. Options 1a-1c are earlier studies only.

## Product model

### Student Library

- Use the main content area for a focused file manager.
- Show only the current directory's immediate folders and files.
- Opening a folder replaces the visible contents with that folder's children.
- Provide an obvious breadcrumb and Up/Back navigation.
- Show folders before files.
- Hide dot-prefixed directories such as `.git`, `.claude`, and `.typstify` from the Student Library. This presentation rule does not hide them from the Classic or contextual editor explorers.
- Grid and list are two presentations of the same focused directory.
- Provide a Library filter menu with All, Typst work, Notebooks, Typst documents, PDFs, Images, and Other files. Typst work is the session default. Filtering changes only the visible immediate children and is session-local. Ordinary subfolders remain visible under every filter so navigation does not dead-end; Typst work includes subfolders, marked projects, and `.typ` documents.
- New Project launched from the Library creates its new project folder inside the currently focused Library directory. The dialog shows that inherited destination but does not ask the user to choose it. After creation, keep the existing Library workspace/root, open the new project's resolved Typst entrypoint, and show its contextual project drawer. New Project launched from a global navigation action still provides the normal location picker and may switch the workspace to the created project because no focused Library destination is implied.
- Opening a file must use Typstify's existing editor, viewer, or external-app flow.
- Opening an ordinary file from the Library opens the editor with the secondary drawer closed. Explorer is available on demand through the rail's Files action or `Ctrl+D` and temporarily scopes to that file's direct parent folder, even when Whole workspace is the saved preference.
- In Student mode, pasting a supported image or PDF into a loose `.typ` document promotes that document into a notebook. Create a sibling directory named after the document, move the document into it, add a `typst.toml` whose entrypoint is that document, and place the asset in the notebook's `images/` or `pdf/` directory. Refuse safely if the destination directory already exists; never overwrite it.
- When pasting into an existing notebook, use the nearest ancestor containing a direct `typst.toml` as the asset root. Nested notebooks therefore own their assets independently. Store images in that root's `images/` directory and PDFs in its `pdf/` directory, then insert a path relative to the active document. Never use the outer Library root as the asset root.
- After promotion, keep the Library workspace unchanged, replace the stale loose-file tab with the promoted document, and open its contextual project drawer. In Classic mode, retain the existing behavior that treats the opened workspace root as the project root.
- If an ordinary file belongs to a marked project, opening it is a project transition: the contextual project drawer opens automatically and shows the complete project.
- In Student mode, opening a file from the native whole-workspace tree applies the same project-or-loose contextual scope. The already-open drawer remains visible. In Classic mode, native tree clicks remain unchanged.
- In Student mode, selecting a native-tree folder containing a direct `typst.toml` opens its resolved notebook entrypoint and focuses the project. Ordinary folders retain their normal expand/collapse behavior, and Classic mode is unchanged.
- The Library is a dedicated home surface owned by the main window, not an editor tab.
- In Student mode, opening or switching a workspace shows the Library first while restored editor tabs remain available behind it.
- The Library hides the editor tab bar while active. Opening a file or selecting the editor/explorer returns to the existing tabs.
- A directory containing `typst.toml` directly inside it is presented as a notebook. It remains a normal filesystem directory, but opening its cover opens the manifest entrypoint in the existing editor, reveals the contextual project drawer, and scopes it to the complete notebook directory.
- Opening a notebook cover is an explicit temporary project-focus action even when the saved editor-explorer preference is Whole workspace. It does not overwrite that preference; opening a file outside the notebook returns to the saved behavior.
- Resolve a notebook entry file from `[package].entrypoint` in `typst.toml`, then fall back to `main.typ`, then the first direct `.typ` file. If no Typst file exists, navigate into the directory instead.

### Contextual editor explorer

- While editing a project document, scope the sidebar tree to that project.
- For a loose `.typ` file, scope it to the containing folder.
- Keep expandable tree behavior inside that smaller scope.
- Show the active scope and provide an ancestor/"Go up to" control.
- Match design-study option 2b: show an `EXPLORER` header, then a project/folder scope card, then the scope's children without duplicating the root row. Highlight the active file.
- Keep the Classic whole-workspace tree available as an alternative.

### Classic interface

- Preserve the current recursive workspace tree, navigation, menu, file operations, expansion state, and open-file restoration.
- Switching to the Student interface must not discard Classic tree state.
- Switching back must restore the familiar Classic experience.

## Settings

Add a dedicated **File Interface** settings tab. Persist settings through Typstify's existing settings service and apply them through the existing `settings.updated` flow.

- **Interface mode:** Classic (default) / Student
- **Library folder:** persisted startup folder for Student mode, with Change and Forget actions. Forget removes only the saved preference and returns the Library home to its folder prompt; it never deletes or alters the folder.
- **Editor explorer:** Contextual project or folder (default) / Whole workspace
- **Default Library view:** Grid (default) / List
- **Navigation/menu layout:** Library-style (default) / Classic

Classic is the backwards-compatible master default. In Classic mode, Student preferences do not alter the current interface. Settings apply live through `settings.updated` and do not require a restart. The settings control presentation only; both interfaces operate on the same workspace and filesystem.

The settings page presents the master interface choice first. When Classic is selected, hide the inactive Student preferences while retaining their saved values. When Student is selected, show editor scope, Library view, and navigation layout together as one clearly labeled Student preferences group.

Keep navigation state separate:

- Classic retains its expanded-node snapshot.
- Student Library retains its focused directory separately for each workspace. Grid/list is a persisted File Interface preference.
- A missing focused directory falls back to its closest existing ancestor inside the workspace, then the workspace root.
- A workspace switch selects that workspace's valid focused directory without changing Classic expansion state.

In Student mode, Library-style navigation is the default: a narrow persistent rail is the stable navigation anchor, and a secondary resizable drawer appears only beside the editor when requested or when a marked project is opened. The rail exposes Library, Explorer, Outline, Assistant Sessions, Open Folder, New Project, Packages, and Settings. If Classic navigation is selected, the existing drawer and bottom menu remain and gain only a Library entry point.

### Student shell transitions

- **Workspace opened or Library selected:** show the Library with the rail only. Close the secondary drawer and keep restored editor tabs available behind the Library.
- **Student mode without a workspace:** show a clear Library-folder prompt with an Open Folder action using the existing workspace picker. A selected folder must become the active workspace even while the drawer is hidden. The prompt is the empty Library home, not a global overlay: Settings, Packages, and other explicitly opened views must remain accessible.
- **Notebook cover or document inside a marked project opened:** show the editor and automatically open the contextual project drawer.
- **Loose document opened:** show the editor with the secondary drawer closed. Its containing folder is prepared as the contextual scope but is shown only after Files or `Ctrl+D` is pressed.
- **Files, Outline, or Assistant selected:** leave the Library, retain the active editor tab, and show the selected secondary drawer. Selecting the active drawer action again closes it.
- **Library selected from the editor:** return to the Library and close the secondary drawer without closing editor tabs.
- **Tab switched:** remain in the editor; update an already-visible contextual scope without forcing a closed drawer open.

When the Student interface uses Classic navigation, a Library/Home action also appears in the editor drawer header. It provides the fast return-home route otherwise supplied by the Library-style rail. This extra header action is not shown with the rail or in the Classic interface. Hiding any Student-only menu action must not leave an empty slot or extra spacing.

The Library must never render beside the native folder drawer. These transitions affect only Student mode; Classic mode retains its current sidebar, bottom menu, shortcuts, and file-opening behavior.

## Visual contract

The prototype supplies information architecture and interaction ideas, not a replacement design system.

- Keep the Gio toolchain and Typstify's existing window, tab, editor, status-bar, menu, popup, tooltip, and resizing conventions.
- Tooltips must have enough horizontal space for their label and must not collapse to the width of a rail icon.
- Treat breadcrumbs as primary Library navigation: use readable text, clear separators, and comfortably sized Up and segment targets rather than caption-sized path text.
- Use the active Typstify theme's semantic colors (`Bg`, `Bg2`, `Fg`, `ContrastBg`, hover and selection alpha). Never hardcode the prototype's dark palette or cyan accent.
- Respect the configured typeface and text size; do not import the prototype's IBM Plex fonts.
- Reuse Typstify's existing SVG icon set and interaction states.
- Make the layout responsive down to the current `960 x 640` minimum window size; the prototype's fixed dimensions are illustrative.
- Keep the interface quiet: restrained borders, theme-derived accents, compact controls, and no unrelated descendants.
- Follow design-study 2a's object hierarchy in grid view: notebooks read as substantial covers, while the surrounding card stays quiet and gains emphasis only through hover or selection.

The sidebar and menu structure may change in Student mode, but their styling must still look native to the currently selected Typstify theme.

## Reuse requirements

Do not create a second workspace service or duplicate filesystem mutations.

- Reuse the existing workspace switch and snapshot lifecycle.
- Reuse the existing file-opening routing for Typst, images, text, and external files.
- Reuse current create, rename, delete, copy/cut/paste, drag-and-drop, context-menu, filesystem-watcher, marker, and extra-menu behavior.
- Reuse the existing direct-child loading and hidden-file filtering behavior.
- Add only the smallest shared API needed for both presentations to operate on the same file nodes.

## First implementation scope

Required:

- File Interface settings tab and persisted toggles.
- Student Library focused navigation with breadcrumbs/Up.
- Grid/list presentation with folders before files.
- Contextual editor explorer.
- Classic interface fallback with preserved state.
- Existing file operations and file-opening behavior in both modes.
- One focused runnable test for new navigation-state logic.

Not required yet:

- Favorites, shared files, cloud sync, storage quotas, tags, indexed search, thumbnails, course metadata, or a database.
- Pixel-for-pixel copying of Goodnotes or the HTML prototype.

## Acceptance example

Given:

```text
_Uni/
  Mathematics/
    Analysis/
      notes.typ
  Mechanics/
    Lecture 01/
      notes.typ
```

In Student Library mode:

1. Opening `_Uni` shows only `Mathematics` and `Mechanics`.
2. Opening `Mechanics` replaces them with `Lecture 01`.
3. Opening `Lecture 01` shows `notes.typ`.
4. Breadcrumb or Up returns to the previous folder.
5. Opening `notes.typ` uses the existing Typst editor flow.
6. Switching to Classic restores the existing recursive explorer and its prior expansion state.

## Notebook definition

A notebook is a normal directory containing a direct child named `typst.toml`. Nested markers do not classify an ancestor as a notebook. Directories without this marker use the ordinary folder presentation. Opening a notebook cover opens its resolved entry file and automatically shows the notebook's complete tree in the contextual editor explorer. A loose document opens without that drawer and reveals its containing folder only on demand.
