# cube_sandbox_image_server development rules

## Project boundary

- Treat this directory as an independent Go project.
- Run Go and goctl commands with this directory as the working directory.
- Use this directory's `go.mod` and `go.sum`; do not rely on the parent project's module dependencies.
- Do not edit files in the parent project unless the user explicitly requests it.

## Code generation

- The API source of truth is `cube_sandbox_image_server.api`.
- Regenerate Go code with:

  ```bash
  go tool goctl api go -api cube_sandbox_image_server.api -dir .
  ```

- Use the goctl version pinned by this project's `go.mod`; do not rely on a globally installed version.

- Remove obsolete generated handler or logic files if goctl leaves files for deleted routes.

## Code quality

- Keep handlers limited to request parsing and response writing; place business behavior in `internal/logic`.
- Put reusable clients and long-lived dependencies in `internal/svc.ServiceContext` and configuration in `internal/config`.
- Prefer small, clearly named functions and idiomatic Go over duplicated request-mapping code.
- Handle and wrap errors with useful context. Do not silently ignore errors or log credentials, tokens, or other secrets.
- Preserve the API contract defined in `cube_sandbox_image_server.api` and add focused tests for non-trivial logic.
- Run `gofmt` on every changed Go file.

## Verification

After dependency or code changes, run from this directory:

```bash
go mod tidy
go vet ./...
go test ./...
```
