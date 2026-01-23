## Update golangci-lint version

The current configuration sets `version: v1.59.0`, but `golangci-lint-action@v9` only supports versions >= v2. Please update this version to `v2.0.0` or a supported latest v2 release to resolve the workflow failure.

### Reference
- Workflow file: [.github/workflows/golangci-lint.yml](https://github.com/PiedroErnikioilis/invoice_manager/blob/e1f340cb1cbd3125c373e9bbf68cece6e83159db/.github/workflows/golangci-lint.yml)