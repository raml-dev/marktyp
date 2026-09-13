<script lang="ts">
  import { onMount } from 'svelte';
  export let kind: 'link' | 'image';
  export let value = '';
  export let onapply: () => void;
  export let oncancel: () => void;
  let dialog: HTMLDialogElement;
  let input: HTMLInputElement;
  $: title = kind === 'image' ? 'Insert Image' : 'Insert Link';
  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    input.focus();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });
  function keydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault();
      onapply();
    }
  }
</script>

<dialog
  bind:this={dialog}
  class="rich-editor-modal"
  aria-labelledby="resource-dialog-title"
  on:cancel|preventDefault={oncancel}
>
  <div class="rich-editor-modal__header"><span id="resource-dialog-title">{title}</span></div>
  <input
    bind:this={input}
    bind:value
    class="resource-dialog__input"
    aria-label={kind === 'image' ? 'Image URL or file path' : 'Link URL'}
    placeholder={kind === 'image' ? 'Image URL or file path' : 'https://example.com'}
    on:keydown={keydown}
  />
  <div class="rich-editor-modal__actions">
    <button class="ghost-button ghost-button--small" on:click={oncancel} type="button">Cancel</button>
    <button class="ghost-button ghost-button--small" disabled={!value.trim()} on:click={onapply} type="button"
      >Insert</button
    >
  </div>
</dialog>
