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

## 2026-11-11 — Instalación local y versión inyectable
- `packaging/install.sh`: nueva opción `--local [DIR]` que compila el checkout actual (sin clonar ni descargar release).
- Sin release publicada y con Go disponible: usa el checkout del propio script antes de clonar GitHub.
- `ldflags()` centraliza `-s -w -X main.version`; aplica también a `--from-source`.
- `cmd/sessions-control/main.go`: `version` pasa de `const` a `var` para que `-ldflags -X` funcione.
- `Makefile`: nuevo `install-local` (compila con versión y copia a `~/.local/bin`); ayuda actualizada.
- `README.md` y `packaging/README.md`: documentado `--local` e `install-local`.
- Instalado en `~/.local/bin/sessions-control` con `./packaging/install.sh --local . --user --force`; smoke `-print` OK.
- `make test` y `make vet` OK.

## 2026-11-23 — Navegación por días con ←/→
- `internal/tui/model.go`: nuevas teclas `left`/`h` y `right`/`l` en vista lista para mover el filtro un día atrás/adelante (`shiftFilterDay`).
- Nuevo `shiftDays`: desplaza `Days` (todos) o el rango `From`/`To`; conserva `project`; sin fechas en el filtro no hace nada y muestra aviso.
- `internal/tui/model_test.go`: test `TestShiftDays` (ayer, mes/año, varios días, rango, solo mes, sin fechas).
- `internal/tui/render.go`: ayuda y pie actualizados con `← →`.
- `make test` y `make vet` OK.

## 2026-11-24 — Columna «Efectivo» (tiempo efectivo)
- Causa: el span de una sesión incluye pausas, cortes de red y esperas; no sirve para facturar.
- `internal/domain/domain.go`: `Segment/Session/Group.Effective` + `SumEffective`.
- `internal/store/store.go`: nueva columna `session_day.effective_ms` (con migración para bases existentes).
  - Cálculo en sync: suma de huecos entre mensajes consecutivos ≤ `gap`; los huecos mayores no cuentan.
  - Umbral en `meta.effective_gap_ms`; por defecto 15 min.
  - Overrides manuales recortan el efectivo si el tramo queda más corto.
- `internal/config/config.go`: `EffectiveGap` (env `SESSIONS_CONTROL_GAP`, minutos).
- `cmd/sessions-control/main.go`: flag `-gap` (minutos); `-no-sync` usa el umbral guardado y avisa si difiere.
- `internal/report/report.go`: columna «Efectivo» en las 4 vistas, totales y resumen (`efectivo ≤ N min`).
- `internal/tui/render.go`: columna en la tabla, umbral en la cabecera y ayuda.
- `internal/export/*`: columna en Excel (hoja + detalle), resumen «Horas efectivas» y PDF.
- Tests: `TestEffectiveTime` (sync, umbral, override), `TestEffectiveSums`, `TestViewShowsEffectiveColumn`.
- Docs: `docs/spec.md`, `docs/AGENTS.md`, `README.md`.
- Smoke real 2026-11-23: lamostradevalencia.com 9,47 h → 5,31 h efectivas (umbral 15 min).
- `make test` y `make vet` OK; XLSX y PDF generados y legibles.
- Instalado en `~/.local/bin/sessions-control` con `make install-local`.

## 2026-11-24 — Reparto de jornada entre proyectos
- Causa: sumar `effective` entre sesiones solapadas da 12-16 h en días de 6 h. El efectivo es por sesión; no debe sumarse.
- `internal/store/store.go`:
  - nueva columna `session_day.allocated_ms` (migración automática).
  - `Options{Gap, DayGap, Internal}` y `Sync(source, Options)`.
  - jornada por día = huecos entre mensajes de proyectos **no internos** ≤ `day_gap` (5 min por defecto).
  - `allocated_i = effective_i × jornada / Σ effective`; la suma diaria se ajusta a la jornada.
  - proyectos internos (nombre o ruta) quedan con reparto 0 y fuera del cómputo.
  - `Stats.DayGap`; meta `day_gap_ms`.
- `internal/config/config.go`: `DayGap`, `Internal`; env `SESSIONS_CONTROL_DAY_GAP`, `SESSIONS_CONTROL_INTERNAL`.
- `internal/domain/domain.go`: `Allocated` + `SumAllocated`.
- `internal/report/report.go`: columna «Reparto» en las 4 vistas, totales y resumen.
- `internal/tui/*`: columna, umbral en cabecera y ayuda.
- `internal/export/*`: «Reparto» en Excel (hoja + Detalle), «Horas repartidas» en Resumen y PDF.
- `cmd/sessions-control/main.go`: flags `-day-gap` y `-internal`.
- Tests: `TestAllocateDays` (proporción, suma exacta, exclusión de internos); ajustados `TestEffectiveTime` y `TestViewShowsEffectiveColumn`.
- Smoke 2026-11-23 con `-internal "web-efects,md-editor"`: efectivo 16,02 h → reparto **6,40 h** (La Mostra 3,00 h, Muniva 3,21 h, Olesa 0,19 h).
- `make test` y `make vet` OK; XLSX y PDF verificados.
