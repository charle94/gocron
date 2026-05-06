GO111MODULE=on

.PHONY: build
build: gocron node

.PHONY: build-race
build-race: enable-race build

.PHONY: run
run: build kill
	./bin/gocron-node &
	./bin/gocron web -e dev

.PHONY: run-race
run-race: enable-race run

.PHONY: kill
kill:
	-killall gocron-node

.PHONY: gocron
gocron:
	go build $(RACE) -o bin/gocron ./cmd/gocron

.PHONY: node
node:
	go build $(RACE) -o bin/gocron-node ./cmd/node

# 静态编译：CGO_ENABLED=0，纯 Go 实现，适合容器化/精简部署
.PHONY: build-static
build-static:
	CGO_ENABLED=0 go build -ldflags '-w -s' -o bin/gocron ./cmd/gocron
	CGO_ENABLED=0 go build -ldflags '-w -s' -o bin/gocron-node ./cmd/node

# macOS ARM64（Apple Silicon）可执行文件
.PHONY: build-mac-arm
build-mac-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o bin/gocron-darwin-arm64 ./cmd/gocron
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o bin/gocron-node-darwin-arm64 ./cmd/node

# Linux AMD64 可执行文件
.PHONY: build-linux-amd64
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o bin/gocron-linux-amd64 ./cmd/gocron
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o bin/gocron-node-linux-amd64 ./cmd/node

# 跨平台编译（macOS ARM64 + Linux AMD64 + 当前平台）
.PHONY: build-cross
build-cross: build-static build-mac-arm build-linux-amd64

.PHONY: test
test:
	go test $(RACE) ./...

.PHONY: test-race
test-race: enable-race test

.PHONY: enable-race
enable-race:
	$(eval RACE = -race)

.PHONY: package
package: build-vue statik
	bash ./package.sh

.PHONY: package-all
package-all: build-vue statik
	bash ./package.sh -p 'linux darwin windows'

.PHONY: build-vue
build-vue:
	cd web/vue && yarn run build
	cp -r web/vue/dist/* web/public/

.PHONY: install-vue
install-vue:
	cd web/vue && yarn install

.PHONY: run-vue
run-vue:
	cd web/vue && yarn run dev

.PHONY: statik
statik:
	go get github.com/rakyll/statik
	go generate ./...

.PHONY: lint
	golangci-lint run

.PHONY: clean
clean:
	rm bin/gocron
	rm bin/gocron-node
