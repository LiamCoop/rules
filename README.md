# Rules

A multi-tenant rules engine for defining tenant-specific schemas and evaluating business logic with [CEL](https://cel.dev/).

The project is split into two parts:

- **`server/`**: Go + PostgreSQL API for tenants, schemas, rules, and evaluation
- **`client/`**: Next.js frontend shell for interacting with the engine

## What it does

Rules lets you:

- isolate data and rules per tenant
- define schemas dynamically without redeploying
- store and update rules as CEL expressions
- evaluate facts against those rules in real time
- version schemas for safer updates

## Highlights

- **Multi-tenant by design** with isolated tenant data
- **Stateless evaluation** backed by PostgreSQL
- **Dynamic schemas** and runtime validation
- **CEL-based rules** for safe, expressive conditions
- **Production-oriented API docs** in `server/docs/`
- **Load-tested and performance-focused** implementation
- **Project notes** call out sustained throughput of ~7k RPS per instance under load

## Architecture at a glance

- **Backend:** Go 1.24, Chi router, PostgreSQL
- **Rule engine:** Google CEL
- **Frontend:** Next.js 16 + React 19
- **Docs:** API and implementation notes live under `server/docs/`

## Project docs

- [`client/README.md`](./client/README.md)
- [`server/docs/README.md`](./server/docs/README.md)
- [`server/docs/API.md`](./server/docs/API.md)

## Quick start

### Server

See `server/docs/README.md` for the API, local setup, and testing workflow.

### Client

```bash
cd client
npm install
npm run dev
```

## Repo structure

```text
client/   Next.js UI
server/   Go API, migrations, rules engine, docs
```

## Notes

This repo is more of a working system than a framework starter: the focus is on rules evaluation, tenant isolation, and API-driven workflows.
