# Run tidy
tidy::
	go mod tidy -v

# Run test
test::
	go test ./internal/... ./pkg/...