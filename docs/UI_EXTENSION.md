# Rancher UI Extension (MakeCloud Node Driver)

UI extension лежит в `pkg/makecloud-node-driver`.

## Dev: запуск Rancher Dashboard локально с прокси на удалённый Rancher

Требуется: Node.js 20+, Yarn.

```bash
yarn install

# URL Rancher (например https://rancher.example.com)
API="https://<RANCHER_HOST>" yarn dev
```

После запуска открой `https://127.0.0.1:8005` (порт может отличаться — см. вывод `yarn dev`) и залогинься в удалённый Rancher.

Дальше:

- `Cluster Management -> Drivers -> Node Drivers -> makecloud`
- или создание `Machine Pool`/`Node Template` — форма `MakeCloud` должна отображаться с полями драйвера.

## Cloud Credential vs token в Machine Config

Extension поддерживает оба варианта:

- `addCloudCredential: false` — токен вводится прямо в Machine Config (поле `token`).
- `addCloudCredential: true` — токен и API параметры вводятся в Cloud Credential (поле `token`), а Machine Config содержит только параметры ВМ.

