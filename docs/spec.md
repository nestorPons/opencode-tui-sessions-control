# sessions-control — spec

## 1. Objetivo
- TUI en Go para visualizar, filtrar, editar y exportar el tiempo y los tokens de las sesiones de opencode.
- Fuente de verdad de sesiones: `~/.local/share/opencode/opencode.db` (solo lectura).
- Datos editables y agregados: SQLite local `~/.local/share/sessions-control/sessions.db`.

## 2. Alcance
- Vistas: sesiones (tramo sesión/día), proyectos, meses y días.
- Filtro por uno o varios días, rango, mes o proyecto.
- Edición manual de hora de inicio/fin por tramo (sesión/día).
- Restauración de horas originales.
- Exportación de la vista actual con totales a `.xlsx` y `.pdf`.
- Sincronización incremental por reconstrucción desde opencode.

## 3. Fuera de alcance
- Escribir en `opencode.db`.
- Gestión de usuarios, autenticación o red.
- Edición de tokens o coste.
- Multi-fuente (una única BD de opencode por ejecución).

## 4. Fuente de datos (opencode.db)
- Tablas usadas: `session`, `message`.
- Campos:
  - `session.id`, `session.directory`, `session.title`, `session.time_created`, `session.time_updated`.
  - `message.time_created`, `message.data` (JSON).
- JSON de `message.data`:
  - `role`, `cost`.
  - `tokens.{input,output,reasoning,cache.read,cache.write}`.
  - `time.{created,completed}`.

## 5. Modelo local (sessions-control.db)
```sql
session_day(
  session_id, day, project, project_name, title,
  start_ms, end_ms,
  tokens_input, tokens_output, tokens_reasoning,
  tokens_cache_read, tokens_cache_write, cost,
  PK(session_id, day)
)
time_override(
  session_id, day, start_ms, end_ms, updated_ms,
  PK(session_id, day)
)
meta(key, value)  -- source, synced_at
```

## 6. Reglas de cálculo
- Tramo: actividad de una sesión dentro de un día local (`YYYY-MM-DD`).
- `start` = mínimo `time.created` de los mensajes del día.
- `end` = máximo `time.completed` (fallback `time.created`).
- Tokens y coste: suma de mensajes con `role=assistant`.
- Sesión sin mensajes: un tramo con `time_created`..`time_updated`.
- `worked`: suma de duraciones de los tramos (solapes contados).
- `unique`: unión de intervalos [start,end] del conjunto mostrado.
- Filtro: los tramos se seleccionan por `day`; nunca se recorta un tramo.
- Edición: `time_override` prevalece sobre `session_day`; se conserva al re-sincronizar.
- Totales:
  - Horas = worked del conjunto filtrado.
  - Único = unión global del conjunto filtrado.
  - Único por grupo (proyecto/mes) = unión dentro del grupo; la fila TOTAL es la unión global.

## 7. Vistas
- **Sesiones**: una fila por sesión y día. Columnas: fecha, inicio, fin, horas, proyecto, sesión, tokens, coste. Editable.
- **Proyectos**: agregado por `project`. Inicio/fin, sesiones, días, worked, único, tokens, coste.
- **Meses**: agregado por `YYYY-MM`.
- **Días**: agregado por `YYYY-MM-DD`.

## 8. Filtros
- Sintaxis TUI/CLI:
  - `2026-10-26` o `2026-10-26,2026-10-27` → días exactos.
  - `from=2026-10-01 to=2026-10-27` → rango inclusivo.
  - `month=2026-10` → mes.
  - `project=clowww.es` → nombre o ruta del proyecto.
  - `ayer`, `hoy`, `semana`, `all`.
  - Atajos TUI: `t` hoy, `y` ayer, `w` últimos 7 días, `m` mes completo; se anclan al último día del periodo visible (`month`, `to`, `from` o `days`) y, sin filtro, a hoy. Conservan `project` y recalculan el periodo.
- Combinación: día/rango + mes + proyecto se aplican con AND.

## 9. Edición de horas
- Disponible solo en vista Sesiones.
- Entrada: `HH:MM` o `HH:MM:SS`; el día se toma del tramo.
- Validación: `end > start`.
- `r` elimina el override del tramo seleccionado.
- Marca visual `*` en inicio/fin editados.

## 10. Exportación
- Formatos: `.xlsx` (excelize) y `.pdf` (fpdf, A4 horizontal).
- Contenido: vista actual + fila TOTAL + resumen.
- XLSX:
  - Hoja de vista (`Sesiones|Proyectos|Meses|Dias`).
  - Hoja `Detalle`: tramos con tokens desglosados.
  - Hoja `Resumen`: métricas globales y filtro.
- PDF: tabla con totales, filtro, fuente y resumen por página.
- Destino por defecto: `~/sessions-control-exports/`.

## 11. CLI
```
sessions-control                     # TUI
sessions-control -print -days ...    # tabla por stdout
sessions-control -export xlsx|pdf -view ... -from ... -to ... -out ...
```
Flags: `-source`, `-db`, `-exports`, `-no-sync`, `-view`, `-days`, `-from`, `-to`, `-month`, `-project`, `-out`, `-reset-overrides`, `-version`.

## 12. Configuración
- Flags > variables de entorno > defaults.
- Variables: `SESSIONS_CONTROL_SOURCE`, `SESSIONS_CONTROL_DB`, `SESSIONS_CONTROL_EXPORTS`.

## 13. Estructura
```
cmd/sessions-control/   main + CLI
internal/config/        rutas y entorno
internal/domain/        tipos, unión de intervalos, agregados
internal/store/         SQLite local + sync + overrides
internal/report/        filas de vista y totales
internal/export/        xlsx y pdf
internal/tui/           bubbletea: vistas, filtros, edición
docs/                   spec.md, AGENTS.md
logs/                   state-changes.md
```

## 14. Criterios de aceptación
- `go build ./...` y `go test ./...` sin errores.
- Sync reproducible desde opencode.db real.
- Editar inicio/fin modifica worked/unique y persiste tras sync.
- Filtros por varios días y por mes devuelven totales coherentes.
- XLSX y PDF abren correctamente e incluyen fila de totales.
- `-print` coincide con la vista TUI equivalente.

## 15. Decisiones
- Día local del sistema; horas mostradas en local.
- Un tramo no se parte al filtrar por día (pertenece al día de su `start`).
- `unique` global no es la suma de `unique` por grupo.
- SQLite local derivada; reconstruible con `s`/`-no-sync=false`.
