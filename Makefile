BINARY_NAME := debug-agent
BUILD_DIR := ./bin
GO := go

.PHONY: bin clean run

bin:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)

run: bin
	$(BUILD_DIR)/$(BINARY_NAME)
