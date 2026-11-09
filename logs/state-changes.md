# sessions-control — cambios de estado

## 2026-10-27 10:19 — Creación del proyecto
- `go mod init github.com/nestorpons/sessions-control`.
- Dependencias: bubbletea, bubbles, lipgloss, modernc/sqlite, excelize/v2, go-pdf/fpdf.
- Estructura `cmd/` + `internal/{config,domain,store,report,export,tui}` + `docs/` + `logs/`.

## 2026-10-27 10:22 — Modelo y store
- `internal/domain`: Segment/Session/Group, `UniqueDuration`, agregados sesión/proyecto/mes/día.
- `internal/store`: schema `session_day`, `time_override`, `meta`; sync desde opencode.db; filtros; overrides.
- `internal/config`: rutas XDG y variables de entorno.

## 2026-10-27 10:23 — Vistas y exportación
- `internal/report`: filas/totales por vista.
- `internal/export`: XLSX (vista + Detalle + Resumen), PDF A4 horizontal.
- `internal/tui`: Bubble Tea, vistas 1-4, filtros, edición de horas, sync, export, ayuda.
- `cmd/sessions-control`: CLI `-print`, `-export`, `-view`, filtros, `-version`.

## 2026-10-27 10:24 — Verificación
- `go build ./...` OK.
- Tests: domain (unión de intervalos, agregados, nombres) y store (sync, filtros, overrides, persistencia tras sync).
- `go test ./...` OK.
- Pruebas con `opencode.db` real: 170 sesiones, 181 tramos.
- Export XLSX/PDF validados (`/tmp/sc-test.*`).
- Smoke TUI en PTY: 1/2/3/4, f, e, ?, x, p, q.

## 2026-10-27 10:33 — Documentación
- `docs/spec.md`, `docs/AGENTS.md`, `AGENTS.md`, `README.md`, `Makefile`, `.gitignore`.
- `logs/state-changes.md` (este archivo).

## 2026-10-27 10:37 — Ajustes finales
- Vista por defecto: todas las sesiones (sin filtro).
- `Días` por grupo = días distintos (no tramos).
- Validación de fechas/mes en CLI (`-days`, `-from`, `-to`, `-month`).
- Formatos numéricos en Excel: horas 0.00, coste 0.0000, tokens con separador.
- Resúmenes con plural correcto (1 día / 1 sesión).
- `make build test vet` OK; smoke TUI en PTY OK.

## 2026-10-27 11:47 — Atajos de filtro por fecha
- TUI: `t` hoy, `y` ayer, `w` últimos 7 días.
- Los atajos conservan el filtro de proyecto y descartan `month`.
- `datePreset` en `internal/tui/model.go` + test `model_test.go`.
- Ayuda, footer, README y spec actualizados.
- `make test` y `make vet` OK.

## 2026-10-27 13:02 — Atajo de mes actual
- TUI: `m` filtra del día 1 al último día del mes actual.
- Conserva `project` y descarta `month`; completa el rango con `from`/`to`.
- Test de `datePreset` ampliado; ayuda, footer, README y spec actualizados.
- `make test` y `make vet` OK.

## 2026-10-27 13:23 — Atajos anclados al periodo visible
- `t/y/w/m` recalculan desde el último día del periodo visible (`month`, `to`, `from`, `days`); sin filtro usan hoy.
- Se conserva `project` y se reemplaza el periodo anterior.
- Test `datePreset` ampliado (sin filtro, mes, rango y días).
- Ayuda, README y spec actualizados.
- `make test` y `make vet` OK.

## 2026-10-29 — Publicación del repositorio
- `.gitignore` ampliado: binarios, bases SQLite, exportaciones, `.vscode/sftp.json`, `.env`, claves y logs.
- `git init`, commit inicial y push a `github.com/nestorPons/opencode-tui-sessions-control` (público).
- `make test` y `make vet` OK.

## 2026-11-06 — Soporte del esquema V2 de opencode
- Causa: opencode migró a `session_v2`/`session_message`; el sync solo leía `session`/`message`, congelado a 2026-10-29.
- `internal/store/store.go`: detección de tablas, sync V2 preferente y fallback legado para IDs ausentes.
- `session_message.type` como rol; se ignoran eventos `idle`, `compaction` y `agent-switched`.
- `internal/store/store_test.go`: test de sync V2, títulos nulos, eventos ignorados y fallback legado.
- `docs/spec.md`, `docs/AGENTS.md` actualizados.
- Sync real: 277 sesiones, 290 tramos (2026-09-05 → 2026-11-06).
- `make test` y `make vet` OK.

## 2026-11-09 — Instaladores y paquetes Linux
- `packaging/install.sh`: instalador universal (release, checksum, `--user`, `--prefix`, `--from-source`, `--uninstall`).
- `packaging/nfpm.yaml`: paquetes `deb`, `rpm`, `apk` y `archlinux`.
- `packaging/PKGBUILD`: receta Arch/AUR compilando desde el tag.
- `Makefile`: `dist` (tarballs amd64/arm64/armv7/386 + `checksums.txt`), `packages`, `release`, `deb|rpm|apk|archlinux`, `packaging-help`; versión por tag y `-X main.version`.
- Etiquetas nativas por distro: `armhf`, `armv7hl`, `armv7h`, `i386/i686/x86`.
- `README.md` y `packaging/README.md`: instalación por distro.
- `.gitignore`: `dist/`.
- Verificado: 4 tarballs + checksums, 16 paquetes (4 distros × 4 arqs), binarios estáticos, `install.sh --from-source` y `--uninstall`; `make test`/`make vet` OK.
