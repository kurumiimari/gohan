PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

dir = $(shell pwd)
temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))
git_hash = $(shell git log --oneline | head -n 1 | cut -d ' ' -f 1)
built_on = $(shell date)
uname = $(shell uname)
tar_cmd = "tar"

ifeq ($(uname), Darwin)
	tar_cmd = "gtar"
endif

build-cross: clean $(PLATFORMS)
.PHONY: build-cross

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -trimpath -o ./bin/gohan-$(os)-$(arch) cmd/gohan/main.go
	mv bin/gohan-$(os)-$(arch) bin/gohan
	cd bin && $(tar_cmd) --owner=0 --group=0 -cvzf gohan-$(build_version).$(os)-$(arch).tar.gz gohan
	rm bin/gohan
.PHONY: $(PLATFORMS)

gohan:
	go build -trimpath -o ./bin/gohan ./cmd/gohan/main.go
.PHONY: gohan

clean:
	rm -rf ./bin
.PHONY: clean

test:
	IS_TEST=1 go test -v ./...
.PHONY: test/l

release:
	bash ./release.sh