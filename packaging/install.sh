#!/usr/bin/env bash
# Instalador universal de sessions-control para distribuciones Linux.
set -euo pipefail

REPO="${SESSIONS_CONTROL_REPO:-nestorPons/opencode-tui-sessions-control}"
BIN="sessions-control"

VERSION="${SESSIONS_CONTROL_VERSION:-}"
PREFIX=""
USER_INSTALL=0
FROM_SOURCE=0
UNINSTALL=0
NO_VERIFY=0
FORCE=0

usage() {
	cat <<'EOF'
Uso: install.sh [opciones]

Opciones:
  -v, --version VER   Versión a instalar (por defecto: última release)
  -p, --prefix DIR    Directorio de instalación (por defecto: /usr/local/bin)
  -u, --user          Instalar en ~/.local/bin (sin sudo)
      --from-source   Compilar desde el código fuente (requiere Go)
      --uninstall     Desinstalar el binario instalado
      --no-verify     No verificar el checksum sha256
      --force         Reinstalar aunque ya exista la misma versión
  -h, --help          Mostrar esta ayuda

Ejemplos:
  curl -fsSL https://raw.githubusercontent.com/nestorPons/opencode-tui-sessions-control/main/packaging/install.sh | bash
  ./packaging/install.sh --user
  ./packaging/install.sh --version 0.1.0 --prefix /opt/bin
  ./packaging/install.sh --from-source --user
EOF
}

log()  { printf '[sessions-control] %s\n' "$*"; }
warn() { printf '[sessions-control] aviso: %s\n' "$*" >&2; }
die()  { printf '[sessions-control] error: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
	case "$1" in
		-v|--version) VERSION="${2:?falta la versión}"; shift 2 ;;
		-p|--prefix) PREFIX="${2:?falta el directorio}"; shift 2 ;;
		-u|--user) USER_INSTALL=1; shift ;;
		--from-source) FROM_SOURCE=1; shift ;;
		--uninstall) UNINSTALL=1; shift ;;
		--no-verify) NO_VERIFY=1; shift ;;
		--force) FORCE=1; shift ;;
		-h|--help) usage; exit 0 ;;
		*) die "opción desconocida: $1 (usa --help)" ;;
	esac
done

[ "$(uname -s)" = "Linux" ] || die "este instalador solo soporta Linux"

ARCH="$(uname -m)"
case "$ARCH" in
	x86_64|amd64) ARCH=amd64 ;;
	aarch64|arm64) ARCH=arm64 ;;
	armv7l|armv7|armhf) ARCH=armv7 ;;
	i386|i486|i586|i686) ARCH=386 ;;
	*) die "arquitectura no soportada: $ARCH" ;;
esac

if [ -z "$PREFIX" ]; then
	if [ "$USER_INSTALL" = 1 ]; then
		PREFIX="$HOME/.local/bin"
	else
		PREFIX="/usr/local/bin"
	fi
fi

# Privilegios solo cuando el prefijo no es escribible por el usuario.
SUDO=""
if [ ! -w "$(dirname "$PREFIX")" ] && [ "$(id -u)" -ne 0 ]; then
	command -v sudo >/dev/null 2>&1 || die "se necesita sudo para escribir en $PREFIX"
	SUDO="sudo"
fi
run_as_root() { if [ -n "$SUDO" ]; then $SUDO "$@"; else "$@"; fi; }

if [ "$UNINSTALL" = 1 ]; then
	target="$PREFIX/$BIN"
	if [ -f "$target" ]; then
		run_as_root rm -f "$target"
		log "desinstalado $target"
	else
		log "no se encontró $target"
	fi
	exit 0
fi

distro_name() {
	if [ -r /etc/os-release ]; then
		# shellcheck disable=SC1091
		. /etc/os-release
		printf '%s' "${PRETTY_NAME:-${NAME:-Linux}}"
	else
		printf 'Linux'
	fi
}

source_deps_hint() {
	case "$(distro_name)" in
		*Debian*|*Ubuntu*|*Mint*|*Pop*)       echo "sudo apt install golang-go" ;;
		*Fedora*|*CentOS*|*RHEL*|*Rocky*)     echo "sudo dnf install golang" ;;
		*Arch*|*Manjaro*|*Endeavour*)         echo "sudo pacman -S go" ;;
		*Alpine*)                             echo "sudo apk add go" ;;
		*openSUSE*|*SUSE*)                    echo "sudo zypper install go" ;;
		*)                                     echo "instala Go desde https://go.dev/dl/" ;;
	esac
}

