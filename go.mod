module github.com/fastygo/app-suite

go 1.25.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/fastygo/app-crm v0.0.0
	github.com/fastygo/app-gocms v0.0.0
	github.com/fastygo/framework v0.0.0
	github.com/fastygo/module-chat v0.0.0
	github.com/fastygo/module-monitoring v0.0.0
	github.com/fastygo/module-support v0.0.0
	github.com/fastygo/platform v0.0.0
	github.com/fastygo/templ v0.0.0
)

require (
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.51.0 // indirect
)

replace github.com/fastygo/app-crm => ../@AppCRM

replace github.com/fastygo/app-gocms => ../@AppCMS

replace github.com/fastygo/framework => ../@Framework

replace github.com/fastygo/module-monitoring => ../@ModuleMonitoring

replace github.com/fastygo/module-support => ../@ModuleSupport

replace github.com/fastygo/module-chat => ../@ModuleChat

replace github.com/fastygo/platform => ../@Platform

replace github.com/fastygo/templ => ../@Templ

tool github.com/a-h/templ/cmd/templ
