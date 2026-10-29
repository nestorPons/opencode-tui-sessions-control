# sessions-control

TUI en Go para auditar el tiempo y los tokens de tus sesiones de opencode.

- Lee `~/.local/share/opencode/opencode.db` en modo solo lectura.
- Guarda agregados y horas editadas en `~/.local/share/sessions-control/sessions.db`.
- Vistas por sesiones, proyectos, meses y días.
- Filtros por varios días, rango, mes o proyecto.
- Edición de hora de inicio/fin por tramo (sesión/día).
- Exportación con totales a Excel y PDF.

## Instalación
```bash
cd ~/Proyectos/sessions-control
make build
make install        # opcional: ~/.local/bin/sessions-control
```

## Uso

```bash
make run        # desarrollo (go run)
make build && ./bin/sessions-control   # binario
```

```bash
./bin/sessions-control
```
Atajos:

| Tecla | Acción |
|---|---|
| `1` `2` `3` `4` | Sesiones · Proyectos · Meses · Días |
| `↑` `↓` `j` `k` | Mover |
| `PgUp` `PgDn` | Página |
| `g` `G` | Inicio · Final |
| `f` | Filtro |
| `t` `y` `w` `m` | Filtro rápido: hoy · ayer · últimos 7 días · mes (anclado al periodo visible) |
| `c` | Quitar filtro |
| `e` | Editar inicio/fin (vista Sesiones) |
| `r` | Restaurar horas originales |
| `s` | Sincronizar con opencode.db |
| `x` | Exportar Excel |
| `p` | Exportar PDF |
| `?` | Ayuda |
| `q` | Salir |

Filtro:
```
2026-10-26, 2026-10-27
month=2026-10
project=clowww.es
from=2026-10-01 to=2026-10-27
ayer
semana
all
```

## CLI
```bash
# Imprimir una vista
sessions-control -print -days 2026-10-26 -view projects

# Exportar
sessions-control -export xlsx -view months -month 2026-10 -out /tmp/octubre.xlsx
sessions-control -export pdf -view sessions -from 2026-10-01 -to 2026-10-31

# Sin re-sincronizar
sessions-control -no-sync -print -view days
```

Flags principales: `-source`, `-db`, `-exports`, `-no-sync`, `-view`,
`-days`, `-from`, `-to`, `-month`, `-project`, `-out`, `-reset-overrides`.

## Variables de entorno
- `SESSIONS_CONTROL_SOURCE`
- `SESSIONS_CONTROL_DB`
- `SESSIONS_CONTROL_EXPORTS`

## Documentación
- `docs/spec.md`: especificación funcional y reglas de cálculo.
- `docs/AGENTS.md`: guía para agentes y convenciones.
- `logs/state-changes.md`: registro de cambios de estado.

## Tests
```bash
make test
make vet
```
