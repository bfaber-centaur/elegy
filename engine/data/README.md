# Component table

`components.json` is a verbatim copy of stars-elegy `data/components.json`
(schema "stars-elegy components v1"), the public measured component table
described by stars-elegy `docs/COMPONENTS.md`.

- Source: stars-elegy commit `10d5397d308953c63f82928f4064fb700ee9f1a8`
  (PR #30), unchanged on `main` at `96fd2e7`.
- SHA-256: `0e0a5fa04cb87b3baf1d0cf3c26267cce8ec5d2f49addde99a8d609349f05a70`.

To update it, copy the file again from stars-elegy, record the new commit
and hash here, and run `go test ./...` (`TestCatalogFileHash` checks the
hash).
