# Spaghetti: a dependency analysis tool for Go packages

Spaghetti is an interactive web-based tool to help you understand the dependencies of a Go program, and to evaluate the benefit of various possible refactorings to eliminate dependencies. Since it is not yet public, run it like so:

```shell
$ git clone git@github.com:adonovan/spaghetti.git
$ cd spaghetti
$ go run . -- [package]
```
where _package_ is or more Go packages, or a pattern understood by `go list`. Then point your browser at the insecure single-user web server at `localhost:18080`.
