<script lang="ts">
  import { onMount } from 'svelte';
  import MarktypLogo from './MarktypLogo.svelte';
  import MarktypWordmark from './MarktypWordmark.svelte';
  import type { AppInfo } from '../../types';

  export let info: AppInfo;
  export let onclose: () => void;

  let dialog: HTMLDialogElement;

  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });
</script>

<dialog
  bind:this={dialog}
  class="about-dialog"
  aria-labelledby="about-dialog-title"
  on:cancel|preventDefault={onclose}
>
  <div class="about-dialog__header">
    <h2 id="about-dialog-title">About</h2>
    <button class="toolbar-button" type="button" on:click={onclose}>Close</button>
  </div>
  <div class="about-dialog__body">
    <div class="about-dialog__brand">
      <MarktypLogo size={52} className="about-dialog__logo" />
      <h1><MarktypWordmark size={24} className="about-dialog__wordmark" /></h1>
    </div>
    <p class="about-dialog__version">Version {info.version}</p>
  </div>
</dialog>
