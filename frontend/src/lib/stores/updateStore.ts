import { get, writable } from 'svelte/store';
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import type { AppInfo, GitHubRelease, UpdateResponse } from '../../types';
import * as backend from '../backend';
import { editorState } from './workspaceStore';

type UpdateState = {
  initialized: boolean;
  visible: boolean;
  loading: boolean;
  ignoredVersion: string;
  currentVersion: string;
  appInfo: AppInfo | null;
  updateInfo: UpdateResponse | null;
  selectedRelease: GitHubRelease | null;
  error: string;
};

const initialState: UpdateState = {
  initialized: false,
  visible: false,
  loading: false,
  ignoredVersion: '',
  currentVersion: '',
  appInfo: null,
  updateInfo: null,
  selectedRelease: null,
  error: '',
};

export const updateState = writable<UpdateState>({ ...initialState });

function releaseVersion(release: GitHubRelease | null): string {
  return (release?.tag_name || release?.name || '').trim();
}

function applyAvailableUpdate(updateInfo: UpdateResponse | null) {
  const release = updateInfo?.Release ?? null;
  const version = releaseVersion(release);
  updateState.update((state) => ({
    ...state,
    updateInfo,
    selectedRelease: release,
    visible: Boolean(release && version && state.ignoredVersion !== version),
  }));
}

export const updateStore = {
  async init() {
    const current = get(updateState);
    if (current.initialized) return;

    const workspace = get(editorState).workspace;
    updateState.update((state) => ({
      ...state,
      initialized: true,
      appInfo: workspace.appInfo,
      currentVersion: workspace.appInfo.productVersion,
    }));

    if (workspace.config.checkForUpdates) await this.refresh();
  },

  async refresh() {
    updateState.update((state) => ({ ...state, loading: true, error: '' }));
    try {
      applyAvailableUpdate(await backend.getUpdatesFromRepo());
    } catch (error) {
      updateState.update((state) => ({ ...state, error: String(error), visible: false }));
      console.warn('Failed to check for updates', error);
    } finally {
      updateState.update((state) => ({ ...state, loading: false }));
    }
  },

  ignoreCurrentRelease() {
    updateState.update((state) => ({
      ...state,
      ignoredVersion: releaseVersion(state.selectedRelease),
      visible: false,
    }));
  },

  dismiss() {
    updateState.update((state) => ({ ...state, visible: false }));
  },

  openReleasePage() {
    const state = get(updateState);
    const version = releaseVersion(state.selectedRelease);
    const repositoryURL = state.appInfo?.ghLink.replace(/\/+$/, '') ?? '';
    const releaseURL =
      state.selectedRelease?.html_url ||
      (repositoryURL && version ? `${repositoryURL}/releases/tag/${version}` : '');
    if (releaseURL) BrowserOpenURL(releaseURL);
  },

  openURL(url: string) {
    if (/^https:\/\//i.test(url)) BrowserOpenURL(url);
  },
};
