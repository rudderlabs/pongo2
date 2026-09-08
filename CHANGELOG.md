# Changelog

## [6.3.0](https://github.com/rudderlabs/pongo2/compare/v6.2.0...v6.3.0)

- Make `TemplateSet.firstTemplateCreated` atomic to avoid races when templates are parsed concurrently.
- Add a bounded exec-body parse cache with `SetExecTemplateCache` and `ExecTemplateCacheStats`.

## [6.1.0](https://github.com/rudderlabs/pongo2/compare/v6.0.18...v6.1.0) (2024-10-08)


### Features

* rename module to github.com/rudderlabs/pongo2/v6 ([43492c8](https://github.com/rudderlabs/pongo2/commit/43492c8a4cbc16c9e4534fd9c4e05e63eb2f2451))


### Miscellaneous

* add gha checks and fix linting and formatting issues ([#24](https://github.com/rudderlabs/pongo2/issues/24)) ([3978634](https://github.com/rudderlabs/pongo2/commit/3978634a3785d83176b65d8485be98e0d84d213c))
