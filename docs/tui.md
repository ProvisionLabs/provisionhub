# Terminal interface

Home offers **Create Project**, **My Projects**, **Plugins**, and **Settings**. Up/Down or `j`/`k` moves through menus; Enter selects; Esc returns; Ctrl+C quits. Text fields accept ordinary letters, including `q`.

Create Project collects a name and destination, then named components. `a`, `e`, and `d` add, edit, and remove components. `p` previews the read-only plan on Review; Enter explicitly saves the manifest and, when executable, generates files. The created manifest is registered for **My Projects** even if generation later fails. The result screen can create a plugin lock with `l` or preview a fresh plan with `p`.

In **My Projects**, Up/Down selects a saved reference. Enter opens details; `r` registers an existing manifest or project directory; `m` relocates the selected reference; `f` asks to Forget Project; `c` opens Cleanup Unavailable Projects. Long paths are shortened in the list; full paths and filesystem errors appear in details. A missing or invalid manifest stays visible until relocated or explicitly forgotten. Details read the current manifest and lockfile, show components, declared connections, lock versions, and diagnostics. `p` views a plan, `l` manages the lock, `o` opens the project directory, and Esc returns to the list.

Cleanup starts with no candidates selected. Use Space to toggle one, `a` for all, `x` for none, and Enter to review the selection. `y` confirms removal of only the selected SQLite registrations; `n` or Esc cancels. Available projects cannot be cleaned up. No project files are deleted. See [saved project management](project-management.md) for recovery choices.

The saved-project plan lists operations and diagnostics. `g` checks the destination and asks for approval; `y` applies; `n` or Esc cancels. Apply uses the same guarded executor as `project apply`. The lock screen uses `v` to validate, `c` to preview creation, `u` to preview an update, and `r` to restore missing official artifacts. A preview never writes; Enter then `y` confirms a lock change. Scroll longer screens with Up/Down or `j`/`k`.

See [saved project management](project-management.md), [generation planning](generation-planning.md), and [project plugin locking](project-plugin-locking.md).
