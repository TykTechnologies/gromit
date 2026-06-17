## Description

<!-- Please include a summary of the changes and the motivation/context for this pull request. -->

**Jira Ticket:** [TT-XXXX](https://tyktech.atlassian.net/browse/TT-XXXX)

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Chore / documentation / template sync

## Verification & Testing

<!-- Please describe the tests that you ran to verify your changes. -->

- [ ] I have run local unit tests (`go test ./policy/...`) and they passed.
- [ ] I have generated policy outputs locally (`go run . policy gen`) and verified the rendered files.
- [ ] I have verified the changes in target downstream repositories (e.g. `tyk`, `tyk-analytics`).

### Command(s) used for testing:

```bash
# Example: go run . policy gen /tmp/output --repo tyk --branch master
```

## Checklist

- [ ] My code follows the style guidelines of this project
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
