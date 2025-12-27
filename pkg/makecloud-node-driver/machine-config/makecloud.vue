<script>
import Banner from "@components/Banner/Banner";
import { LabeledInput } from "@components/Form/LabeledInput";
import Checkbox from "@components/Form/Checkbox/Checkbox";
import { SECRET } from "@shell/config/types";
import LabeledSelect from "@shell/components/form/LabeledSelect";
import { parse as parseUrl } from "@shell/utils/url";

export default {
  components: {
    Banner,
    Checkbox,
    LabeledInput,
    LabeledSelect,
  },

  props: {
    uuid: {
      type: String,
      required: true,
    },
    mode: {
      type: String,
      required: true,
    },
    value: {
      type: Object,
      required: true,
    },
    credentialId: {
      type: String,
      required: false,
      default: "",
    },
  },

  async fetch() {
    await this.fetchCredential();
    await Promise.all([
      this.fetchTemplates(),
      this.fetchStorageProfiles(),
      this.fetchNetworks(),
      this.fetchFloatingPorts(),
      this.fetchFirewallTemplates(),
    ]);
  },

  data() {
    return {
      fetchingCount: 0,
      credential: null,
      driver: {},

      templates: null,
      storageProfiles: null,
      networks: null,
      floatingPorts: null,
      firewallTemplates: null,

      error: "",
      errorAllowHost: false,
      allowBusy: false,
    };
  },

  created() {
    if (!this.credentialId) {
      this.value.vdcId = this.value.vdcId ?? "";
    }
    this.value.templateId = this.value.templateId ?? "";
    this.value.templateName = this.value.templateName ?? "";
    this.value.storageProfileId = this.value.storageProfileId ?? "";
    this.value.networkId = this.value.networkId ?? "";
    this.value.floatingIp = this.value.floatingIp ?? "";
    this.value.allocateFloatingIp = this.value.allocateFloatingIp ?? false;

    if (!this.credentialId) {
      this.value.token = this.value.token ?? "";
      this.value.baseUrl = this.value.baseUrl ?? "https://cp.iteco.cloud";
      this.value.insecure = this.value.insecure ?? false;
    }

    this.value.cpu = (this.value.cpu ?? "").toString() || "2";
    this.value.ram = (this.value.ram ?? "").toString() || "4";
    this.value.diskSize = (this.value.diskSize ?? "").toString() || "40";

    const firewallTemplateId = this.value.firewallTemplateId;
    if (Array.isArray(firewallTemplateId)) {
      this.value.firewallTemplateId = firewallTemplateId;
    } else if (typeof firewallTemplateId === "string") {
      this.setStringArray("firewallTemplateId", firewallTemplateId);
    } else {
      this.value.firewallTemplateId = [];
    }
    this.value.tags = this.value.tags ?? [];
    this.value.metadata = this.value.metadata ?? [];
    this.value.userData = this.value.userData ?? "";
    this.value.noInjectSshKey = this.value.noInjectSshKey ?? false;
    this.value.waitTimeout = (this.value.waitTimeout ?? "").toString() || "600";

    this.value.sshUser = this.value.sshUser ?? "root";
    this.value.sshPort = (this.value.sshPort ?? "").toString() || "22";

    this.validate();
  },

  watch: {
    async credentialId() {
      await this.$fetch();
      this.validate();
    },
    "value.allocateFloatingIp"(neu) {
      if (neu) {
        this.value.floatingIp = "";
      }
      this.applyDefaultFirewallTemplates();
    },
    "value.floatingIp"() {
      this.applyDefaultFirewallTemplates();
    },
    value: {
      deep: true,
      handler() {
        this.validate();
      },
    },
    templates() {
      if (!this.templates || this.templates.length === 0) {
        return;
      }

      if (!this.value.templateId && !this.value.templateName) {
        const preferred = this.templates.find((t) => {
          const name = (t.name || "").toLowerCase();

          return name.includes("ubuntu") && name.includes("24");
        });
        if (preferred?.id) {
          this.value.templateId = preferred.id;
          return;
        }
      }

      if (this.templates.length === 1 && !this.value.templateId) {
        this.value.templateId = this.templates[0]?.id || "";
      }
      if (
        this.value.templateId &&
        !this.templates.find((t) => t.id === this.value.templateId)
      ) {
        // keep user-entered value if templateName is used
        if (!this.value.templateName) {
          this.value.templateId = "";
        }
      }
    },
    storageProfiles() {
      if (!this.storageProfiles || this.storageProfiles.length === 0) {
        return;
      }

      if (!this.value.storageProfileId) {
        const preferred = this.storageProfiles.find((sp) =>
          (sp.name || "").toLowerCase().includes("ssd")
        );
        if (preferred?.id) {
          this.value.storageProfileId = preferred.id;
          return;
        }
      }

      if (this.storageProfiles.length === 1 && !this.value.storageProfileId) {
        this.value.storageProfileId = this.storageProfiles[0]?.id || "";
      }
      if (
        this.value.storageProfileId &&
        !this.storageProfiles.find(
          (sp) => sp.id === this.value.storageProfileId
        )
      ) {
        this.value.storageProfileId = "";
      }
    },
    firewallTemplates() {
      this.applyDefaultFirewallTemplates();
    },
  },

  computed: {
    disabled() {
      return this.mode === "view";
    },
    hasCredential() {
      return !!this.credentialId;
    },
    hasApiAccess() {
      return !!this.credential && !this.credential.insecure;
    },
    makecloudHost() {
      const baseUrl = this.credential?.baseUrl || "";
      const u = parseUrl(baseUrl);

      return u?.host || baseUrl.replace(/^https?:\/\//, "");
    },
    templateOptions() {
      if (!this.templates) {
        return null;
      }

      const sorted = [...this.templates].sort((a, b) =>
        (a.name || "").localeCompare(b.name || "")
      );

      return sorted.map((t) => ({
        label: `${t.name} (${t.id})`,
        value: t.id,
      }));
    },
    storageProfileOptions() {
      if (!this.storageProfiles) {
        return null;
      }

      const sorted = [...this.storageProfiles].sort((a, b) =>
        (a.name || "").localeCompare(b.name || "")
      );

      return sorted.map((sp) => ({
        label: `${sp.name} (${sp.id})`,
        value: sp.id,
      }));
    },
    networkOptions() {
      if (!this.networks) {
        return null;
      }

      const sorted = [...this.networks].sort((a, b) =>
        (a.name || "").localeCompare(b.name || "")
      );

      const auto = { label: "Auto (VDC default)", value: "" };
      const opts = sorted.map((n) => ({
        label: `${n.name} (${n.id})${n.is_default ? " [default]" : ""}`,
        value: n.id,
      }));

      return [auto, ...opts];
    },
    floatingIpOptions() {
      if (!this.floatingPorts) {
        return null;
      }

      const freeOnly = this.floatingPorts.filter(
        (p) => !p.connected || !p.connected.id
      );
      const sorted = [...freeOnly].sort((a, b) => {
        const aIp = a.ip_address || "";
        const bIp = b.ip_address || "";
        return aIp.localeCompare(bIp);
      });

      const none = { label: "None", value: "" };
      const opts = sorted.map((p) => {
        const ip = p.ip_address || "";
        return {
          label: `${ip || p.id} (${p.id})`,
          value: p.id,
        };
      });

      return [none, ...opts];
    },
    firewallTemplateOptions() {
      if (!this.firewallTemplates) {
        return null;
      }

      const sorted = [...this.firewallTemplates].sort((a, b) =>
        (a.name || "").localeCompare(b.name || "")
      );

      return sorted.map((ft) => ({
        label: `${ft.name} (${ft.id})`,
        value: ft.id,
      }));
    },
  },

  methods: {
    applyDefaultFirewallTemplates() {
      if (!this.hasCredential || !Array.isArray(this.firewallTemplates)) {
        return;
      }

      const current = Array.isArray(this.value.firewallTemplateId)
        ? this.value.firewallTemplateId
        : [];

      const egressNames = [
        "Разрешить все исходящие соединения",
        "По-умолчанию",
        "По умолчанию",
        "Default",
      ];
      const webNames = [
        "Разрешить WEB порты, доступные из Интернета",
        "Разрешить WEB порты",
        "Разрешить WEB",
      ];

      const findByNames = (names) => {
        for (const want of names) {
          const w = (want || "").toLowerCase().trim();
          if (!w) {
            continue;
          }

          for (const ft of this.firewallTemplates || []) {
            if (!ft) {
              continue;
            }
            const n = (ft.name || "").toLowerCase().trim();
            if (!n) {
              continue;
            }
            if (n === w || n.includes(w) || w.includes(n)) {
              return ft;
            }
          }
        }
        return null;
      };

      const egress = findByNames(egressNames);
      const web = findByNames(webNames);
      const wantPublic =
        !!this.value.allocateFloatingIp ||
        (this.value.floatingIp || "").toString().trim() !== "";

      const set = (ids) => {
        const uniq = Array.from(
          new Set(
            (ids || []).map((x) => (x || "").toString().trim()).filter(Boolean)
          )
        );
        this.value.firewallTemplateId = uniq;
      };

      if (current.length === 0) {
        const ids = [];
        if (egress?.id) {
          ids.push(egress.id);
        }
        if (wantPublic && web?.id) {
          ids.push(web.id);
        }
        if (ids.length > 0) {
          set(ids);
        }
        return;
      }

      // If we previously auto-selected only egress, and user later enabled public IP,
      // auto-add WEB template once.
      if (
        wantPublic &&
        web?.id &&
        !current.includes(web.id) &&
        egress?.id &&
        current.length === 1 &&
        current[0] === egress.id
      ) {
        set([egress.id, web.id]);
      }
    },

    async fetchCredential() {
      if (!this.credentialId) {
        this.credential = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        this.error = "";
        this.errorAllowHost = false;

        const secret = await this.$store.dispatch("management/find", {
          type: SECRET,
          id: this.credentialId.replace(":", "/"),
        });

        const decodedData = {
          token: atob(secret.data["makecloudcredentialConfig-token"] || ""),
          baseUrl: atob(secret.data["makecloudcredentialConfig-baseUrl"] || ""),
          insecure:
            atob(secret.data["makecloudcredentialConfig-insecure"] || "") ===
            "true",
          vdcId: atob(secret.data["makecloudcredentialConfig-vdcId"] || ""),
        };

        if (decodedData.insecure) {
          throw new Error("Insecure TLS is enabled");
        }

        this.credential = decodedData;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn(
          "Failed to fetch MakeCloud credential, enhanced form fields are disabled",
          e
        );
        this.credential = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    async fetchFromMakecloud(path, query = {}) {
      const host = this.makecloudHost;
      const params = new URLSearchParams(query);
      const url = params.toString()
        ? `/meta/proxy/${host}/${path}?${params.toString()}`
        : `/meta/proxy/${host}/${path}`;

      return await this.$store.dispatch(
        "management/request",
        {
          method: "GET",
          url,
          headers: {
            Accept: "application/json",
            "X-API-Auth-Header": `Bearer ${this.credential.token}`,
          },
          redirectUnauthorized: false,
        },
        { root: true }
      );
    },

    async fetchPagedItems(path, baseQuery = {}) {
      const out = [];

      let page = 1;
      for (;;) {
        const res = await this.fetchFromMakecloud(path, {
          ...baseQuery,
          page: page.toString(),
        });
        const items = res?.items || [];
        out.push(...items);

        const total = res?.total ?? out.length;
        if (out.length >= total || items.length === 0) {
          break;
        }

        page += 1;
      }

      return out;
    },

    async fetchTemplates() {
      if (!this.hasApiAccess) {
        this.templates = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        const res = await this.fetchFromMakecloud("v1/template", {
          vdc: this.credential.vdcId,
        });

        const list = Array.isArray(res)
          ? res
          : Array.isArray(res?.items)
          ? res.items
          : [];

        this.templates = list;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn("Failed to fetch templates", e);
        this.templates = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    async fetchStorageProfiles() {
      if (!this.hasApiAccess) {
        this.storageProfiles = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        this.storageProfiles = await this.fetchPagedItems(
          "v1/storage_profile",
          {
            vdc: this.credential.vdcId,
          }
        );
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn("Failed to fetch storage profiles", e);
        this.storageProfiles = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    async fetchNetworks() {
      if (!this.hasApiAccess) {
        this.networks = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        this.networks = await this.fetchPagedItems("v1/network", {
          vdc: this.credential.vdcId,
        });
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn("Failed to fetch networks", e);
        this.networks = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    async fetchFloatingPorts() {
      if (!this.hasApiAccess) {
        this.floatingPorts = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        const ports = await this.fetchPagedItems("v1/port", {
          vdc: this.credential.vdcId,
          filter_type: "external",
        });

        this.floatingPorts = ports;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn("Failed to fetch floating IPs", e);
        this.floatingPorts = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    async fetchFirewallTemplates() {
      if (!this.hasApiAccess) {
        this.firewallTemplates = null;
        return;
      }

      try {
        this.fetchingCount += 1;
        this.firewallTemplates = await this.fetchPagedItems("v1/firewall", {
          vdc: this.credential.vdcId,
        });
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn("Failed to fetch firewall templates", e);
        this.firewallTemplates = null;
      } finally {
        this.fetchingCount -= 1;
      }
    },

    validate() {
      const hasTemplate =
        (this.value.templateId && this.value.templateId.trim() !== "") ||
        (this.value.templateName && this.value.templateName.trim() !== "");

      if (!this.hasCredential) {
        if (!this.value.token || this.value.token.trim() === "") {
          this.$emit("validationChanged", false);
          return;
        }

        if (!this.value.vdcId || this.value.vdcId.trim() === "") {
          this.$emit("validationChanged", false);
          return;
        }
      }

      if (!hasTemplate) {
        this.$emit("validationChanged", false);
        return;
      }

      if (
        this.hasCredential &&
        this.storageProfiles &&
        this.storageProfiles.length > 1 &&
        (!this.value.storageProfileId ||
          this.value.storageProfileId.trim() === "")
      ) {
        this.$emit("validationChanged", false);
        return;
      }

      const cpu = parseInt((this.value.cpu || "").toString(), 10);
      if (!cpu || cpu < 1) {
        this.$emit("validationChanged", false);
        return;
      }

      const ram = parseFloat((this.value.ram || "").toString());
      if (!ram || ram <= 0) {
        this.$emit("validationChanged", false);
        return;
      }

      const diskSize = parseInt((this.value.diskSize || "").toString(), 10);
      if (!diskSize || diskSize < 1) {
        this.$emit("validationChanged", false);
        return;
      }

      this.$emit("validationChanged", true);
    },

    setStringArray(field, value) {
      const out = (value || "")
        .split(/[\n,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      this.value[field] = out;
    },
  },
};
</script>

<template>
  <div>
    <template v-if="!hasCredential">
      <h3>
        <t k="cluster.machineConfig.makecloud.api.header" />
      </h3>

      <div class="row mb-20">
        <div class="col span-6">
          <LabeledInput
            type="password"
            :mode="mode"
            :disabled="disabled"
            :value="value.token"
            @change="
              (e) => {
                value.token = e.target.value;
              }
            "
            label-key="cluster.machineConfig.makecloud.api.token.label"
            required
          />
        </div>
        <div class="col span-6">
          <LabeledInput
            type="text"
            :mode="mode"
            :disabled="disabled"
            :value="value.baseUrl"
            @change="
              (e) => {
                value.baseUrl = e.target.value;
              }
            "
            label-key="cluster.machineConfig.makecloud.api.baseUrl.label"
          />
        </div>
      </div>

      <div class="row mb-20">
        <div class="col span-6">
          <Checkbox
            :mode="mode"
            :disabled="disabled"
            :value="value.insecure"
            @click="
              (e) => {
                value.insecure = !value.insecure;
              }
            "
            label-key="cluster.machineConfig.makecloud.api.insecureTLS.label"
          />
        </div>
      </div>
    </template>

    <h3>
      <t k="cluster.machineConfig.makecloud.vm.header" />
    </h3>

    <div class="row mb-10">
      <div v-if="!hasCredential" class="col span-6">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.vdcId"
          @change="
            (e) => {
              value.vdcId = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.vdcId.label"
          required
        />
      </div>
      <div :class="hasCredential ? 'col span-12' : 'col span-6'">
        <LabeledSelect
          v-if="hasCredential && storageProfileOptions"
          :mode="mode"
          :disabled="disabled"
          v-model:value="value.storageProfileId"
          :options="storageProfileOptions"
          label-key="cluster.machineConfig.makecloud.vm.storageProfileId.label"
          :required="storageProfiles && storageProfiles.length > 1"
        />

        <LabeledInput
          v-else
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.storageProfileId"
          @change="
            (e) => {
              value.storageProfileId = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.storageProfileId.label"
        />
      </div>
    </div>

    <div class="row mb-10">
      <div class="col span-6">
        <LabeledSelect
          v-if="hasCredential && templateOptions"
          :mode="mode"
          :disabled="disabled"
          v-model:value="value.templateId"
          :options="templateOptions"
          label-key="cluster.machineConfig.makecloud.vm.templateId.label"
        />

        <LabeledInput
          v-else
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.templateId"
          @change="
            (e) => {
              value.templateId = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.templateId.label"
        />
      </div>
      <div class="col span-6">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.templateName"
          @change="
            (e) => {
              value.templateName = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.templateName.label"
        />
      </div>
    </div>

    <div class="row mb-20">
      <div class="col span-6">
        <LabeledSelect
          v-if="hasCredential && networkOptions"
          :mode="mode"
          :disabled="disabled"
          v-model:value="value.networkId"
          :options="networkOptions"
          label-key="cluster.machineConfig.makecloud.vm.networkId.label"
        />

        <LabeledInput
          v-else
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.networkId"
          @change="
            (e) => {
              value.networkId = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.networkId.label"
        />
      </div>
      <div class="col span-6">
        <Checkbox
          :mode="mode"
          :disabled="disabled"
          :value="value.allocateFloatingIp"
          @click="
            (e) => {
              value.allocateFloatingIp = !value.allocateFloatingIp;
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.allocateFloatingIp.label"
        />

        <div
          class="floating-ip-wrapper"
          :class="{ 'is-hidden': value.allocateFloatingIp }"
        >
          <LabeledSelect
            v-if="hasCredential && floatingIpOptions"
            :mode="mode"
            :disabled="disabled || value.allocateFloatingIp"
            v-model:value="value.floatingIp"
            :options="floatingIpOptions"
            label-key="cluster.machineConfig.makecloud.vm.floatingIp.label"
          />

          <LabeledInput
            v-else
            type="text"
            :mode="mode"
            :disabled="disabled || value.allocateFloatingIp"
            :value="value.floatingIp"
            @change="
              (e) => {
                value.floatingIp = e.target.value;
              }
            "
            label-key="cluster.machineConfig.makecloud.vm.floatingIp.label"
          />
        </div>
      </div>
    </div>

    <div class="row mb-20">
      <div class="col span-12">
        <LabeledSelect
          v-if="hasCredential && firewallTemplateOptions"
          :mode="mode"
          :disabled="disabled"
          v-model:value="value.firewallTemplateId"
          :options="firewallTemplateOptions"
          :multiple="true"
          :searchable="true"
          :close-on-select="false"
          label-key="cluster.machineConfig.makecloud.vm.firewallTemplateIds.label"
          tooltip-key="cluster.machineConfig.makecloud.vm.firewallTemplateIds.tooltip"
        />

        <LabeledInput
          v-else
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="
            (Array.isArray(value.firewallTemplateId)
              ? value.firewallTemplateId
              : []
            ).join(',')
          "
          @change="
            (e) => {
              setStringArray('firewallTemplateId', e.target.value);
            }
          "
          label-key="cluster.machineConfig.makecloud.vm.firewallTemplateIds.label"
          tooltip-key="cluster.machineConfig.makecloud.vm.firewallTemplateIds.tooltip"
        />
      </div>
    </div>

    <h3>
      <t k="cluster.machineConfig.makecloud.resources.header" />
    </h3>

    <div class="row mb-20">
      <div class="col span-4">
        <LabeledInput
          type="number"
          :mode="mode"
          :disabled="disabled"
          min="1"
          step="1"
          :value="value.cpu"
          @change="
            (e) => {
              value.cpu = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.resources.cpu.label"
          required
        />
      </div>
      <div class="col span-4">
        <LabeledInput
          type="number"
          :mode="mode"
          :disabled="disabled"
          min="0.1"
          step="0.1"
          :value="value.ram"
          @change="
            (e) => {
              value.ram = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.resources.ram.label"
          required
        />
      </div>
      <div class="col span-4">
        <LabeledInput
          type="number"
          :mode="mode"
          :disabled="disabled"
          min="1"
          step="1"
          :value="value.diskSize"
          @change="
            (e) => {
              value.diskSize = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.resources.diskSize.label"
          required
        />
      </div>
    </div>

    <h3>
      <t k="cluster.machineConfig.makecloud.ssh.header" />
    </h3>

    <div class="row mb-20">
      <div class="col span-6">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.sshUser"
          @change="
            (e) => {
              value.sshUser = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.ssh.user.label"
        />
      </div>
      <div class="col span-6">
        <LabeledInput
          type="number"
          :mode="mode"
          :disabled="disabled"
          min="1"
          step="1"
          :value="value.sshPort"
          @change="
            (e) => {
              value.sshPort = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.ssh.port.label"
        />
      </div>
    </div>

    <portal :to="`advanced-${uuid}`">
      <h3>
        <t k="cluster.machineConfig.makecloud.advanced.header" />
      </h3>

      <div class="mb-20">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="(value.tags || []).join(',')"
          @change="
            (e) => {
              setStringArray('tags', e.target.value);
            }
          "
          label-key="cluster.machineConfig.makecloud.advanced.tags.label"
        />
      </div>

      <div class="mb-20">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="(value.metadata || []).join('\\n')"
          @change="
            (e) => {
              setStringArray('metadata', e.target.value);
            }
          "
          label-key="cluster.machineConfig.makecloud.advanced.metadata.label"
        />
      </div>

      <div class="mb-20">
        <LabeledInput
          type="text"
          :mode="mode"
          :disabled="disabled"
          :value="value.userData"
          @change="
            (e) => {
              value.userData = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.advanced.userData.label"
        />
      </div>

      <div class="mb-20">
        <Checkbox
          :mode="mode"
          :disabled="disabled"
          :value="value.noInjectSshKey"
          @click="
            (e) => {
              value.noInjectSshKey = !value.noInjectSshKey;
            }
          "
          label-key="cluster.machineConfig.makecloud.advanced.noInjectSshKey.label"
        />
      </div>

      <div class="mb-20">
        <LabeledInput
          type="number"
          :mode="mode"
          :disabled="disabled"
          min="1"
          step="1"
          :value="value.waitTimeout"
          @change="
            (e) => {
              value.waitTimeout = e.target.value;
            }
          "
          label-key="cluster.machineConfig.makecloud.advanced.waitTimeout.label"
        />
      </div>
    </portal>
  </div>
</template>

<style lang="scss" scoped>
.floating-ip-wrapper.is-hidden {
  visibility: hidden;
  pointer-events: none;
}
</style>
