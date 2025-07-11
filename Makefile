AUTHOR_EMAIL = sysops@teamwork.com
AUTHOR_NAME  = Teamwork Github Actions
GH_TOKEN     = XXXXXXXX
SSH_AGENT    = default
VCS_REF      = $(shell git rev-parse --short HEAD)
VERSION      = v$(shell git describe --always --match "v*")
BRANCH       = $(shell git rev-parse --abbrev-ref HEAD)
LATEST_TAG   = 343218184206.dkr.ecr.us-east-1.amazonaws.com/teamwork/mcp:$(subst /,,${BRANCH})-latest
TAG          = 343218184206.dkr.ecr.us-east-1.amazonaws.com/teamwork/mcp:$(VERSION)

.PHONY: build push chart-update git-push install

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

chart-update:
	yq eval -i '.appVersion = "$(VERSION)"' chart/Chart.yaml

git-push: chart-update
	git commit -am "[ci skip] Updated helm chart to $(VERSION)"
	git push gh HEAD:$(BRANCH)

install:
	sudo wget https://github.com/mikefarah/yq/releases/download/v4.16.2/yq_linux_amd64 -O /usr/bin/yq
	sudo chmod +x /usr/bin/yq