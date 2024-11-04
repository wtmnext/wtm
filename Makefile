MAIN=cmd/wtm/server.go
P=bin/wtm
FLAGS=-ldflags="-s -w"

test:
	@go test ./tests/... -v
fmt:
	@go fmt ./...
tpl:
	@templ generate
build: tpl
	@go build -o $(P) $(MAIN)
release: tpl
	@go build -o $(P) $(FLAGS) $(MAIN)
run: build
	$(P)
serve:
	@air --build.cmd "templ generate && go build -o $(P) $(MAIN)" \
			 --build.bin "$(P)" --build.exclude_dir "assets,tmp,vendor,.git,_docker" --build.exclude_regex ".*_templ.go" \
	     --build.include_ext "go,tpl,tmpl,templ,html" \
	     --build.delay "1000" --build.stop_on_error "true"

