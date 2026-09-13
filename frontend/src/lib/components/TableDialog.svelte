<script lang="ts">
  import { onMount } from 'svelte';
  export let rows = 3;
  export let columns = 3;
  export let onapply: () => void;
  export let oncancel: () => void;
  let dialog: HTMLDialogElement;
  let rowsInput: HTMLInputElement;
  onMount(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.showModal();
    rowsInput.focus();
    rowsInput.select();
    return () => {
      dialog.close();
      previous?.focus();
    };
  });
  function normalize() {
    rows = Math.max(2, Math.min(20, Math.round(rows || 2)));
    columns = Math.max(1, Math.min(10, Math.round(columns || 1)));
  }
  function apply() {
    normalize();
    onapply();
  }
  function keydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault();
      apply();
    }
  }
</script>

<dialog
  bind:this={dialog}
  class="rich-editor-modal table-dialog"
  aria-labelledby="table-dialog-title"
  on:cancel|preventDefault={oncancel}
>
  <div class="rich-editor-modal__header"><span id="table-dialog-title">Insert Table</span></div>
  <div class="table-dialog__dimensions">
    <label
      >Rows<input
        bind:this={rowsInput}
        bind:value={rows}
        min="2"
        max="20"
        type="number"
        on:keydown={keydown}
      /></label
    >
    <label>Columns<input bind:value={columns} min="1" max="10" type="number" on:keydown={keydown} /></label>
  </div>
  <p class="panel__hint">The first row is used for column headings.</p>
  <div class="rich-editor-modal__actions">
    <button class="ghost-button ghost-button--small" on:click={oncancel} type="button">Cancel</button>
    <button class="ghost-button ghost-button--small" on:click={apply} type="button">Insert</button>
  </div>
</dialog>
