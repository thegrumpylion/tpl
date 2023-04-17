~/.local/bin/tpl: $(wildcard *.go)
	go build -o $@

uninstall:
	rm -f ~/.local/bin/tpl

