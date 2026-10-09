# Repo rules

## YAML validation

Validate YAML files with `yamllint`, not a custom Python script. Accepts a file or directory. Example: `yamllint path/to/file.yaml` or `yamllint path/to/dir`.

For each finding, evaluate before acting: fix it if it's a real issue, or exclude the rule in `.yamllint.yaml` if it's a false positive or not needed (e.g. noting why, like the GitHub Actions `on:` truthy warning).

## Markdown formatting

Never hand-wrap lines in markdown documents (no manual line breaks mid-paragraph at ~80 columns). Write each paragraph, list item, and table row as a single line; let the editor/viewer soft-wrap. Hand-wrapping breaks reflow and diffs badly.
