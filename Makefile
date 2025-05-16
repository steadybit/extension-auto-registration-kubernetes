# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## licenses-report: generate a report of all licenses
.PHONY: licenses-report
licenses-report:
ifeq ($(SKIP_LICENSES_REPORT), true)
	@echo "Skipping licenses report"
	rm -rf ./licenses && mkdir -p ./licenses
else
	@echo "Generating licenses report"
	rm -rf ./licenses
	go run github.com/google/go-licenses@v1.6.0 save . --save_path ./licenses
	go run github.com/google/go-licenses@v1.6.0 report . > ./licenses/THIRD-PARTY.csv
	cp LICENSE ./licenses/LICENSE.txt
endif

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	gofmt -l .
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-SA1019,-ST1000,-ST1003,-U1000 ./...
	go test -race -vet=off -coverprofile=coverage.out ./...
	go mod verify

## charttesting: Run Helm chart unit tests
.PHONY: charttesting
charttesting:
	echo "No helm chart"

## chartlint: Lint charts
.PHONY: chartlint
chartlint:
	echo "No helm chart"

## chart-bump-version: Bump the patch version and optionally set the appVersion
.PHONY: chart-bump-version
chart-bump-version:
	echo "No helm chart"
# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: build the extension
.PHONY: build
build:
	go mod verify
	go build -o=./extension

## run: run the extension
.PHONY: run
run: tidy build
	./extension

## container: build the container image
.PHONY: container
container:
	docker build --build-arg ADDITIONAL_BUILD_PARAMS="-cover -covermode=atomic" --build-arg SKIP_LICENSES_REPORT="true" -t extension-auto-registration-kubernetes:latest .
