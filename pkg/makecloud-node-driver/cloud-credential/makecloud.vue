<script>
import Banner from "@components/Banner/Banner";
import { Checkbox } from "@components/Form/Checkbox";
import { LabeledInput } from "@components/Form/LabeledInput";
import LabeledSelect from "@shell/components/form/LabeledSelect";
import { parse as parseUrl } from "@shell/utils/url";

import BusyButton from "../components/BusyButton.vue";

export default {
  components: {
    Banner,
    BusyButton,
    Checkbox,
    LabeledInput,
    LabeledSelect,
  },

  props: {
    mode: {
      type: String,
      required: true,
    },
    value: {
      type: Object,
      required: true,
    },
  },

  async fetch() {
    this.driver = await this.$store.dispatch("rancher/find", {
      type: "nodedriver",
      id: "makecloud",
    });
  },

  data() {
    return {
      vdcs: null,
      vdcId: this.value.decodedData.vdcId ?? "",
      step: this.value.decodedData.vdcId ?? "" ? 2 : 1,
      busy: false,
      errorAllowHost: false,
      allowBusy: false,
      error: "",
      driver: {},
    };
  },

  computed: {
    hostname() {
      const u = parseUrl(this.value.decodedData.baseUrl);

      return u?.host || "";
    },

    canAuthenticate() {
      return (
        !!this.value?.decodedData?.baseUrl && !!this.value?.decodedData?.token
      );
    },

    vdcOptions() {
      const sorted = (this.vdcs || []).sort((a, b) =>
        a.name.localeCompare(b.name)
      );

      return sorted.map((vdc) => {
        return {
          label: `${vdc.name} (${vdc.id})`,
          value: vdc.id,
        };
      });
    },
  },

  created() {
    this.value.setData("token", this.value.decodedData.token ?? "");
    this.value.setData(
      "baseUrl",
      this.value.decodedData.baseUrl ?? "https://cp.iteco.cloud"
    );
    this.value.setData("insecure", this.value.decodedData.insecure ?? false);
    this.value.setData("vdcId", this.value.decodedData.vdcId ?? "");

    this.validate();
  },

  methods: {
    validate() {
      if (
        !this.value.decodedData.baseUrl ||
        this.value.decodedData.baseUrl === ""
      ) {
        this.$emit("validationChanged", false);
        return;
      }

      if (
        !this.value.decodedData.token ||
        this.value.decodedData.token === ""
      ) {
        this.$emit("validationChanged", false);
        return;
      }

      if (
        !this.value.decodedData.vdcId ||
        this.value.decodedData.vdcId === ""
      ) {
        this.$emit("validationChanged", false);
        return;
      }

      this.$emit("validationChanged", true);
    },

    clear() {
      this.step = 1;
      this.vdcs = null;
      this.errorAllowHost = false;
      this.error = "";
      this.$emit("validationChanged", false);
    },

    hostInAllowList() {
      if (!this.driver?.whitelistDomains) {
        return false;
      }

      const u = parseUrl(this.value.decodedData.baseUrl);

      if (!u.host) {
        return true;
      }

      return (this.driver?.whitelistDomains || []).includes(u.host);
    },

    async addHostToAllowList() {
      this.allowBusy = true;
      const u = parseUrl(this.value.decodedData.baseUrl);

      this.driver.whitelistDomains = this.driver.whitelistDomains || [];

      if (!this.hostInAllowList()) {
        this.driver.whitelistDomains.push(u.host);
      }

      try {
        await this.driver.save();

        this.$refs.connect.$el.click();
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("Could not update driver allow list", e);
        this.allowBusy = false;
      }
    },

    async connect(cb) {
      this.error = "";
      this.errorAllowHost = false;

      let okay = false;

      const u = parseUrl(this.value.decodedData.baseUrl);
      const host =
        u?.host || this.value.decodedData.baseUrl.replace(/^https?:\/\//, "");

      if (!host) {
        return cb(okay);
      }

      if (this.value.decodedData.insecure) {
        this.error = this.t("cluster.credential.makecloud.errors.fetchVdcs");
        return cb(okay);
      }

      this.step = 2;
      this.busy = true;

      const headers = {
        Accept: "application/json",
        "X-API-Auth-Header": `Bearer ${this.value.decodedData.token}`,
      };

      try {
        const out = [];

        let page = 1;
        for (;;) {
          const res = await this.$store.dispatch(
            "management/request",
            {
              method: "GET",
              url: `/meta/proxy/${host}/v1/vdc?page=${page}`,
              headers,
              redirectUnauthorized: false,
            },
            { root: true }
          );

          const items = res?.items || [];
          out.push(...items);

          const total = res?.total ?? out.length;
          if (out.length >= total || items.length === 0) {
            break;
          }

          page += 1;
        }

        this.vdcs = out;
        okay = true;
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("Failed to fetch VDCs", e);

        okay = false;
        this.step = 1;
        this.vdcs = null;

        if (e?._status === 502 && !this.hostInAllowList()) {
          this.errorAllowHost = true;
        } else if (e?._status === 401) {
          this.error = this.t(
            "cluster.credential.makecloud.errors.unauthorized"
          );
        } else {
          this.error =
            e?.message ||
            this.t("cluster.credential.makecloud.errors.fetchVdcs");
        }
      } finally {
        this.busy = false;
      }

      this.vdcId = this.vdcOptions[0]?.value || "";
      if (this.vdcId) {
        this.value.setData("vdcId", this.vdcId);
      }

      this.validate();
      cb(okay);
    },
  },
};
</script>

<template>
  <div>
    <div class="mb-20">
      <LabeledInput
        type="password"
        :mode="mode"
        :disabled="step !== 1"
        :value="value.decodedData.token"
        @change="
          (e) => {
            value.setData('token', e.target.value);
            validate();
          }
        "
        label-key="cluster.credential.makecloud.token.label"
        placeholder-key="cluster.credential.makecloud.token.placeholder"
        required
      />
    </div>

    <div class="mb-20">
      <LabeledInput
        type="text"
        :mode="mode"
        :disabled="step !== 1"
        :value="value.decodedData.baseUrl"
        @change="
          (e) => {
            value.setData('baseUrl', e.target.value);
            validate();
          }
        "
        label-key="cluster.credential.makecloud.baseUrl.label"
        placeholder-key="cluster.credential.makecloud.baseUrl.placeholder"
      />
    </div>

    <div class="mb-20">
      <Checkbox
        :mode="mode"
        :disabled="step !== 1"
        :value="value.decodedData.insecure"
        @click="
          (e) => {
            value.setData('insecure', !value.decodedData.insecure);
            validate();
          }
        "
        label-key="cluster.credential.makecloud.insecureTLS.label"
      />
    </div>

    <BusyButton
      ref="connect"
      label-key="cluster.credential.makecloud.actions.authenticate"
      :disabled="step !== 1 || !canAuthenticate"
      class="mt-20"
      @click="connect"
    />

    <button
      class="btn role-primary mt-20 ml-20"
      :disabled="busy || step === 1"
      @click="clear"
    >
      {{ t("cluster.credential.makecloud.actions.edit") }}
    </button>

    <Banner v-if="error" class="mt-20" color="error">
      {{ error }}
    </Banner>

    <Banner v-if="errorAllowHost" color="error" class="allow-list-error mt-20">
      <div>
        {{ t("cluster.credential.makecloud.errors.notAllowed", { hostname }) }}
      </div>
      <button
        :disabled="allowBusy"
        class="btn ml-10 role-primary"
        @click="addHostToAllowList"
      >
        {{ t("cluster.credential.makecloud.actions.addToAllowList") }}
      </button>
    </Banner>

    <div v-if="vdcs" class="row mt-20">
      <div class="col span-12">
        <LabeledSelect
          v-model:value="vdcId"
          label-key="cluster.credential.makecloud.vdcId.label"
          :options="vdcOptions"
          :searchable="true"
          @update:value="
            value.setData('vdcId', $event);
            validate();
          "
          required
        />
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.allow-list-error {
  display: flex;

  > :first-child {
    flex: 1;
  }
}
</style>
