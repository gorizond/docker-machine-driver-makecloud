<script>
export default {
  props: {
    label: {
      type:    String,
      default: undefined,
    },
    labelKey: {
      type:    String,
      default: undefined,
    },
    disabled: {
      type:    Boolean,
      default: false,
    },
    tabIndex: {
      type:    Number,
      default: null,
    },
    componentTestid: {
      type:    String,
      default: 'busy-button'
    },
  },

  data() {
    return { busy: false };
  },

  computed: {
    displayLabel() {
      if (this.labelKey) {
        const t = this.$store.getters['i18n/t'];

        return t(this.labelKey);
      }

      return this.label;
    }
  },

  methods: {
    clicked($event) {
      if ($event) {
        $event.stopPropagation();
        $event.preventDefault();
      }

      if (this.disabled) {
        return;
      }

      const cb = () => {
        this.busy = false;
      };

      this.busy = true;
      this.$emit('click', cb);
    },
  }
};
</script>

<template>
  <button
    ref="btn"
    class="btn role-primary"
    :disabled="disabled"
    :tab-index="tabIndex"
    :data-testid="componentTestid + '-async-button'"
    @click="clicked"
  >
    <i
      v-if="busy"
      class="icon icon-lg icon-spinner icon-spin"
    />
    <span
      v-else
      class="icon-spacer"
    />
    <span>{{ displayLabel }}</span>
    <span class="icon-spacer" />
  </button>
</template>

<style lang="scss" scoped>
  .icon-spacer {
    width: 24px;
  }
</style>
