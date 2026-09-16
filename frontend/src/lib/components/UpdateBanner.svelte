<script lang="ts">
  import { onDestroy } from 'svelte';
  import { updateState, updateStore } from '../stores/updateStore';
  import { markdownToHtml } from '../utils/markdown';

  let releaseNotesDialog: HTMLDialogElement;
  let releaseNotesHtml = '';

  function version(): string {
    return ($updateState.selectedRelease?.tag_name || $updateState.selectedRelease?.name || '').trim();
  }

  function openReleaseNotes() {
    const markdown = ($updateState.selectedRelease?.body || '').trim();
    releaseNotesHtml = markdown ? markdownToHtml(markdown) : '<p>No release notes available.</p>';
    releaseNotesDialog.showModal();
  }

  function closeReleaseNotes() {
    releaseNotesDialog.close();
  }

  function handleReleaseNotesClick(event: MouseEvent) {
    const link = (event.target as Element | null)?.closest('a');
    if (!(link instanceof HTMLAnchorElement)) return;
    event.preventDefault();
    updateStore.openURL(link.href);
  }

  function openExternalReleaseLinks(node: HTMLElement) {
    node.addEventListener('click', handleReleaseNotesClick);
    return {
      destroy() {
        node.removeEventListener('click', handleReleaseNotesClick);
      },
    };
  }

  onDestroy(() => {
    if (releaseNotesDialog?.open) releaseNotesDialog.close();
  });
</script>

{#if $updateState.visible && $updateState.selectedRelease}
  <section class="update-banner" aria-label="Update available">
    <div class="update-banner__message">
      <strong>Update available</strong>
      <span>
        Marktyp version {version()} is available.
        <button class="update-banner__link" type="button" on:click={openReleaseNotes}
          >Show release notes</button
        >
      </span>
    </div>
    <div class="update-banner__actions">
      <button
        class="ghost-button"
        type="button"
        disabled={$updateState.loading}
        on:click={() => updateStore.ignoreCurrentRelease()}>Ignore</button
      >
      <button class="ghost-button" type="button" on:click={() => updateStore.dismiss()}>Dismiss</button>
      <button
        class="ghost-button update-banner__primary"
        type="button"
        disabled={$updateState.loading}
        on:click={() => updateStore.openReleasePage()}>Release page</button
      >
    </div>
  </section>
{/if}

<dialog
  bind:this={releaseNotesDialog}
  class="release-notes-dialog"
  aria-labelledby="release-notes-title"
  on:cancel|preventDefault={closeReleaseNotes}
>
  <div class="release-notes-dialog__header">
    <h2 id="release-notes-title">Release notes</h2>
    <button class="toolbar-button" type="button" on:click={closeReleaseNotes}>Close</button>
  </div>
  <article class="release-notes-dialog__content" use:openExternalReleaseLinks>
    <!-- eslint-disable-next-line svelte/no-at-html-tags -- release markdown is sanitized by markdownToHtml -->
    {@html releaseNotesHtml}
  </article>
</dialog>
