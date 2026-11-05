# Terminal interface

Home offers **Create Project**, **Plugins**, and **Settings**. Up/Down or `j`/`k` navigates menus; Enter selects; Esc returns; Ctrl+C quits. Text fields accept ordinary letters, including `q`.

Create Project collects a project name and destination, then a list of named components. Use `a` to add, `e` to edit, and `d` to remove one. Multiple components may use the same technology. Enter opens Review, where `p` shows a scrollable, read-only plan. Enter/Esc returns from preview. When every selected capability has a generator, Review shows **Generate Project**; Enter explicitly approves the generation. Otherwise Review reports that generation is unavailable and writes nothing.

The result screen reports generated files or an error. After a failed attempt, `p` previews a fresh plan and `g` performs read-only preflight; `y` confirms a retry, while `n` or Esc cancels. A generated project is not registered or tracked. The wizard does not create a project YAML file or plugin lockfile. Keyboard quit is ignored while generation is in progress to avoid interrupting publication.

**Plugins** discovers, installs, selects, updates, and removes managed official plugins. An Installed `s` selection chooses a version for the TUI session. **Settings** persists workspace and provider preferences and manages optional Git provider accounts. Both features retain their SQLite and credential-vault storage. See [plugin management](plugin-registry.md) and [settings](settings.md).

The TUI uses a compact layout for narrow terminals. See [generation planning](generation-planning.md) for output safety and [CLI reference](cli-reference.md) for noninteractive use.
