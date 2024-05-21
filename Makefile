TARGETS := $(shell ls scripts)

$(TARGETS):
	./scripts/$@

.DEFAULT_GOAL := default

# Charts Build Scripts

pull-scripts:
	./scripts/charts-build-scripts/pull-scripts

rebase:
	./scripts/charts-build-scripts/rebase

CHARTS_BUILD_SCRIPTS_TARGETS := prepare patch clean clean-cache charts list index unzip zip standardize template

$(CHARTS_BUILD_SCRIPTS_TARGETS):
	@./scripts/charts-build-scripts/pull-scripts
	@./bin/charts-build-scripts $@

.PHONY: $(TARGETS) $(CHARTS_BUILD_SCRIPTS_TARGETS)