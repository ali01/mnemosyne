export function openInObsidian(vaultName: string, filePath: string): void {
  const pathWithoutExt = filePath.replace(/\.md$/, '');
  const uri = `obsidian://open?vault=${encodeURIComponent(vaultName)}&file=${encodeURIComponent(pathWithoutExt)}`;
  const a = document.createElement('a');
  a.href = uri;
  a.click();
}
