.PHONY: import

import:
	terraform import module.$(REPO).github_repository.repository $(REPO)
	terraform import module.$(REPO).github_branch.default $(REPO):$(BRANCH)
	terraform import module.$(REPO).github_branch_default.default $(REPO)

import_protection:
	terraform import 'module.tyk.module.protected_branches["$(BRANCH)"].github_branch_protection.automerge' $(REPO):$(BRANCH)

fixed_import_protection:
#tyk
	terraform import 'module.tyk.module.protected_branches["master"].github_branch_protection.automerge' tyk:master
#	terraform import 'module.tyk.module.protected_branches["release-2.9"].github_branch_protection.automerge' tyk:release-2.9
	terraform import 'module.tyk.module.protected_branches["release-3-lts"].github_branch_protection.automerge' tyk:release-3-lts
	terraform import 'module.tyk.module.protected_branches["release-3.2"].github_branch_protection.automerge' tyk:release-3.2 
#tyk-analytics
	terraform import 'module.tyk-analytics.module.protected_branches["master"].github_branch_protection.automerge' tyk-analytics:master
	terraform import 'module.tyk-analytics.module.protected_branches["stable"].github_branch_protection.automerge' tyk-analytics:stable
	terraform import 'module.tyk-analytics.module.protected_branches["target/cloud"].github_branch_protection.automerge' tyk-analytics:target/cloud
	terraform import 'module.tyk-analytics.module.protected_branches["target/stage"].github_branch_protection.automerge' tyk-analytics:target/stage
#tyk-pump
	terraform import 'module.tyk-pump.module.protected_branches["master"].github_branch_protection.automerge' tyk-pump:master
	terraform import 'module.tyk-pump.module.protected_branches["stable"].github_branch_protection.automerge' tyk-pump:stable
	terraform import 'module.tyk-pump.module.protected_branches["target/cloud"].github_branch_protection.automerge' tyk-pump:target/cloud
	terraform import 'module.tyk-pump.module.protected_branches["target/stage"].github_branch_protection.automerge' tyk-pump:target/stage
#tyk-identity-broker
	terraform import 'module.tyk-identity-broker.module.protected_branches["master"].github_branch_protection.automerge' tyk-identity-broker:master

validate:
	terraform fmt
	terraform validate
	terraform fmt -recursive ../../../modules

clean:
	rm -rf ./.terraform*