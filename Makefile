.PHONY: css css-watch build build-optimized run

css:
	./tailwindcss-macos-arm64 -i static/css/input.css -o static/css/style.css --minify

css-watch:
	./tailwindcss-macos-arm64 -i static/css/input.css -o static/css/style.css --watch

# Standard build (for development)
build:
	go build -o zrp .

# Optimized build (production) - strips symbols and debug info, removes file paths
build-optimized:
	go build -ldflags="-s -w" -trimpath -o zrp .

run: css build
	./zrp
