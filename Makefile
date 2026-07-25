.PHONY: test test-repeat test-race vet bench verify baseline

test:
	go test -count=1 ./...

test-repeat:
	go test -count=10 ./...

test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

bench:
	go test ./benchmark -run='^$$' -bench=. -benchmem -benchtime=200ms -count=5

verify: test vet test-race

baseline: test-repeat vet test-race bench
