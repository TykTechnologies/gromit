.PHONY: import

import:
	terraform import module.$(REPO).github_repository.repository $(REPO)
	terraform import module.$(REPO).github_branch.default $(REPO):$(BRANCH)
	terraform import module.$(REPO).github_branch_default.default $(REPO)