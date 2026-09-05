# `pagination`

Import path: `github.com/mhmdkzr/loop/pkg/pagination`

Pagination metadata and validation utilities for list endpoints.

## Types

- `Meta` — struct with `Page`, `Size`, and `Total` fields, with normalization (`Normalize`) and validation (`Validate`) functions.
- Constants: `DefaultPage=1`, `DefaultSize=10`, `MaxPageSize=1000`.
- `ParseRequest` reads `page` and `size` query parameters from an HTTP request.
