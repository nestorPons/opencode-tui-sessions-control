# Empaquetado e instalación

## Estructura
- `install.sh`: instalador universal (descarga release o compila desde fuente).
- `nfpm.yaml`: plantilla de paquetes `deb`, `rpm`, `apk` y Arch Linux.
  El `Makefile` sustituye `${PKG_VERSION}`, `${PKG_ARCH}` y `${BINARY}`.
- `PKGBUILD`: receta para Arch Linux / AUR.

## Instalador universal
```bash
# última release, instalación de usuario
curl -fsSL https://raw.githubusercontent.com/nestorPons/opencode-tui-sessions-control/main/packaging/install.sh | bash -s -- --user

# sistema (/usr/local/bin, pide sudo)
./packaging/install.sh

# versión concreta, sin publicaciones disponibles
./packaging/install.sh --version 0.1.0 --from-source --user
```
Opciones: `--user`, `--prefix DIR`, `--version VER`, `--from-source`,
`--uninstall`, `--no-verify`, `--force`.

## Paquetes nativos
Requiere `nfpm` (sin cgo):
```bash
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
make dist                 # tarballs + checksums.txt
make packages             # deb, rpm, apk, archlinux (amd64)
make release              # los cuatro paquetes para 4 arquitecturas
make rpm PKG_ARCH=arm64   # otra arquitectura
```
Si `nfpm` no está en `PATH`: `make packages NFPM=$$HOME/go/bin/nfpm`.

## Instalación por distro
| Distro | Comando |
|---|---|
| Debian/Ubuntu/Mint | `sudo apt install ./dist/sessions-control_*.deb` |
| Fedora/RHEL/Rocky | `sudo dnf install ./dist/sessions-control-*.rpm` |
| openSUSE | `sudo zypper install ./dist/sessions-control-*.rpm` |
| Alpine | `sudo apk add --allow-untrusted ./dist/sessions-control_*.apk` |
| Arch/Manjaro | `cd packaging && makepkg -si` |
| Cualquiera | `packaging/install.sh` |

## Arch Linux / AUR
```bash
cd packaging
makepkg -si          # compila desde el tag vX.Y.Z
```
Publicación en AUR: actualizar `pkgver`, regenerar `.SRCINFO` con
`makepkg --printsrcinfo > .SRCINFO` y subir `PKGBUILD` + `.SRCINFO`.

## Notas
- Binarios estáticos (`CGO_ENABLED=0`): sin dependencias de runtime.
- Arquitecturas: `amd64`, `arm64`, `armv7`, `386`.
- Etiquetas nativas: `amd64/x86_64`, `arm64/aarch64`, `armv7/armhf/armv7hl/armv7h`, `386/i386/i686/x86`.
- La app es TUI; no instala servicio ni demonio.
