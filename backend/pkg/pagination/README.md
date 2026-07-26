# `pkg/pagination`

Pagination metadata and validation utilities for list endpoints.

## Types

- `Meta` — struct with `Page`, `Size`, and `Total` fields, with normalization (`Normalize`) and validation (`Validate`) functions.
- Constants: `DefaultPage=1`, `DefaultSize=10`, `MaxPageSize=1000`.
- Used by list-style query slices (asset list, balance list, tx list, wallet list, etc.).
