# sessions-control — AGENTS

## Contexto
- Go 1.27 · SQLite (modernc, sin cgo) · Bubble Tea · excelize · fpdf.
- Fuente remota: `~/.local/share/opencode/opencode.db` (solo lectura).
- BD local: `~/.local/share/sessions-control/sessions.db`.
- Spec: `docs/spec.md`.

## Comandos
```bash
make build      # bin/sessions-control
make run        # TUI
make test       # go test ./...
make vet        # go vet ./...
make fmt        # gofmt
make install    # ~/.local/bin/sessions-control
```

## Estructura
- `cmd/sessions-control`: flags, modos `-print`/`-export`, arranque TUI.
- `internal/domain`: tipos + `UniqueDuration` + agregados. Sin I/O.
- `internal/store`: schema, sync, filtros, overrides. Único punto SQL.
- `internal/report`: filas y totales de cada vista.
- `internal/export`: xlsx y pdf; consumen `report.Report` + `store.Stats`.
- `internal/tui`: modelo Bubble Tea, render y atajos.
- `internal/config`: rutas y variables de entorno.

## Invariantes
- `opencode.db` **nunca** se abre en escritura.
- Ediciones solo en `time_override`; `session_day` es reconstruible.
- `sync` no borra overrides.
- `unique` = unión de intervalos; `worked` = suma de duraciones.
- Fila TOTAL de grupos = unión global, no suma de únicos por grupo.
- Tokens solo de mensajes `assistant`.
- Día y horas en `time.Local`.

## Convenciones
- Error envuelto con `%w` y contexto en minúsculas.
- Sin comentarios salvo invariantes no obvios.
- Tests de store con SQLite temporal; no depender de datos reales.
- Estilos TUI en `render.go`; lógica de teclas en `model.go`.
- Añadir vista: `report.View*` + `report.Build` + `tui.columns` + export.

## Validación
- `make test` y `make vet` antes de cerrar cambios.
- Smoke real: `go run ./cmd/sessions-control -print -days YYYY-MM-DD`.
- TUI: probar 1/2/3/4, `f`, `e`, `r`, `s`, `x`, `p`, `?`, `q`.

## No hacer
- No editar `opencode.db`.
- No romper el formato de `session_day` sin migración en `schema`.
- No introducir dependencias que requieran cgo.
- No duplicar cálculo de agregados fuera de `domain`.
