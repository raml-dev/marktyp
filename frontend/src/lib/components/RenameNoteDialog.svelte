<script lang="ts">
  import { onMount } from 'svelte';

  export let title: string;
  export let onconfirm: (title: string) => void;
  export let oncancel: () => void;

  let dialog: HTMLDialogElement;
  let input: HTMLInputElement;

  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    input.focus();
    input.select();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });

  function submit() {
    const nextTitle = title.trim();
    if (nextTitle) onconfirm(nextTitle);
  }

  function keydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault();
      submit();
    }
  }
</script>

<dialog
  bind:this={dialog}
  class="rich-editor-modal"
  aria-labelledby="rename-note-title"
  on:cancel|preventDefault={oncancel}
>
  <div class="rich-editor-modal__header"><span id="rename-note-title">Rename note</span></div>
  <input
    bind:this={input}
    bind:value={title}
    class="resource-dialog__input"
    aria-label="Note title"
    placeholder="Note title"
    on:keydown={keydown}
  />
  <div class="rich-editor-modal__actions">
    <button class="ghost-button ghost-button--small" on:click={oncancel} type="button">Cancel</button>
    <button class="ghost-button ghost-button--small" disabled={!title.trim()} on:click={submit} type="button"
      >Rename</button
    >
  </div>
</dialog>
