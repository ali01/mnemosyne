# Data Directory

This directory contains all data files used by the Mnemosyne backend:

## Contents

- `sample_graph.json` - Sample graph data for testing and development
- `testdata/` - Test fixtures for unit and integration tests
- `*-clone/` - Git clones of Obsidian vaults (automatically created, not tracked in git)

## Vault Clones

When the application runs, it will clone the configured Obsidian vault repository into this directory. The default clone directory is `vault-clone/`, but this can be configured in `config.yaml`.

These clone directories are listed in `.gitignore` and should not be committed to the repository.

## Important Notes

- All data storage should happen within this directory
- Vault clones are read-only and managed by the Git integration module
- The application will automatically create clone directories as needed
