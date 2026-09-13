<script lang="ts">
  export let openMenu: string | null = null;
  export let onmousedown: (event: MouseEvent) => void;
  export let actions: Record<string, () => unknown>;
  const groups = [
    { name: 'text', label: 'Text', items: ['Bold', 'Italic', 'Link'] },
    { name: 'blocks', label: 'Blocks', items: ['Heading 2', 'Bullet List', 'Task List', 'Blockquote'] },
    { name: 'code', label: 'Code', items: ['Inline Code', 'Code Block'] },
    { name: 'insert', label: 'Insert', items: ['Image', 'Table', 'Mermaid', 'Inline Math', 'Math Block'] },
  ];
</script>

<div
  aria-label="Formatting toolbar"
  class="toolbar-cluster toolbar-cluster--editing"
  on:mousedown={onmousedown}
  role="toolbar"
  tabindex="-1"
>
  {#each groups as group (group.name)}
    <details
      class="toolbar-menu"
      open={openMenu === group.name}
      on:toggle={(event) => {
        if (event.currentTarget.open) openMenu = group.name;
        else if (openMenu === group.name) openMenu = null;
      }}
    >
      <summary class="toolbar-menu__trigger"
        >{group.label}<span class="toolbar-menu__chevron">▾</span></summary
      >
      <div class="toolbar-menu__content">
        {#each group.items as label (label)}
          <button class="toolbar-menu__item" on:click={() => actions[label]?.()} type="button">{label}</button
          >
        {/each}
      </div>
    </details>
  {/each}
</div>
