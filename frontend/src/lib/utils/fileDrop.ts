export function isFileDrop(event: DragEvent): boolean {
  return Array.from(event.dataTransfer?.types ?? []).includes('Files');
}

// Cancel navigation and native text insertion, while allowing Wails to resolve paths.
export function preventFileDropNavigation(event: DragEvent) {
  if (isFileDrop(event)) event.preventDefault();
}
