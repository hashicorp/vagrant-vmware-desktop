CGO_ENABLED?=0

.PHONY: bin
bin: # bin creates the binaries for Vagrant for the current platform
	CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility ./go_src/vagrant-vmware-utility

.PHONY: debug
debug: # debug creates an executable with optimizations off, suitable for debugger attachment
	GCFLAGS="all=-N -l" $(MAKE) bin

.PHONY: bin/windows
bin/windows: # create windows binaries
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility.exe ./go_src/vagrant-vmware-utility

.PHONY: bin/linux
bin/linux: # create Linux binaries
	GOOS=linux GOARCH=amd64 $(MAKE) bin

.PHONY: bin/darwin
bin/darwin: # create Darwin binaries
	GOOS=darwin GOARCH=amd64 $(MAKE) bin

.PHONY: bin/darwin-arm
bin/darwin-arm: # create Darwin binaries
	GOOS=darwin GOARCH=arm64 $(MAKE) bin

.PHONY: bin/darwin-universal
bin/darwin-universal: # create Darwin universal binaries
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/.vagrant-vmware-utility_darwin_arm64 ./go_src/vagrant-vmware-utility
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/.vagrant-vmware-utility_darwin_amd64 ./go_src/vagrant-vmware-utility
	go run github.com/randall77/makefat ./bin/vagrant-vmware-utility_darwin_universal ./bin/.vagrant-vmware-utility_darwin_arm64 ./bin/.vagrant-vmware-utility_darwin_amd64
	rm -f ./bin/.vagrant-vmware-utility_darwin*

.PHONY: all
all: # create all binaries
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility.exe ./go_src/vagrant-vmware-utility
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility_linux ./go_src/vagrant-vmware-utility
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility_darwin_amd64 ./go_src/vagrant-vmware-utility
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/vagrant-vmware-utility_darwin_arm64 ./go_src/vagrant-vmware-utility
	go run github.com/randall77/makefat ./bin/vagrant-vmware-utility_darwin_universal ./bin/vagrant-vmware-utility_darwin_amd64 ./bin/vagrant-vmware-utility_darwin_arm64

.PHONY: clean
clean:
	rm -f ./bin/* ./bin/.v*

.PHONY: test
test: # run tests
	go test ./go_src/vagrant-vmware-utility/...
