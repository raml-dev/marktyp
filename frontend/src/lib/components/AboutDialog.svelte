<script lang="ts">
  import { onMount } from 'svelte';
  import MarktypWordmark from './MarktypWordmark.svelte';
  import { workspaceStore } from '../stores/workspaceStore';
  import type { AppInfo } from '../../types';

  export let info: AppInfo;
  export let onclose: () => void;

  let dialog: HTMLDialogElement;

  $: repositoryURL = info.ghLink.replace(/\/+$/, '');
  $: releaseURL =
    repositoryURL && info.productVersion ? `${repositoryURL}/releases/tag/${info.productVersion}` : '';

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
      <h1><MarktypWordmark size={24} className="about-dialog__wordmark" /></h1>
      <p class="about-dialog__version">
        version
        {#if releaseURL}
          <button type="button" title={releaseURL} on:click={() => workspaceStore.openLink(releaseURL)}
            >{info.productVersion || '-'}</button
          >
        {:else}
          {info.productVersion || '-'}
        {/if}
      </p>
    </div>
    <p>Licensed under the {info.license || '-'}</p>
    <div class="about-dialog__actions">
      {#if repositoryURL}
        <button class="ghost-button" type="button" on:click={() => workspaceStore.openLink(repositoryURL)}
          >Source code</button
        >
      {/if}
      {#if info.docsLink}
        <button class="ghost-button" type="button" on:click={() => workspaceStore.openLink(info.docsLink)}
          >Docs</button
        >
      {/if}
    </div>
    <p class="about-dialog__credit">
      Built by
      {#if info.orgLink}
        <button type="button" title={info.orgLink} on:click={() => workspaceStore.openLink(info.orgLink)}
          >{info.companyName || 'raml-dev'}</button
        >
      {:else}
        {info.companyName || 'raml-dev'}
      {/if}
    </p>
  </div>
</dialog>
