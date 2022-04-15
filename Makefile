TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-$$(uname -s)-$$(uname -m) > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper $@

.DEFAULT_GOAL := default

# Charts Build Scripts

pull-scripts:
	./scripts/charts-build-scripts/pull-scripts

CHARTS_BUILD_SCRIPTS_TARGETS := prepare patch clean clean-cache charts list index unzip zip standardize template

$(CHARTS_BUILD_SCRIPTS_TARGETS):
	@./scripts/charts-build-scripts/pull-scripts
	@./bin/charts-build-scripts $@

.PHONY: $(TARGETS) $(CHARTS_BUILD_SCRIPTS_TARGETS)