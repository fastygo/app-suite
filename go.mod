module github.com/fastygo/app-suite

go 1.25.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/fastygo/module-cms v0.0.0
	github.com/fastygo/module-crm v0.0.0
	github.com/fastygo/module-monitoring v0.0.0
	github.com/fastygo/panel v0.0.0
	github.com/fastygo/platform v0.0.0
)

require (
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/fastygo/templ v0.0.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
)

replace github.com/fastygo/module-cms => ../@ModuleCMS

replace github.com/fastygo/module-crm => ../@ModuleCRM

replace github.com/fastygo/module-monitoring => ../@ModuleMonitoring

replace github.com/fastygo/panel => ../@Panel

replace github.com/fastygo/platform => ../@Platform

replace github.com/fastygo/templ => ../@Templ

tool github.com/a-h/templ/cmd/templ
