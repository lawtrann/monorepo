.PHONY: setup

setup:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest
	@echo "Install SOPS and age via brew:"
	brew install sops age
