TARGETS := $(shell ls scripts|grep -ve "^util-\|entry\|^pull-scripts")

# Define target platforms, image builder and the fully qualified image name.
TARGET_PLATFORMS ?= linux/amd64,linux/arm64

# Default behavior for targets without dapper
$(TARGETS):
	./scripts/$@


.DEFAULT_GOAL := default

# Charts Build Scripts
pull-scripts:
	./scripts/pull-scripts

rebase:
	./scripts/charts-build-scripts/rebase

CHARTS_BUILD_SCRIPTS_TARGETS := prepare patch clean clean-cache charts list index unzip zip standardize template

$(CHARTS_BUILD_SCRIPTS_TARGETS): pull-scripts
	@./bin/charts-build-scripts $@

.PHONY: $(TARGETS) $(CHARTS_BUILD_SCRIPTS_TARGETS) list

list-make:
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'
# IMPORTANT: The line above must be indented by (at least one)
#            *actual TAB character* - *spaces* do *not* work.

push-image: validate version ## build the container image targeting all platforms defined by TARGET_PLATFORMS and push to a registry.
	docker buildx build -f package/Dockerfile-prometheus-federator \
		--build-arg RANCHER_PROJECT_MONITORING=$RANCHER_PROJECT_MONITORING \
		--platform=$(TARGET_PLATFORMS) \
		-t "$(IMAGE)" --push .
	@echo "Pushed $(IMAGE)"
