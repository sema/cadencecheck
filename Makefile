
ALL_CMDS := $(shell go list ./cmd/... )
ALL_PACKAGES := $(shell go list ./... | grep -v mocks)

build:
	@echo $(ALL_CMDS) | xargs -n1 go build

lint:
	go fmt $(ALL_PACKAGES)
	go vet $(ALL_PACKAGES)
	dep check

test:
	go test --coverprofile=cover.out $(ALL_PACKAGES)
	go tool cover -func=cover.out

mocks:
	go generate $(ALL_PACKAGES)

updategolden:
	UPDATE_GOLDEN=1 go test github.com/sema/cadencecheck/examples
