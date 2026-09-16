<script lang="ts">
  import { onMount } from 'svelte';
  export let source = '';
  export let kind: 'math-inline' | 'math-block' | 'mermaid';
  export let onapply: () => void;
  export let oncancel: () => void;
  let dialog: HTMLDialogElement;
  let input: HTMLTextAreaElement;
  function keydown(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault();
      onapply();
    }
  }
  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    input.focus();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });
</script>

<dialog
  bind:this={dialog}
  class="rich-editor-modal"
  aria-labelledby="rich-editor-title"
  on:cancel|preventDefault={oncancel}
>
  <div class="rich-editor-modal__header">
    <span id="rich-editor-title"
      >{kind === 'math-inline' ? 'Inline Math' : kind === 'math-block' ? 'Math Block' : 'Mermaid'}</span
    >
  </div>
  <textarea
    bind:this={input}
    bind:value={source}
    aria-labelledby="rich-editor-title"
    class="rich-editor-modal__input"
    spellcheck={false}
    on:keydown={keydown}
  ></textarea>
  <div class="rich-editor-modal__actions">
    <button class="ghost-button ghost-button--small" on:click={oncancel} type="button">Cancel</button>
    <button class="ghost-button ghost-button--small" on:click={onapply} type="button">Apply</button>
  </div>
</dialog>
