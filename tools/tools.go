// +build tools

package tools

//go:generate go clean
//go:generate go install -v -mod=vendor github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install -v -mod=vendor github.com/vasi-stripe/gogroup/cmd/gogroup
//go:generate go install -v -mod=vendor github.com/goreleaser/goreleaser
//go:generate go install -mod=vendor golang.org/x/tools/gopls

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/vasi-stripe/gogroup/cmd/gogroup"
	_ "golang.org/x/tools/gopls"
)
