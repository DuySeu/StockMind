# Roadmap

## Core Attributes

- **Goal**: Audit & Refactor Go Backend
- **Strategy**: Sequential transformation of dependencies and interfaces, concluding with state fixes.

## Phases

### Phase 1: Dependency Injection Assembly
- **Goal**: Establish manual Dependency Injection across all Go packages to standardize initialization and remove global states.
- **Requirements**: [ARCH-01]
- **Status**: Not Started

#### Success Criteria
- [ ] Global variables removed from core service implementations.
- [ ] Server startup utilizes the new explicit DI wiring initialization logic.
- [ ] Packages accept configuration and repository interfaces directly.

---

### Phase 2: Database Interface Abstraction
- **Goal**: Introduce strict boundaries between `sqlc` models and the `service` layers by wrapping DB methods.
- **Requirements**: [ARCH-02]
- **Status**: Not Started

#### Success Criteria
- [ ] Creation of an isolation layer above the `/internal/database` generated schema.
- [ ] Services actively updated to utilize the abstracted DB interfaces instead of tight bindings.
- [ ] Compile success strictly maintained with 100% equivalent DB behaviors across HTTP routes.

---

### Phase 3: Secure Context Management
- **Goal**: Secure all LLM streams, websockets, and background processes to properly cascade cancellations, preventing memory leaks.
- **Requirements**: [RELI-01]
- **Status**: Not Started

#### Success Criteria
- [ ] Standardized `r.Context()` propagation deeply through all services and native MCP orchestration.
- [ ] WebSocket disconnect processes explicitly terminate pending background LLM requests.
- [ ] Database requests gracefully cancel and return if the parent context cancels mid-execution.
