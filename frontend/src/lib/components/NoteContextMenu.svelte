<script lang="ts">
  import { onMount } from 'svelte';
  export let x: number;
  export let y: number;
  export let onclose: () => void;
  export let onreveal: () => void;
  export let onhtml: () => void;
  export let onpdf: () => void;
  export let ondelete: () => void;
  let menu: HTMLDivElement;
  onMount(() => {
    const rect = menu.getBoundingClientRect();
    x = Math.max(8, Math.min(x, window.innerWidth - rect.width - 8));
    y = Math.max(8, Math.min(y, window.innerHeight - rect.height - 8));
    menu.querySelector('button')?.focus();
  });
  function keydown(event: KeyboardEvent) {
    const buttons = Array.from(menu.querySelectorAll('button'));
    const current = buttons.indexOf(document.activeElement as HTMLButtonElement);
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      buttons[(current + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length]?.focus();
    } else if (event.key === 'Escape' || event.key === 'Tab') onclose();
  }
  function run(action: () => unknown) {
    const selectedAction = action;
    selectedAction();
    onclose();
  }
</script>

<div
  bind:this={menu}
  class="note-context-menu"
  role="menu"
  aria-label="Note actions"
  tabindex="-1"
  on:keydown={keydown}
  style={`left: ${x}px; top: ${y}px;`}
>
  <button class="note-context-menu__item" role="menuitem" on:click={() => run(onreveal)} type="button"
    >Reveal In Files</button
  >
  <button class="note-context-menu__item" role="menuitem" on:click={() => run(onhtml)} type="button"
    >Export HTML</button
  >
  <button class="note-context-menu__item" role="menuitem" on:click={() => run(onpdf)} type="button"
    >Export PDF</button
  >
  <button
    class="note-context-menu__item is-danger"
    role="menuitem"
    on:click={() => run(ondelete)}
    type="button">Delete</button
  >
</div>
