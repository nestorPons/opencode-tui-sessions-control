# sessions-control

TUI en Go para auditar el tiempo y los tokens de tus sesiones de opencode.

- Lee `~/.local/share/opencode/opencode.db` en modo solo lectura.
- Guarda agregados y horas editadas en `~/.local/share/sessions-control/sessions.db`.
- Vistas por sesiones, proyectos, meses y días.
- Tiempo efectivo: descuenta pausas y cortes (huecos entre mensajes mayores que `-gap`, 15 min por defecto).
- Reparto: reparte la jornada real entre proyectos facturables. Marca los internos con `i` en la vista Proyectos (se guarda en la base local; `global` y `sessions-control` son siempre internos); la suma del reparto es la jornada, no la suma de efectivos.
- Importe: reparto × tarifa horaria. Ajusta la tarifa con `v` (19 €/h por defecto); se guarda en la base local.
- Filtros por varios días, rango, mes o proyecto.
- Edición de hora de inicio/fin por tramo (sesión/día).
- Exportación con totales a Excel y PDF.

## Instalación

### Instalador universal (cualquier distro)
```bash
curl -fsSL https://raw.githubusercontent.com/nestorPons/opencode-tui-sessions-control/main/packaging/install.sh | bash -s -- --user
```
Opciones: `--user` (~/.local/bin), `--prefix DIR`, `--version VER`,
`--from-source`, `--local DIR` (checkout local), `--uninstall`. Ver `packaging/README.md`.

### Paquetes por distro
```bash
make dist        # tarballs linux amd64/arm64/armv7/386 + checksums
make packages    # deb, rpm, apk y archlinux (requiere nfpm)
```

| Distro | Instalación |
|---|---|
| Debian/Ubuntu/Mint | `sudo apt install ./dist/sessions-control_*.deb` |
| Fedora/RHEL/Rocky | `sudo dnf install ./dist/sessions-control-*.rpm` |
| openSUSE | `sudo zypper install ./dist/sessions-control-*.rpm` |
| Alpine | `sudo apk add --allow-untrusted ./dist/sessions-control_*.apk` |
| Arch/Manjaro | `cd packaging && makepkg -si` |

### Desde el código
```bash
make build
make install        # ~/.local/bin/sessions-control
./packaging/install.sh --local . --user   # instalador, compila el checkout
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
| `i` | Marcar facturable / interno (vista Proyectos) |
| `v` | Ajustar la tarifa horaria (€/h) |
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

# Umbral de tiempo efectivo (minutos entre mensajes)
sessions-control -print -days 2026-11-23 -gap 20

# Jornada y reparto: 5 min y proyectos internos fuera del cómputo
sessions-control -print -days 2026-11-23 -day-gap 5 \
  -internal "mi-web,mi-blog"
```

Flags principales: `-source`, `-db`, `-exports`, `-no-sync`, `-view`,
`-days`, `-from`, `-to`, `-month`, `-project`, `-gap`, `-day-gap`, `-internal`,
`-out`, `-reset-overrides`.

## Variables de entorno
- `SESSIONS_CONTROL_SOURCE`
- `SESSIONS_CONTROL_DB`
- `SESSIONS_CONTROL_EXPORTS`
- `SESSIONS_CONTROL_GAP` (minutos)
- `SESSIONS_CONTROL_DAY_GAP` (minutos)
- `SESSIONS_CONTROL_INTERNAL` (proyectos internos separados por comas)

## Documentación
- `docs/spec.md`: especificación funcional y reglas de cálculo.
- `docs/AGENTS.md`: guía para agentes y convenciones.
- `logs/state-changes.md`: registro de cambios de estado.

## Tests
```bash
make test
make vet
```
