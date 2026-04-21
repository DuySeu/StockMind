# StockMind

StockMind is an AI-powered financial assistant tailored for the Vietnamese stock market. It streamlines the investment research process by providing intuitive access to financial data, intelligent analysis, and real-time market insights. By combining robust data retrieval with intelligent evaluations, StockMind empowers investors and analysts to make data-driven decisions efficiently.

---

## Table of Contents

1. [Features](#features)
2. [Tech Stack](#tech-stack)
3. [Running Locally](#running-locally)

---

## Features

StockMind includes a suite of tools for exploring the market, accessible via our intelligent conversational agent and market dashboard. 

For a comprehensive breakdown of implemented and planned capabilities, please see our [Features Documentation](./docs/features/features.md).

### Core Capabilities

*   **Financial Data Access**: Retrieve and preview official financial statements for any listed company directly through the AI chat.
*   **Fundamental Indicators**: Instantly access key metrics including P/E, P/S, ROA, ROE, EPS, PEG, and more.
*   **Intelligent Evaluation**: Assess stocks systematically using established frameworks like the Piotroski F-Score and Altman Z-Score through automated market research reports.
*   **Event Tracking & News**: Monitor critical financial events, earnings announcements, dividends, and regulatory updates affecting your portfolio.
*   **Market Researcher & Watchlist**: Build your own watchlist of tickers and generate deep-dive automated research reports detailing fundamental analysis.

---

## Tech Stack

*   **Backend**: Go (Golang)
*   **Frontend**: React (Vite, TypeScript)
*   **Data & Tooling Server**: Python (`vnstock3` integration)
*   **Database**: PostgreSQL (via Docker)

---

## Running Locally

To run the StockMind application on your local machine for development or testing, follow these steps.

### Prerequisites
*   [Go](https://go.dev/doc/install) (1.21+)
*   [Node.js](https://nodejs.org/en) (v18+)
*   [Docker](https://docs.docker.com/get-docker/) & Docker Compose
*   (Optional) `make` utility installed.

### 1. Start the Database
The project uses Docker to spin up the required PostgreSQL database. Run docker compose in detached mode:

```bash
docker-compose up -d
```

### 2. Run the Application
You can easily start both the Go backend server and the React frontend development server simultaneously using the provided `Makefile` command:

```bash
make run
```

This single command will:
1. Start the Go backend on the appropriate port.
2. Install the necessary NPM dependencies in the `frontend/` directory.
3. Start the Vite React development server.

### 3. Access the Dashboard
Once both servers are running, open your browser and navigate to the frontend URL (typically `http://localhost:5173`) to start using StockMind.

*(To stop the application, terminate the `make run` process and run `docker-compose down` to stop the database container).*

### 4, TODO:
- Rehandle Streaming workflow, ensure that the agent core working well with both openai and anthropic client using openrouter API key.
- Modify RAG flow to make it work.
- Add MinIO for file storage if needed.
- Add Login for application with best practice authentication.
