<script>
import { LabeledInput } from "@components/Form/LabeledInput";
import Checkbox from "@components/Form/Checkbox/Checkbox";

export default {
    components: {
        Checkbox,
        LabeledInput,
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

    created() {
        this.value.vdcId = this.value.vdcId ?? "";
        this.value.templateId = this.value.templateId ?? "";
        this.value.templateName = this.value.templateName ?? "";
        this.value.storageProfileId = this.value.storageProfileId ?? "";
        this.value.networkId = this.value.networkId ?? "";
        this.value.floatingIp = this.value.floatingIp ?? "";

        if (!this.credentialId) {
            this.value.token = this.value.token ?? "";
            this.value.baseUrl = this.value.baseUrl ?? "https://cp.iteco.cloud";
            this.value.insecure = this.value.insecure ?? false;
        }

        this.value.cpu = (this.value.cpu ?? "").toString() || "2";
        this.value.ram = (this.value.ram ?? "").toString() || "4";
        this.value.diskSize = (this.value.diskSize ?? "").toString() || "40";

        this.value.firewallTemplateId = this.value.firewallTemplateId ?? [];
        this.value.tags = this.value.tags ?? [];
        this.value.metadata = this.value.metadata ?? [];
        this.value.userData = this.value.userData ?? "";
        this.value.noInjectSshKey = this.value.noInjectSshKey ?? false;
        this.value.waitTimeout =
            (this.value.waitTimeout ?? "").toString() || "600";

        this.value.sshUser = this.value.sshUser ?? "root";
        this.value.sshPort = (this.value.sshPort ?? "").toString() || "22";

        this.validate();
    },

    watch: {
        value: {
            deep: true,
            handler() {
                this.validate();
            },
        },
    },

    computed: {
        disabled() {
            return this.mode === "view";
        },
        hasCredential() {
            return !!this.credentialId;
        },
    },

    methods: {
        validate() {
            const hasTemplate =
                (this.value.templateId &&
                    this.value.templateId.trim() !== "") ||
                (this.value.templateName &&
                    this.value.templateName.trim() !== "");

            if (!this.value.vdcId || this.value.vdcId.trim() === "") {
                this.$emit("validationChanged", false);
                return;
            }

            if (!this.hasCredential) {
                if (!this.value.token || this.value.token.trim() === "") {
                    this.$emit("validationChanged", false);
                    return;
                }
            }

            if (!hasTemplate) {
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

            const diskSize = parseInt(
                (this.value.diskSize || "").toString(),
                10,
            );
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
            <div class="col span-6">
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
            <div class="col span-6">
                <LabeledInput
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
                <LabeledInput
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
                <LabeledInput
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
                <LabeledInput
                    type="text"
                    :mode="mode"
                    :disabled="disabled"
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
                    :value="(value.firewallTemplateId || []).join(',')"
                    @change="
                        (e) => {
                            setStringArray(
                                'firewallTemplateId',
                                e.target.value,
                            );
                        }
                    "
                    label-key="cluster.machineConfig.makecloud.advanced.firewallTemplateIds.label"
                    tooltip-key="cluster.machineConfig.makecloud.advanced.firewallTemplateIds.tooltip"
                />
            </div>

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
