import { importTypes } from '@rancher/auto-import';
import { IPlugin } from '@shell/core/types';

export default function(plugin: IPlugin): void {
  importTypes(plugin);

  plugin.metadata = require('./package.json');
}
