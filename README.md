# docker-machine-driver-makecloud

Docker Machine driver для MakeCloud (BCC) на базе Go SDK `github.com/basis-cloud/bcc-go`.

Бинарник предназначен для использования через `docker-machine`/`rancher-machine` (например, в Rancher через node driver).

## Сборка

```bash
go build -o docker-machine-driver-makecloud ./cmd/docker-machine-driver-makecloud
```

## Установка в Rancher (NodeDriver)

Пример манифеста `NodeDriver` (подставь актуальный тег/ссылку на архив под Linux):

```yaml
apiVersion: management.cattle.io/v3
kind: NodeDriver
metadata:
  annotations:
    privateCredentialFields: token
  name: makecloud
spec:
  active: true
  addCloudCredential: false
  builtin: false
  checksum: ''
  description: ''
  displayName: makecloud
  externalId: ''
  uiUrl: ''
  url: https://github.com/gorizond/docker-machine-driver-makecloud/releases/download/v0.1.0/docker-machine-driver-makecloud_v0.1.0_linux_amd64.tar.gz
  whitelistDomains: []
```

## Пример использования (docker-machine / rancher-machine)

```bash
docker-machine create -d makecloud \
  --makecloud-token "$MAKECLOUD_TOKEN" \
  --makecloud-vdc-id "$MAKECLOUD_VDC_ID" \
  --makecloud-template-id "$MAKECLOUD_TEMPLATE_ID" \
  --makecloud-storage-profile-id "$MAKECLOUD_STORAGE_PROFILE_ID" \
  --makecloud-cpu 2 \
  --makecloud-ram "4" \
  --makecloud-disk-size 40 \
  --makecloud-network-id "$MAKECLOUD_NETWORK_ID" \
  my-node-01
```

## Важные параметры

- `--makecloud-token` — API token (обязателен)
- `--makecloud-base-url` — базовый URL API (по умолчанию `https://cp.iteco.cloud`)
- `--makecloud-vdc-id` — VDC ID (обязателен)
- `--makecloud-template-id` / `--makecloud-template-name` — шаблон VM
- `--makecloud-storage-profile-id` — storage profile (если в VDC ровно один, можно не задавать)
- `--makecloud-cpu` — CPU cores
- `--makecloud-ram` — RAM в GiB (строка, допускает float)
- `--makecloud-disk-size` — root disk size в GiB
- `--makecloud-network-id` — сеть (если не задано, берётся default сеть VDC)
- `--makecloud-firewall-template-id` — шаблоны FW, которые прикрепляются к порту VM (повторяемый). Если не задано — драйвер пытается применить дефолтный шаблон для исходящих (`По-умолчанию` / `Разрешить все исходящие соединения`). При использовании `--makecloud-floating-ip` по умолчанию добавляется ещё шаблон WEB.
- `--makecloud-floating-ip` — floating IP (адрес или ID). Если задан, драйвер предпочитает его для подключения.
- `--makecloud-user-data` — cloud-init user-data (строкой, `@path` или просто `path`)
- `--makecloud-no-inject-ssh-key` — отключить автодобавление SSH ключа через cloud-init (по умолчанию включено)
- `--makecloud-metadata` — значения template fields в формате `key=value` (ключ может быть field ID / system_alias / name; повторяемый)

## Примечания по доступности

- Драйвер выбирает IP в таком порядке: `floating.ip_address` → первый `port.ip_address`.
- Если VM создаётся только во внутренней сети без floating IP и без маршрутизации наружу — `docker-machine`/`rancher-machine` не смогут подключиться по SSH.
- При использовании `--makecloud-floating-ip` драйвер также пытается применить на floating-порту шаблон FW для WEB (если он есть в VDC: `Разрешить WEB`). Для SSH добавь нужный шаблон через `--makecloud-firewall-template-id`.
