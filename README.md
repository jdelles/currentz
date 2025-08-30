# Currentz

**Currentz** is a modern personal finance app inspired by classics like Microsoft Money — rebuilt with today's tools.  
Its primary focus is **cash flow**: helping you see not just where your money went, but where it’s going.

---

## ✨ Features (MVP)

- ✅ Minimal Go API with a `/health` endpoint  
- 📂 Clean, idiomatic project structure (`cmd/api`, `internal/…`)  
- 🐳 Ready for Postgres + migrations (coming soon)  
- 🔮 Future: transaction import (CSV/OFX), recurring events, cash flow projections  

---

## 🚀 Quickstart

Clone the repo:

```bash
git clone git@github.com:jdelles/currentz.git
cd currentz
```

Run the API server:

```bash
go run ./cmd/api
```

Test it:

```bash
curl http://localhost:8080/health
# -> ok
```

## Tech Stack

Go  
Chi  
caarlos0/env  
Planned: Postgres, Goose, sqlc, react  

## Project Layout

```bash
currentz/
  cmd/
    api/
  internal/
    http/handlers/
    core/
    db/
  go.mod
  README.md
```
