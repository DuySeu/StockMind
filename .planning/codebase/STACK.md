# STACK

## Overview
StockMind is a two-tier full-stack application leveraging Go for the backend and React (via Vite) for the frontend.

## Backend
- **Language**: Go 1.25.1
- **Router**: chi (`github.com/go-chi/chi/v5`)
- **Database Driver**: pgx (`github.com/jackc/pgx/v5`)
- **Code Generator**: sqlc for type-safe database access
- **Migrations**: goose (`github.com/pressly/goose/v3`)
- **Websockets**: coder/websocket (`github.com/coder/websocket`)

## Frontend
- **Framework**: React 19.1.1
- **Build Tool**: Vite 7.1.2
- **Language**: TypeScript 5.8
- **Styling**: TailwindCSS 4.1, `clsx`, `tailwind-merge`
- **UI Components**: Radix UI Primitives (`@radix-ui/react-*`), Lucide React icons
- **Form Handling**: React Hook Form, zod
- **Routing**: React Router DOM 7.8

## Infrastructure & Tooling
- **Database**: PostgreSQL 17 (via Docker alpine)
- **Containerization**: Docker & Docker Compose
- **Build Automation**: Makefile
