BINARY_NAME := Spotify_controller
BIN_DIR := ./bin
DESTDIR = $(HOME)
BINDIR ?= $(DESTDIR)/usr/bin
RESOURCES_DIR := $(DESTDIR)/usr/share/assets/qualc
INSFLAGS = -m 0755


#color codes
GREEN := \033[1;32m
YELLOW := \033[1;33m
NC := \033[0m

suffix :=
buildFlag :=
rm_cmd := rm -f 
zip := zip -r
ifeq ($(OS),Windows_NT)
    suffix =.exe
    buildFlag = -ldflags -H=windowsgui
	GREEN := 
	YELLOW := 
	NC := 
	rm_cmd := cmd /C del /q
	zip = zip -r
endif

.PHONY: all build install clean profile

all: dep build coverage

build: dep tidy
	@echo "$(GREEN)building binary$(NC)"
	go build -o ./bin/${BINARY_NAME}${suffix} ${buildFlag} ./main.go
	@echo "$(YELLOW)done building$(NC)"

run: build
	@echo "$(GREEN)running binary program$(NC)"
	./bin/${BINARY_NAME}${suffix}
	

tidy:
	@echo "$(GREEN)cleaning imports$(NC)"
	go mod tidy
	@echo "$(YELLOW)done$(NC)"
	


dep:
	@echo "$(GREEN)downloading libraries$(NC)"
	go mod download
	@echo "$(YELLOW)done$(NC)"

doc:
	@echo "$(GREEN)getting tool for documentation$(NC)"
	go install golang.org/x/pkgsite/cmd/pkgsite@latest
	@echo "$(YELLOW)done$(NC)"
	@echo "$(GREEN)opening documentation$(NC)"
	pkgsite .
