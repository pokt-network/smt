#############################################
##          Configuration variables        ##
#############################################

GH_WORKFLOWS := .github/workflows

#####################################
##       CI/CD Workflow Testing    ##
#####################################

.PHONY: check_secrets
# Internal helper: Check if .secrets file exists with valid GITHUB_TOKEN
check_secrets:
	@if [ ! -f .secrets ]; then \
		echo "❌ .secrets file not found!"; \
		echo "Please create a .secrets file with your GitHub token:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		exit 1; \
	fi
	@if ! grep -q "GITHUB_TOKEN=" .secrets; then \
		echo "❌ GITHUB_TOKEN not found in .secrets file!"; \
		echo "Please add GITHUB_TOKEN to your .secrets file:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		echo "You can create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi
	@if grep -q "GITHUB_TOKEN=$$" .secrets || grep -q "GITHUB_TOKEN=\"\"" .secrets || grep -q "GITHUB_TOKEN=''" .secrets; then \
		echo "❌ GITHUB_TOKEN is empty in .secrets file!"; \
		echo "Please set a valid GitHub token:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		echo "You can create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi

.PHONY: install_act
install_act: ## Install act for local GitHub Actions testing
	@echo "Installing act..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		brew install act; \
	else \
		curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash; \
	fi
	@echo "✅ act installed successfully"

###########################
###   Release Helpers   ###
###########################

# Common variables
GITHUB_REPO_URL := https://github.com/pokt-network/smt/releases/new

define print_next_steps
	$(call print_info_section,Next Steps)
	@echo "$(BOLD)1.$(RESET) Push the new tag:"
	@echo "   $(CYAN)git push origin $(1)$(RESET)"
	@echo ""
	@echo "$(BOLD)2.$(RESET) Draft a new release:"
	$(call print_url,$(GITHUB_REPO_URL))
	$(if $(2),@echo "   $(CYAN)- Mark it as a pre-release$(RESET)")
	$(if $(2),@echo "   $(CYAN)- Include PR/branch information in the description$(RESET)")
	@echo ""
endef

define print_cleanup_commands
	$(call print_info_section,If you need to delete the tag)
	@echo "$(BOLD)Local:$(RESET)"
	@echo "   $(CYAN)git tag -d $(1)$(RESET)"
	@echo "$(BOLD)Remote:$(RESET)"
	@echo "   $(CYAN)git push origin --delete $(1)$(RESET)"
	@echo ""
endef

define print_additional_info
	$(call print_info_section,Additional Information)
	@echo ""
endef

.PHONY: release_tag_dev
release_tag_dev: ## Tag a new dev release for unmerged PRs (e.g. v0.14.0-dev-feat-xyz, v0.14.0-dev-pr-123)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval CURRENT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD))
	@$(eval SHORT_COMMIT=$(shell git rev-parse --short HEAD))
	@if [ "$(CURRENT_BRANCH)" = "main" ] || [ "$(CURRENT_BRANCH)" = "master" ]; then \
		$(call print_warning,Cannot create dev tag from main/master branch. Switch to a feature branch first.); \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		$(call print_warning,Working directory has uncommitted changes.); \
		read -p "Continue anyway? (y/N): " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "Aborted."; \
			exit 1; \
		fi; \
	fi
	@$(eval BRANCH_CLEAN=$(shell echo $(CURRENT_BRANCH) | sed 's/[^a-zA-Z0-9-]/-/g' | sed 's/--*/-/g' | sed 's/^-\|-$$//g'))
	@$(eval NEW_TAG=$(LATEST_TAG)-dev-$(BRANCH_CLEAN)-$(SHORT_COMMIT))
	@git tag $(NEW_TAG)
	$(call print_success,Dev version tagged: $(NEW_TAG))
	@echo "$(BOLD)Branch:$(RESET) $(CYAN)$(CURRENT_BRANCH)$(RESET)"
	@echo "$(BOLD)Commit:$(RESET) $(CYAN)$(SHORT_COMMIT)$(RESET)"
	@echo ""
	$(call print_next_steps,$(NEW_TAG),pre-release)
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_rc
release_tag_rc: ## Tag a new rc release (e.g. v0.14.0 -> v0.14.0-rc1, v0.14.0-rc1 -> v0.14.0-rc2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval EXISTING_RC_TAG=$(shell git tag --sort=-v:refname | grep "^$(LATEST_TAG)-rc[0-9]*$$" | head -n 1))
	@$(eval NEW_TAG=$(shell \
		if [ -z "$(LATEST_TAG)" ]; then \
			echo "No stable version tags found" >&2; \
			exit 1; \
		elif [ -z "$(EXISTING_RC_TAG)" ]; then \
			echo "$(LATEST_TAG)-rc1"; \
		else \
			RC_NUM=$$(echo "$(EXISTING_RC_TAG)" | sed 's/.*-rc\([0-9]*\)$$/\1/'); \
			NEW_RC_NUM=$$((RC_NUM + 1)); \
			echo "$(LATEST_TAG)-rc$$NEW_RC_NUM"; \
		fi))
	@git tag $(NEW_TAG)
	$(call print_success,RC version tagged: $(NEW_TAG))
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_minor
release_tag_minor: ## Tag a new minor release (e.g. v0.14.0 -> v0.14.1)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. -v OFS=. '{ $$NF = sprintf("%d", $$NF + 1); print }'))
	@git tag $(NEW_TAG)
	$(call print_success,Bug fix version tagged: $(NEW_TAG))
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_major
release_tag_major: ## Tag a new major release (e.g. v0.14.0 -> v0.15.0)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. '{$$2 += 1; $$3 = 0; print $$1 "." $$2 "." $$3}'))
	@git tag $(NEW_TAG)
	$(call print_success,Minor release version tagged: $(NEW_TAG))
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)