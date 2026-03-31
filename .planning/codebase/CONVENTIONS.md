# CONVENTIONS

## Backend Pattern & Style
- `sqlc` generates all PostgreSQL interactions ensuring statically typed and compiled statements against raw query bugs.
- Configuration managed primarily via environmental bindings. 
- Logging follows standardized structured formatting (`go test -v`, typical Go abstractions used across `service` layers).
- Effective Go rules are to be aggressively observed: Small readable functions, interface adherence, explicit package visibility. 

## Frontend Style
- Built purely on functional React Components, leveraging Context & Custom Hooks for state access.
- React Hook Form is the standardized API for all form interactions ensuring controlled components are not manually maintained. Zod is used alongside for strict payload validation.
- Radix UI is standard for complex interactive primitives instead of rebuilding ARIA components.
- TailWindCSS replaces vanilla CSS with utility-classes. 

## Git and Versioning
- Standard `go mod` dependency versioning management and `package-lock.json` respectively.
- Goose is used for database migration versions, strictly monotonic.
