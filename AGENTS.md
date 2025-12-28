# AGENTS.md

Этот репозиторий содержит:

- Go docker-machine driver `makecloud` (CLI: `cmd/docker-machine-driver-makecloud`, пакет: `makecloud/`)
- Rancher UI Extension (пакет: `pkg/makecloud-node-driver/`)

## Общие правила

- Не добавляй в репозиторий секреты (API токены, kubeconfig, приватные ключи, пароли).
- Не создавай папку `docs/` (документация ведётся в `README.md`).
- Не коммить артефакты сборки/публикации: `dist/`, `dist-pkg/`, `node_modules/`, `artifacts/`, `tmp/`, `assets/`, `charts/`, `extensions/`, `index.yaml`.
- Для изменений в CI придерживайся текущей схемы: reusable workflow Rancher для UI extension и отдельный workflow для Go-бинарей.

## Go driver (docker-machine / rancher-machine)

- Сборка локально: `go build -o docker-machine-driver-makecloud ./cmd/docker-machine-driver-makecloud`
- Релизы бинарей: теги `vX.Y.Z` → GitHub Release → `.github/workflows/release.yml` собирает и прикрепляет архивы.

## Rancher UI Extension

- Требования: Node.js 20+ (см. `.nvmrc`), Yarn classic.
- Dev: `API="https://<RANCHER_HOST>" yarn dev`

### Публикация UI extension (Helm repo в `gh-pages`)

- Версия берётся из `pkg/makecloud-node-driver/package.json`.
- Релизы UI extension: теги вида `makecloud-node-driver-X.Y.Z` → GitHub Release → `.github/workflows/build-extension-charts.yml`.
- Важно: chart version (= `X.Y.Z`) по умолчанию используется как версия драйвера (URL строится на базе `vX.Y.Z`), поэтому держи версии синхронизированными или переопредели:
  - `--set nodeDriver.versionOverride=<driverVersion>` или
  - `--set nodeDriver.url=<customUrl>`.

### NodeDriver в Helm chart

При публикации UI extension в chart добавляется ещё один ресурс `kind: NodeDriver`:

- Шаблон: `pkg/makecloud-node-driver/chart-templates/nodedriver.yaml`
- Инжект/перепаковка происходит в обёртке: `scripts/publish-pkgs.sh` (вызывается через `yarn publish-pkgs` из корневого `package.json`)
- Отключение NodeDriver в чарте: `--set nodeDriver.enabled=false`
