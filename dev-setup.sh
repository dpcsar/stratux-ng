#!/usr/bin/env bash

set -euo pipefail

[[ "${TRACE:-0}" == "1" ]] && set -x

SET_DEFAULT_SHELL=0

log() {
	printf '[dev-setup] %s\n' "$*"
}

err() {
	printf '[dev-setup] ERROR: %s\n' "$*" >&2
}

die() {
	err "$1"
	exit "${2:-1}"
}

usage() {
	cat <<'EOF'
Usage: dev-setup.sh [options]

Options:
  --set-default-shell   Run chsh to set zsh as the default shell for the current user.
  -h, --help            Show this help text and exit.
EOF
}

parse_args() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--set-default-shell)
			SET_DEFAULT_SHELL=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			die "Unknown option: $1"
			;;
		esac
	done
}

require_commands() {
	local missing=()
	for cmd in "$@"; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done
	if ((${#missing[@]})); then
		die "Missing required commands: ${missing[*]}"
	fi
}

ensure_sudo() {
	if [[ "${EUID}" -ne 0 ]]; then
		require_commands sudo
		sudo -v
	fi
}

install_packages() {
	local packages=(
		git curl ca-certificates zsh neovim build-essential pkg-config golang-go
	)
	if command -v apt-get >/dev/null 2>&1; then
		ensure_sudo
		log "Updating apt cache"
		sudo apt-get update -y
		log "Installing packages: ${packages[*]}"
		sudo apt-get install -y --no-install-recommends "${packages[@]}"
	else
		die "Unsupported package manager. Only apt-based systems are currently supported."
	fi
}

install_docker() {
	if command -v docker >/dev/null 2>&1; then
		log "docker already installed"
		return
	fi
	ensure_sudo
	if command -v apt-get >/dev/null 2>&1; then
		log "Installing Docker Engine via apt (docker.io)"
		if ! sudo apt-get install -y --no-install-recommends docker.io docker-buildx docker-compose; then
			log "apt install failed; falling back to get.docker.com script"
		else
			return
		fi
	fi
	local installer="/tmp/get-docker.sh"
	log "Installing Docker using official convenience script"
	curl -fsSL https://get.docker.com -o "${installer}"
	sudo sh "${installer}"
	rm -f "${installer}"
}

ensure_docker_ready() {
	if ! command -v docker >/dev/null 2>&1; then
		die "docker command missing even after package install"
	fi
	ensure_sudo
	if ! getent group docker >/dev/null; then
		log "Creating docker group"
		sudo groupadd docker
	fi
	if ! id -nG "$USER" | grep -q '\bdocker\b'; then
		log "Adding ${USER} to docker group"
		sudo usermod -aG docker "$USER"
		log "You'll need to log out/in for docker group membership to take effect"
	fi
	log "Enabling docker service"
	sudo systemctl enable --now docker >/dev/null 2>&1 || sudo systemctl start docker
}

install_oh_my_zsh() {
	local zsh_dir="${ZSH:-$HOME/.oh-my-zsh}"
	if [[ -d "${zsh_dir}" ]]; then
		log "oh-my-zsh already installed"
		return
	fi
	log "Installing oh-my-zsh"
	export RUNZSH=no
	export CHSH=no
	export KEEP_ZSHRC=yes
	sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
}

ensure_repo() {
	local repo_url="$1"
	local destination="$2"
	if [[ -d "${destination}/.git" ]]; then
		log "Updating ${destination}"
		git -C "${destination}" pull --ff-only
	elif [[ -d "${destination}" ]]; then
		die "${destination} exists but is not a git repo"
	else
		log "Cloning ${repo_url} -> ${destination}"
		git clone "${repo_url}" "${destination}"
	fi
}

install_plugins() {
	local zsh_custom="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"
	mkdir -p "${zsh_custom}/plugins" "${zsh_custom}/themes"

	local plugins=(
		"https://github.com/zsh-users/zsh-autosuggestions|${zsh_custom}/plugins/zsh-autosuggestions"
		"https://github.com/zsh-users/zsh-syntax-highlighting|${zsh_custom}/plugins/zsh-syntax-highlighting"
		"https://github.com/zsh-users/zsh-completions|${zsh_custom}/plugins/zsh-completions"
	)

	local theme="https://github.com/romkatv/powerlevel10k.git|${zsh_custom}/themes/powerlevel10k"

	for entry in "${plugins[@]}"; do
		local repo="${entry%%|*}"
		local dest="${entry#*|}"
		ensure_repo "${repo}" "${dest}"
	done

	local theme_repo="${theme%%|*}"
	local theme_dest="${theme#*|}"
	ensure_repo "${theme_repo}" "${theme_dest}"
}

install_atuin() {
	if command -v atuin >/dev/null 2>&1; then
		log "atuin already installed"
		return
	fi
	log "Installing atuin"
	curl --proto '=https' --tlsv1.2 -sSf https://setup.atuin.sh | bash
}

update_zshrc() {
	local zshrc="$HOME/.zshrc"
	local block_start="# >>> stratux-ng dev setup >>>"
	local block_end="# <<< stratux-ng dev setup <<<"
	local tmp_block
	tmp_block="$(mktemp)"

	cat >"${tmp_block}" <<'EOF'
# >>> stratux-ng dev setup >>>
ZSH=${ZSH:-$HOME/.oh-my-zsh}
ZSH_THEME="powerlevel10k/powerlevel10k"
plugins=(git gh vi-mode zsh-autosuggestions zsh-syntax-highlighting zsh-completions)

source "$ZSH/oh-my-zsh.sh"

[[ -r "$HOME/.p10k.zsh" ]] && source "$HOME/.p10k.zsh"
command -v atuin >/dev/null 2>&1 && eval "$(atuin init zsh)"
# <<< stratux-ng dev setup <<<
EOF

	if [[ -f "${zshrc}" ]]; then
		if grep -q "${block_start}" "${zshrc}"; then
			log "Updating existing stratux-ng block in ${zshrc}"
			awk -v start="${block_start}" -v end="${block_end}" -v file="${tmp_block}" '
				BEGIN {printed=0}
				$0==start {while ((getline line < file) > 0) print line; in_block=1; next}
				$0==end {in_block=0; next}
				in_block {next}
				{print}
			' "${zshrc}" >"${zshrc}.tmp"
			mv "${zshrc}.tmp" "${zshrc}"
		else
			log "Appending stratux-ng block to ${zshrc}"
			{
				printf '\n'
				cat "${tmp_block}"
			} >>"${zshrc}"
		fi
	else
		log "Creating ${zshrc}"
		cp "${tmp_block}" "${zshrc}"
	fi

	rm -f "${tmp_block}"
}

maybe_set_default_shell() {
	[[ "${SET_DEFAULT_SHELL}" == "1" ]] || return
	command -v chsh >/dev/null 2>&1 || die "chsh not available; cannot change default shell"
	local zsh_path
	zsh_path="$(command -v zsh)" || die "zsh binary not found"
	if [[ "${SHELL:-}" == "${zsh_path}" ]]; then
		log "Default shell already ${zsh_path}"
		return
	fi
	log "Setting default shell to ${zsh_path}"
	chsh -s "${zsh_path}" "${USER}"
	log "Default shell updated; log out/in for changes to take effect."
}

main() {
	parse_args "$@"
	require_commands curl git 
	install_packages
	install_docker
	install_oh_my_zsh
	install_plugins
	install_atuin
	update_zshrc
	ensure_docker_ready
	maybe_set_default_shell

	log "Development environment bootstrap complete."
	if [[ "${SET_DEFAULT_SHELL}" == "1" ]]; then
		log "Default shell change requested during this run."
	else
		log "Run with --set-default-shell to switch your login shell to zsh."
	fi
}

main "$@"
