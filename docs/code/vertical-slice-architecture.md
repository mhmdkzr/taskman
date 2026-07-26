# Vertical Slice Architecture

Vertical Slice Architecture (VSA) is an approach to organizing code around features first, rather than technical layers.

## The Problem with Layers

In a layered (n-tier) architecture, code is grouped by technical role: all handlers together, all services together, all repositories together. Adding or changing a feature means touching multiple directories, each containing code from many unrelated features.

```
handlers/
  transfer.go      ← alongside account.go, user.go, report.go...
services/
  transfer.go
repositories/
  transfer.go
```

## Slices

A vertical slice groups everything belonging to a single feature or flow in one place. The package path identifies the feature; files within the package separate technical concerns.

```
user/
  profile/
    update/
      handler.go   ← HTTP binding
      service.go   ← business logic
      repo.go      ← database queries
```

Layers still exist — handler, service, store — but they are local to the feature. When you open a package, everything inside is relevant to one thing. You are not reading code for unrelated features.

## Why It Matters

Each slice is self-contained. Features grow independently. The blast radius of a change is naturally bounded by the package boundary. This keeps complexity local. As the system grows, each new feature adds a new slice — it doesn't spread across existing ones. The cognitive load stays relatively flat: finding code means knowing what feature it belongs to, not knowing the layer and then filtering out the noise. Parallel development is natural. Multiple developers/agents can work on different slices simultaneously with minimal conflict.
