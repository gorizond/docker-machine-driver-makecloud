<script>
import { Checkbox } from '@components/Form/Checkbox';
import { LabeledInput } from '@components/Form/LabeledInput';

export default {
  components: {
    Checkbox,
    LabeledInput,
  },

  props: {
    mode: {
      type:     String,
      required: true,
    },
    value: {
      type:     Object,
      required: true,
    },
  },

  created() {
    this.value.setData('token', this.value.decodedData.token ?? '');
    this.value.setData('baseUrl', this.value.decodedData.baseUrl ?? 'https://cp.iteco.cloud');
    this.value.setData('insecure', this.value.decodedData.insecure ?? false);

    this.validate();
  },

  methods: {
    validate() {
      if (this.value.decodedData.token === '') {
        this.$emit('validationChanged', false);
        return;
      }

      this.$emit('validationChanged', true);
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
        :value="value.decodedData.token"
        @change="e => { value.setData('token', e.target.value); validate(); }"
        label-key="cluster.credential.makecloud.token.label"
        placeholder-key="cluster.credential.makecloud.token.placeholder"
        required
      />
    </div>

    <div class="mb-20">
      <LabeledInput
        type="text"
        :mode="mode"
        :value="value.decodedData.baseUrl"
        @change="e => { value.setData('baseUrl', e.target.value); validate(); }"
        label-key="cluster.credential.makecloud.baseUrl.label"
        placeholder-key="cluster.credential.makecloud.baseUrl.placeholder"
      />
    </div>

    <div>
      <Checkbox
        :mode="mode"
        :value="value.decodedData.insecure"
        @click="e => { value.setData('insecure', !value.decodedData.insecure); validate(); }"
        label-key="cluster.credential.makecloud.insecureTLS.label"
      />
    </div>
  </div>
</template>
