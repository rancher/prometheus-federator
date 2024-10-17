TARGETS := $(shell ls scripts|grep -ve "^util-\|entry\|^pull-scripts")
LOCAL_TARGETS := $(addprefix local-,$(TARGETS))

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-$$(uname -s)-$$(uname -m) > ./bin/.dapper.tmp
	@@chmod +x ./bin/.dapper.tmp
	@./bin/.dapper.tmp -v
	@mv ./bin/.dapper.tmp ./bin/.dapper

# Default behavior for targets without dapper
$(TARGETS):
	./scripts/$@

# Behavior for targets prefixed with "local-" using dapper
$(LOCAL_TARGETS): local-%: .dapper
	./bin/.dapper $(@:local-%=%)

.DEFAULT_GOAL := default

# Charts Build Scripts
pull-scripts:
	./scripts/pull-scripts

rebase:
	./scripts/charts-build-scripts/rebase

CHARTS_BUILD_SCRIPTS_TARGETS := prepare patch clean clean-cache charts list index unzip zip standardize template

$(CHARTS_BUILD_SCRIPTS_TARGETS): pull-scripts
	@./bin/charts-build-scripts $@

.PHONY: $(TARGETS) $(CHARTS_BUILD_SCRIPTS_TARGETS) $(LOCAL_TARGETS) list

list-make:
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'
# IMPORTANT: The line above must be indented by (at least one)
#            *actual TAB character* - *spaces* do *not* work.
