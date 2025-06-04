# Reeve CI / CD - Pipeline Step: Forgejo Test Package

This is a [Reeve](https://github.com/reeveci/reeve) step that tests whether files are present in a generic [Forgejo](https://forgejo.org) or [Gitea](https://gitea.com) package.

The files are selected using [glob patterns](https://pkg.go.dev/github.com/bmatcuk/doublestar/v4#Match).

## Configuration

See the environment variables mentioned in [Dockerfile](Dockerfile).
