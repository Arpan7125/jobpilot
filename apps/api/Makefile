include .env
export

.PHONY: run test

run:
	go run ./cmd/server

# Wipes and re-migrates the jobpilot_test database.
test:
	TEST_DATABASE_URL=postgres://jobpilot:jobpilot@localhost:5432/jobpilot_test go test ./...
