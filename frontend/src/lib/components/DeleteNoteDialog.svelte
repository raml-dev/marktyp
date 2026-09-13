<script lang="ts">
  import { onMount } from 'svelte';

  export let path: string;
  export let losesChanges = false;
  export let onconfirm: () => void;
  export let oncancel: () => void;

  let dialog: HTMLDialogElement;
  let cancelButton: HTMLButtonElement;

  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    cancelButton.focus();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });
</script>

<dialog
  bind:this={dialog}
  class="rich-editor-modal delete-note-dialog"
  aria-labelledby="delete-note-dialog-title"
  on:cancel|preventDefault={oncancel}
>
  <div class="rich-editor-modal__header">
    <span id="delete-note-dialog-title">Delete note?</span>
  </div>
  <p class="delete-note-dialog__path">{path}</p>
  {#if losesChanges}
    <p class="delete-note-dialog__warning">Unsaved changes will also be lost.</p>
  {/if}
  <p class="panel__hint">This action cannot be undone.</p>
  <div class="rich-editor-modal__actions">
    <button
      bind:this={cancelButton}
      class="ghost-button ghost-button--small"
      on:click={oncancel}
      type="button">Cancel</button
    >
    <button class="ghost-button ghost-button--small ghost-button--danger" on:click={onconfirm} type="button"
      >Delete</button
    >
  </div>
</dialog>
