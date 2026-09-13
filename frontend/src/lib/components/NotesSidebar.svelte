<script lang="ts">
  import MarktypWordmark from './MarktypWordmark.svelte';
  import type { NoteSummary } from '../../types';
  export let collapsed: boolean;
  export let query: string;
  export let notes: NoteSummary[];
  export let activePath: string;
  export let ontoggle: () => void;
  export let oncreate: () => void;
  export let ondelete: () => void;
  export let onselect: (path: string) => void;
  export let oncontextmenu: (event: MouseEvent, note: NoteSummary) => void;
</script>

<aside class={collapsed ? 'sidebar is-collapsed' : 'sidebar'} aria-label="Notes">
  {#if collapsed}
    <div class="sidebar-collapsed">
      <button
        class="sidebar-collapsed__toggle"
        on:click={ontoggle}
        type="button"
        title="Show notes"
        aria-expanded="false">»</button
      >
    </div>
  {:else}
    <div class="sidebar__collapse-row">
      <button
        class="sidebar-collapsed__toggle"
        on:click={ontoggle}
        type="button"
        title="Hide notes"
        aria-expanded="true">«</button
      >
    </div>
    <div class="sidebar__brand">
      <h1><MarktypWordmark size={22} className="brand-wordmark" /></h1>
    </div>
    <div class="sidebar__section">
      <div class="sidebar__section-header">
        <span>Recent notes</span>
        <div class="sidebar__section-actions">
          <button class="ghost-button" on:click={oncreate} type="button">New</button>
          <button
            class="ghost-button ghost-button--danger"
            disabled={!activePath}
            on:click={ondelete}
            type="button">Delete</button
          >
        </div>
      </div>
      <input
        class="note-search"
        bind:value={query}
        aria-label="Search notes"
        placeholder="Search notes"
        type="search"
      />
      <div class="note-list">
        {#each notes as note (note.path)}
          <button
            class={note.path === activePath ? 'note-card is-active' : 'note-card'}
            aria-current={note.path === activePath ? 'page' : undefined}
            on:click={() => onselect(note.path)}
            on:contextmenu={(event) => oncontextmenu(event, note)}
            type="button"
            title={note.path}
          >
            <span class="note-card__title">{note.title}</span><span class="note-card__meta"
              >{note.updatedLabel}</span
            >
          </button>
        {:else}
          <p class="panel__hint">{query ? 'No matching notes' : 'No recent notes'}</p>
        {/each}
      </div>
    </div>
  {/if}
</aside>
