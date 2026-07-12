# SimWorkShare v0.4 delivery package

This package contains implementation-ready artifacts for SimWorkShare v0.4.

Files:

- `legacy_v0_3/` - unchanged uploaded v0.3 spec and default config kept for compatibility reference.
- `SimWorkShare_v0_4_specification_implementation_ready.docx` - formatted specification.
- `SimWorkShare_v0_4_specification_implementation_ready.pdf` - PDF export for review and sharing.
- `SimWorkShare_v0_4_specification_implementation_ready.md` - source Markdown version.
- `default_config_v0_4.json` - full default configuration with units and scenario definitions.
- `config_schema_v0_4.json` - JSON Schema for strict validation of the v0.4 config structure.
- `parameter_catalog_v0_4.csv` - parameter catalog: name, type, unit, range, default, meaning.
- `required_tests_v0_4.json` - mandatory test manifest and expected outcomes.
- `migration_map_v0_3_to_v0_4.csv` - field-level migration map from v0.3 to v0.4.
- `new_assumptions_v0_4.csv` - new assumptions introduced in v0.4 and their impact.
- `validation_report_v0_4.json` - validation report for the default v0.4 config.

Primary implementation entry points:

1. Validate `default_config_v0_4.json` against `config_schema_v0_4.json`.
2. Implement monthly calculation order from the specification, section 7.
3. Add invariant checks from section 18.
4. Run the tests from `required_tests_v0_4.json`.
5. Use `compatibility_v0_3` only to reproduce legacy v0.3 behavior.

Integrity note: `MANIFEST.sha256` intentionally does not list itself. A file
cannot contain a stable SHA-256 digest of its own complete contents; all other
delivery-package entries remain covered by the manifest.
