# AWS Fargate Deployment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Containerize the React frontend and create Infrastructure-as-Code (Terraform) to deploy StockMind to AWS Fargate.

**Architecture:** Fully Containerized ECS Fargate with an ALB, RDS Postgres, and Cloud Map for internal service discovery.

**Tech Stack:** Docker, Nginx, Terraform, AWS (ECS, ALB, RDS, VPC)

---

## Chunk 1: Containerize Frontend

**Files:**
- Create: `frontend/Dockerfile.prod`
- Create: `frontend/nginx.conf`

- [ ] **Step 1: Write nginx configuration**
Create `frontend/nginx.conf` to serve static files (compiled by Vite) and gracefully fallback to `index.html` for React Router navigation.

- [ ] **Step 2: Write production Dockerfile**
Create `frontend/Dockerfile.prod` using a multi-stage Docker build: 
1. `node:18-alpine` to install dependencies and run `npm run build`.
2. `nginx:alpine` to copy the `dist/` directory and `nginx.conf` to serve it.

- [ ] **Step 3: Test frontend container locally**
Run: `docker build -t stockmind-fe -f frontend/Dockerfile.prod ./frontend && docker run -p 8080:80 stockmind-fe`
Expected: Frontend loads successfully at `http://localhost:8080`.

- [ ] **Step 4: Commit**
```bash
git add frontend/Dockerfile.prod frontend/nginx.conf
git commit -m "chore(deploy): add production Dockerfile and nginx config for frontend"
```

## Chunk 2: Infrastructure as Code (Terraform Setup)

**Files:**
- Create: `infra/main.tf`
- Create: `infra/variables.tf`
- Create: `infra/vpc.tf`

- [ ] **Step 1: Setup Terraform networking**
Define the VPC, Public/Private Subnets, Internet Gateway, and NAT Gateway in `infra/vpc.tf` to establish a secure foundation.

- [ ] **Step 2: Setup Database**
Define the Amazon RDS PostgreSQL instance in private subnets within `infra/main.tf`. Output the connection endpoint.

- [ ] **Step 3: Setup ECS Cluster & ALB**
Define the `aws_ecs_cluster`, `aws_lb` (Application Load Balancer), and route-level Target Groups (`/v1/*` to backend group, `/*` to frontend group).

- [ ] **Step 4: Define ECS Services & Cloud Map**
Define the Fargate Task Definitions and `aws_ecs_service` for the Nginx Frontend, Go Backend, and Python Tooling. Establish an `aws_service_discovery_private_dns_namespace` allowing the Go backend to address the Python service internally.

- [ ] **Step 5: Commit**
```bash
git add infra/
git commit -m "infra: add terraform configuration for AWS Fargate deployment"
```
