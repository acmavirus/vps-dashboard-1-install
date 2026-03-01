.PHONY: all build-frontend build-backend clean

all: build

build-frontend:
	cd frontend && npm install && npm run build

build-backend:
	go mod tidy
	go build -ldflags "-s -w" -o vps-dash main.go

build: build-frontend build-backend
	@echo "Build thành công! Chạy './vps-dash' để khởi động Dashboard."

clean:
	rm -rf vps-dash frontend/dist
