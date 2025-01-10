current_patch_number := $(shell git tag --list "v0.2.*" | sort -V | tail -n 1 | cut -c 6-)
next_patch_number := $(shell echo $$(($(current_patch_number)+1)))

release:
	git tag v0.2.$(next_patch_number)
	git push origin master v0.2.$(next_patch_number)

test:
	go test -v ./...

test_and_watch:
	onchange '**/*' -- go test -v ./...

setup:
	echo "Setup in tester-utils"