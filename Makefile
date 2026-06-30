.PHONY: generate build run test clean

# 生成 Ent 代码
generate:
	go generate ./internal/ent/...

# 生成 proto 代码（需在 proto 目录执行 buf generate）
proto-gen:
	cd ../proto && buf generate

# 构建
build:
	go build -o bin/server ./cmd/server

# 运行
run:
	go run ./cmd/server

# 测试
test:
	go test ./... -v -race -count=1

# 清理
clean:
	rm -rf bin/ data/
