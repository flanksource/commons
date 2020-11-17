TEST_FLAGS = -v
TEST_TARGET = ./...

test:
	go test $(TEST_FLAGS) $(TEST_TARGET)