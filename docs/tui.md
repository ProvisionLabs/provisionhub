# Terminal interface

Home offers **Create Project**, **Plugins**, and **Settings**. Up/Down or `j`/`k` navigates menus; Enter selects; Esc returns; Ctrl+C quits. Text fields accept ordinary letters, including `q`.

Create Project collects a project name and destination, then a list of named components. Use `a` to add, `e` to edit, and `d` to remove one. Multiple components may use the same technology. Enter opens Review, where `p` shows a scrollable, read-only plan. Enter/Esc returns from preview. When every selected capability has a generator, Review shows **Generate Project**; Enter explicitly approves the generation. Otherwise Review reports that generation is unavailable and writes nothing.

The result screen reports generated files or an error. After a failed attempt, `p` previews a fresh plan and `g` performs read-only preflight; `y` confirms a retry, while `n` or Esc cancels. A generated project is not registered or tracked. The wizard does not create a project YAML file or plugin lockfile. Keyboard quit is ignored while generation is in progress to avoid interrupting publication.

**Plugins** discovers, installs, selects, updates, and removes managed official plugins. An Installed `s` selection chooses a version for the TUI session. **Settings** persists workspace and provider preferences and manages optional Git provider accounts. Both features retain their SQLite and credential-vault storage. See [plugin management](plugin-registry.md) and [settings](settings.md).

The TUI uses a compact layout for narrow terminals. See [generation planning](generation-planning.md) for output safety and [CLI reference](cli-reference.md) for noninteractive use.

After category, technology, and component name, configurable plugins open a
dynamic configuration screen. Fields appear in lexical key order with labels,
descriptions, defaults, and required/optional indicators. Strings and integers
use text inputs; boolean and enum fields use Left/Right/Space selectors.
Enter/Tab/Down validates and advances; the last field saves the component.
Up/Shift+Tab returns to a previous field after validation. Esc returns to the
name while preserving the draft; editing a saved component restores its values.
Ctrl+U clears an override, using the declared default or leaving an optional
property absent. A required field without a default must be supplied. Empty
string is a value when entered explicitly; integer fields cannot contain
fractional or incompatible text. Validation errors keep the active field open.

Schema loading runs asynchronously; Esc cancels the pending screen. Components
without fields return directly to the component list. Repeated technologies
have independent configuration. Review and the scrollable plan preview show
resolved values; use `p` to inspect complete summaries on small terminals.
Configuration screens keep the active control and error visible in compact
layouts. Values remain in memory for this creation session. See the
[configuration contract](plugin-configuration.md).
