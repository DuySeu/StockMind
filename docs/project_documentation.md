# StockMind Project Documentation

## 1. Overview
StockMind is an AI-powered financial assistant tailored for the Vietnamese stock market. It aims to streamline the investment research process by providing intuitive access to financial data, intelligent analysis, and real-time market insights. The project uses a modern web stack backed by robust data retrieval services.

## 2. System Architecture
StockMind operates on a typical three-tier architecture augmented with AI capabilities:
*   **Frontend (React/Vite)**: Provides the user interface, including the AI chat interface, price boards, and market research dashboards.
*   **Backend (Go/Chi)**: Handles API requests, acts as an SSE (Server-Sent Events) server for the streaming chat, manages user sessions via PostgreSQL, and orchestrates the AI Agent logic (leveraging OpenAI/Anthropic APIs).
*   **Data Layer/Tooling (Python - vnstock3)**: Serves as the primary data ingestion point for Vietnamese stock market data.

## 3. Core Capabilities
The features of StockMind are divided into the conversational AI capabilities and the visual dashboard tools. 
For a complete breakdown of current and planned capabilities, please see the [Features Documentation](./features/features.md).

### 3.1 Conversational AI Flow
When a user interacts with the Chatbot:
1.  The user's text is sent to the Go backend (`/v1/chat`).
2.  The backend establishes a session and forwards the prompt to the AI Agent (e.g., Anthropic Claude).
3.  The agent decides if it needs data (e.g., "Get the financial statement for VCB"). It then makes a tool call to the internal Python tool server.
4.  The output is streamed back to the React UI in real-time.

### 3.2 Market Dashboard
The dashboard uses traditional REST endpoints (`/v1/stock/*`) to retrieve JSON data representing watchlist status, recent news, and high-level market metrics.

## 4. Setup and Installation

### Dependencies
- Go (v1.21+)
- Node.js (v18+)
- Docker & Docker Compose

### Running the Application Structure
To start the application locally:
1. Start the PostgreSQL database:
   ```bash
   docker-compose up -d
   ```
2. Start the Backend and Frontend nodes simultaneously using the provided `Makefile`:
   ```bash
   make run
   ```

## 5. Directory Structure
```text
StockMind/
├── cmd/                # Go application entrypoints
├── internal/           # Core Go backend logic (agent, server, handling)
├── frontend/           # React + TypeScript + Vite web application
├── docs/               # Project documentation (You are here)
│   ├── features/       # Capability breakdowns and status
│   └── project_documentation.md
├── docker-compose.yml  # Docker composition for database
└── Makefile            # Commands for execution and building
```

## 6. Development Workflow
To run tests on the backend services:
```bash
make test    # Run unit tests
make itest   # Run integration tests against DB
```
