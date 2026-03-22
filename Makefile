# NeuroForge SDX Root Makefile

.PHONY: all test api-compliance clean

all: api-compliance test

api-compliance:
	@echo "Running API Compliance Checks..."
	@$(MAKE) -C tools/swagger api-compliance-full

test:
	@echo "Running Contract Tests..."
	@cd tests/contract && go test -v -count=1 ./...

clean:
	@$(MAKE) -C tools/swagger clean
