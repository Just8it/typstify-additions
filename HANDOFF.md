# Handoff: Focused file interface for students

## Worktree and branch

- Open this folder as the workspace: `C:\Users\jugri\Documents\Programieren\Typstify\feature-new-file-interface-for-students`
- Use the branch `feature/new-file-interface-for-students`.
- The worktree and branch already exist from `upstream/main`. Verify with `git branch --show-current`; do not create a second branch with a different name.

## User problem

Typstify's current file tree works for one or two small projects, but becomes noisy when the workspace is a large study or lecture archive such as `_Uni`.

The user wants navigation closer to Goodnotes or a normal file explorer: show one folder at a time instead of recursively expanding the entire hierarchy.


design studie pls use this:

Import the attached im-currently-wanting-to-implemt-an-goodnotes-style-file-mana.zip — read the README inside.
Implement: Typstify Library.dc.html

Importaint!!!!! keep to the typstify design language, and toolchain, this is just a studie, how it should broughtly look feel and work.


old but usfull as fallback if quetions arrise, if however you find you selfe needing to fall back here, ask the user whats up.

Example:

1. The selected workspace root is `_Uni`.
2. The first view shows only its immediate child folders, such as subjects or lectures.
3. Clicking a subject replaces the view with only that folder's immediate children.
4. The user can navigate back or upward without seeing every descendant at once.

## Required behavior

- Add a clean **focused browser** mode to the file navigation.
- In focused mode, render only the current directory's immediate folders and files; never flatten or expand descendants into the same view.
- Clicking a folder navigates into it and replaces the visible contents.
- Provide an obvious Back/Up action or breadcrumb path.
- Clicking a file must keep using Typstify's existing file-opening flow.
- Show folders before files and reuse the existing filtering/skip behavior where possible.
- Add a simple toggle between the new focused browser and the existing recursive tree. Do not remove the existing tree.
- Keep the interface quiet and readable for large roots such as `_Uni`; avoid showing unrelated nested files.

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

- Opening `_Uni` shows only `Mathematics` and `Mechanics`.
- Clicking `Mechanics` shows only `Lecture 01`.
- Clicking `Lecture 01` shows `notes.typ`.
- Navigating up returns to the previous folder.
- Switching to classic tree mode restores the existing recursive explorer behavior.

## Likely starting points

- `ui/navpanel/filetree.go` owns the current navigation panel and file selection flow.
- `widgets/filetree/tree.go` owns the recursive/flattened tree UI and filesystem operations.
- `ui/navpanel/navdrawer.go` registers the File Explorer section.
- `service/workspace.go` stores the current workspace and tree state.

Reuse the current file-opening and filesystem operations. Do not build a second workspace service or duplicate file mutation logic.

## Scope boundary

The first version does not need tags, search, cloud sync, course metadata, thumbnails, or a database. The filesystem hierarchy is the organization model.

Before implementation, trace the current folder-selection and file-opening flow end to end, then make the smallest change that supports the acceptance example. Add one focused runnable test for any new navigation-state logic.
