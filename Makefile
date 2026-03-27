.PHONY: setup

setup:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest
	brew install bufbuild/buf/buf sops age
