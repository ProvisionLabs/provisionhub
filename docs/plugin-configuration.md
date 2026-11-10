# Plugin configuration contract and authoring

ProvisionHub core compatibility version 0.2.0 adds an optional `configuration`
mapping to `provisionhub.dev/plugin/v1alpha1`. Plugins without it still work.
Older cores reject this new field; publish a new plugin version with
`minimumCoreVersion: 0.2.0` in its registry release metadata. Never replace a
published version or checksum. Package layout and integrity checks are unchanged.

## Schema

```yaml
configuration:
  properties:
    title:
      type: string
      label: Application title
      description: Visible heading in the starter page.
      default: Welcome
      minLength: 1
      maxLength: 120
    enabled:
      type: boolean
      default: true
    port:
      type: integer
      default: 3000
      minimum: 1
      maximum: 65535
    style:
      type: enum
      options: [simple, detailed]
      required: true
      default: simple
```

`properties` is a flat mapping. Keys are case sensitive, start with an ASCII
letter, and contain at most 63 ASCII letters, digits, or underscores. Keys are
the stable API; changing a label does not change a key. Each property requires
`type`. `label`, `description`, `default`, and `required` are optional.

| Type | Values and constraints |
| --- | --- |
| `string` | UTF-8 text; optional inclusive `minLength`/`maxLength` count Unicode code points |
| `boolean` | `true` or `false` |
| `integer` | Signed 64-bit integer; optional inclusive `minimum`/`maximum` |
| `enum` | String matching one of the nonempty, unique strings in `options` |

Omitted `required` means false. An omitted property receives its declared
default, including `false`, `0`, and `""`. A required property can have a default.
Optional properties without defaults remain absent. Explicit null is invalid.
Defaults must pass the same type, range, length, and enum checks as user values.
There is no conversion of strings to booleans or integers inside the resolver.

Unknown fields, duplicate YAML keys, aliases in configuration declarations,
unsupported types, inappropriate constraints, negative lengths, inverted bounds,
invalid enum choices, and incompatible defaults are rejected during loading.
Enum options and string defaults must be YAML strings: quote numeric or boolean
looking text. Bounds must be YAML integers. Nested objects, expressions,
conditional forms, regexes, and validation scripts are unsupported.

## Resolution and input

`internal/configuration` owns schema validation, typed text parsing, and pure
resolution. Resolution returns a new scalar map for each component without I/O
or mutating plugin definitions or caller values. Unknown user keys are checked
in lexical order, followed by declared keys in lexical order. It reports the
first error per component, identifying the component and configuration key.
The planner sorts components by name and collects their diagnostics before
contributing any operations. Invalid configuration makes the plan invalid.

CLI `project plan` and `project apply` accept repeated
`--set component.key=value`. The schema determines the type: strings and enum
choices are literal text, booleans must be exactly `true` or `false`, and
integers must be base-10 signed 64-bit values. Shell quoting protects spaces;
quotes removed by the shell are not part of the value. Only the first `=` is a
separator, and commas remain part of a string. Duplicate settings, even with
identical values, and unknown component names are errors.

The TUI loads the selected plugin schema in a Bubble Tea command, displays
fields in lexical key order, and uses the same parser and resolver. See the
[TUI guide](tui.md) for controls and optional values. Configuration belongs to
each named component, so repeated uses of a technology remain independent.
It exists only in the in-memory specification and review plan.

## Template access and escaping

`template` files retain `{{projectName}}` and `{{componentName}}`.
`{{component.name}}` exposes the instance name. New references are:

| Reference | Output |
| --- | --- |
| `{{config.title\|json}}` | Complete JSON string literal, including quotes; also suitable as a JavaScript/TypeScript string literal |
| `{{config.title\|html}}` | Escaped HTML text (or quoted HTML attribute content) |
| `{{config.enabled}}` | Literal `true` or `false` |
| `{{config.port}}` | Decimal integer |

Enum values are strings and require `json` or `html`. These modes are also
accepted for booleans and integers. No other filters, whitespace variants,
functions, loops, includes, or conditionals are supported. Use a JSON literal
as a whole value, without adding surrounding quotes. For example:

```text
{"title": {{config.title|json}}, "port": {{config.port}}}
const title = {{config.title|json}};
<h1>{{config.title|html}}</h1>
```

For JSX text use a JavaScript expression with a JSON literal, for example
`<h1>{ {{config.title|json}} }</h1>`. The modes are context specific; they do not
provide escaping for shell commands, CSS, or arbitrary embedded languages.
Configuration is never expanded into source or output paths.

Undeclared references, unsupported filters, and malformed delimiters fail plugin
loading. References to absent optional values fail planning with the component
and source file named. User strings containing template delimiters remain
literal data; expansion is a single pass. `copy` files remain byte-for-byte.
The planner renders in memory to validate references and record content hashes;
it writes no output and runs no commands. Apply reloads inputs, resolves and
renders with the same functions, and requires the reviewed plan to match.

## Author workflow

Declare only options that affect generated files. Keep defaults compatible with
existing generation. Select an escaping mode for each string context, then test
defaults, overrides, repeated components, invalid input, and generated source.
Use the local `--plugins` directory while developing. Build packages using
`cmd/pluginpack` and test managed installation separately. Run the generated
applications' own build checks explicitly; ProvisionHub never installs or starts
them. See the [manifest specification](architecture/plugins.md),
[planning guide](generation-planning.md), and
[official examples](https://github.com/ProvisionLabs/provisionhub-official-plugins).
