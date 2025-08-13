# Function to get the effective branch (default branch if current is a tag)
define get_effective_branch
$(if $(shell echo $(1) | grep -E '^refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$$'),"main",$(1))
endef

AUTHOR_EMAIL     = sysops@teamwork.com
AUTHOR_NAME      = Teamwork Github Actions
GH_TOKEN         = XXXXXXXX
SSH_AGENT        = default
VCS_REF          = $(shell git rev-parse --short HEAD)
VERSION          = v$(shell git describe --always --match "v*" | sed 's/^v//')
BRANCH           = $(shell git rev-parse --abbrev-ref HEAD)
EFFECTIVE_BRANCH = $(call get_effective_branch,$(BRANCH))
LATEST_TAG       = 343218184206.dkr.ecr.us-east-1.amazonaws.com/teamwork/mcp:$(subst /,,${EFFECTIVE_BRANCH})-latest
TAG              = 343218184206.dkr.ecr.us-east-1.amazonaws.com/teamwork/mcp:$(VERSION)

.PHONY: build push install

default: build

build:
	docker buildx build \
	  --build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	  --build-arg BUILD_VCS_REF=$(VCS_REF) \
	  --build-arg BUILD_VERSION=$(VERSION) \
	  --load \
	  --progress=plain \
	  --ssh $(SSH_AGENT) \
	  .

push:
	docker buildx build \
	  --platform linux/amd64,linux/arm64 \
	  --build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	  --build-arg BUILD_VCS_REF=$(VCS_REF) \
	  --build-arg BUILD_VERSION=$(VERSION) \
	  -t $(TAG) \
	  -t $(LATEST_TAG) \
	  --push \
	  --progress=plain \
	  --ssh $(SSH_AGENT) \
	  .

install:
	sudo wget https://github.com/mikefarah/yq/releases/download/v4.16.2/yq_linux_amd64 -O /usr/bin/yq
	sudo chmod +x /usr/bin/yq