build_from_source() {
	command -v go >/dev/null 2>&1 || die "Go no está instalado. $(source_deps_hint)"
	command -v git >/dev/null 2>&1 || die "git no está instalado"
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	log "clonando $REPO"
	git clone --depth 1 https://github.com/"$REPO".git "$tmp/src" >/dev/null 2>&1
	if [ -n "$VERSION" ]; then
		git -C "$tmp/src" fetch --depth 1 origin "v$VERSION" >/dev/null 2>&1 &&
			git -C "$tmp/src" checkout -q FETCH_HEAD ||
			die "no existe la versión v$VERSION"
	fi
	log "compilando (linux/$ARCH)"
	( cd "$tmp/src" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$tmp/$BIN" ./cmd/sessions-control )
	install_binary "$tmp/$BIN"
}

fetch() {
	url="$1"
	out="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --retry 3 -o "$out" "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$out" "$url"
	else
		die "se necesita curl o wget"
	fi
}

install_binary() {
	src="$1"
	run_as_root install -d "$PREFIX"
	run_as_root install -m 0755 "$src" "$PREFIX/$BIN"
	log "instalado en $PREFIX/$BIN"
	if ! printf '%s' ":$PATH:" | grep -q ":$PREFIX:"; then
		warn "$PREFIX no está en PATH; añádelo a tu shell"
	fi
	"$PREFIX/$BIN" -version || true
}

latest_version() {
	url="https://api.github.com/repos/$REPO/releases/latest"
	if command -v curl >/dev/null 2>&1; then
		json="$(curl -fsSL "$url" 2>/dev/null || true)"
	else
		json="$(wget -qO- "$url" 2>/dev/null || true)"
	fi
	printf '%s' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' | head -n1
}

if [ "$FROM_SOURCE" = 1 ]; then
	build_from_source
	exit 0
fi

if [ -z "$VERSION" ]; then
	VERSION="$(latest_version)"
	if [ -z "$VERSION" ]; then
		warn "no se encontró ninguna release publicada"
		if command -v go >/dev/null 2>&1; then
			log "compilando desde fuente"
			build_from_source
			exit 0
		fi
		die "indica --version VER o instala Go para usar --from-source"
	fi
fi
VERSION="${VERSION#v}"

if [ "$FORCE" != 1 ] && [ -x "$PREFIX/$BIN" ]; then
	current="$("$PREFIX/$BIN" -version 2>/dev/null | awk '{print $2}' || true)"
	if [ "$current" = "$VERSION" ]; then
		log "ya está instalada la versión $VERSION en $PREFIX/$BIN"
		exit 0
	fi
fi

asset="${BIN}_${VERSION}_linux_${ARCH}.tar.gz"
base="https://github.com/$REPO/releases/download/v$VERSION"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log "distribución: $(distro_name) · arquitectura: $ARCH"
log "descargando $asset"
if ! fetch "$base/$asset" "$tmp/$asset"; then
	warn "no hay binario publicado para v$VERSION/$ARCH"
	if command -v go >/dev/null 2>&1; then
		log "compilando desde fuente"
		build_from_source
		exit 0
	fi
	die "instala Go y repite con --from-source"
fi

if [ "$NO_VERIFY" != 1 ]; then
	if fetch "$base/checksums.txt" "$tmp/checksums.txt"; then
		expected="$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums.txt")"
		if [ -n "$expected" ]; then
			if command -v sha256sum >/dev/null 2>&1; then
				actual="$(sha256sum "$tmp/$asset" | awk '{print $1}')"
			elif command -v shasum >/dev/null 2>&1; then
				actual="$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')"
			else
				warn "sin sha256sum/shasum; se omite la verificación"
				actual="$expected"
			fi
			[ "$actual" = "$expected" ] || die "checksum inválido para $asset"
			log "checksum verificado"
		else
			warn "no se encontró $asset en checksums.txt"
		fi
	else
		warn "no se pudo descargar checksums.txt; se omite la verificación"
	fi
fi

tar -C "$tmp" -xzf "$tmp/$asset"
src="$(find "$tmp" -maxdepth 2 -type f -name "$BIN" -perm -u+x | head -n1)"
[ -n "$src" ] || die "el archivo $asset no contiene el binario"
install_binary "$src"
log "versión $VERSION lista. Ejecuta: $BIN"
