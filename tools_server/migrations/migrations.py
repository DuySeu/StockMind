import glob
import hashlib
import asyncpg

from pathlib import Path
from typing import List, Set
from logsim import CustomLogger

log = CustomLogger()


class DatabaseMigrator:
    def __init__(self, connection: asyncpg.Connection):
        self.connection = connection
        self.schema_path = Path("/home/apps/schema")

    async def create_migration_table(self):
        """Create the migration tracking table if it doesn't exist."""
        create_table_sql = """
            CREATE TABLE IF NOT EXISTS migrations (
                id SERIAL PRIMARY KEY,
                script_name VARCHAR(255) NOT NULL UNIQUE,
                script_hash VARCHAR(64) NOT NULL,
                migrated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            );
        """
        try:
            await self.connection.execute(create_table_sql)
            log.info("Migration tracking table created/verified")
        except Exception as e:
            log.error(f"Error creating migration table: {str(e)}")
            raise

    def get_sql_files(self) -> List[Path]:
        """Get all SQL files from schema directory, sorted by filename."""
        sql_files = glob.glob(str(self.schema_path / "*.sql"))
        return sorted([Path(f) for f in sql_files])

    def calculate_file_hash(self, file_path: Path) -> str:
        """Calculate SHA-256 hash of the SQL file content."""
        try:
            with open(file_path, "r", encoding="utf-8") as file:
                content = file.read()
            return hashlib.sha256(content.encode("utf-8")).hexdigest()
        except Exception as e:
            log.error(f"Error calculating hash for {file_path.name}: {str(e)}")
            raise

    async def get_executed_migrations(self) -> Set[str]:
        """Get set of already executed migration script names."""
        try:
            result = await self.connection.fetch("SELECT script_name FROM migrations")
            return {row["script_name"] for row in result}
        except Exception as e:
            log.error(f"Error fetching executed migrations: {str(e)}")
            raise

    async def record_migration(self, script_name: str, script_hash: str):
        """Record a successfully executed migration."""
        try:
            await self.connection.execute(
                "INSERT INTO migrations (script_name, script_hash) VALUES ($1, $2)",
                script_name,
                script_hash,
            )
            log.info(f"Recorded migration: {script_name}")
        except Exception as e:
            log.error(f"Error recording migration {script_name}: {str(e)}")
            raise

    async def execute_sql_file(self, file_path: Path):
        """Execute a single SQL file."""
        script_name = file_path.name
        script_hash = self.calculate_file_hash(file_path)

        try:
            with open(file_path, "r", encoding="utf-8") as file:
                sql_content = file.read()

            # Split SQL content by semicolon and execute each statement
            statements = [
                stmt.strip() for stmt in sql_content.split(";") if stmt.strip()
            ]

            for statement in statements:
                if statement:
                    try:
                        await self.connection.execute(statement)
                        log.debug(f"Executed statement: {statement[:50]}...")
                    except Exception as e:
                        # Log the specific statement that failed
                        log.error(f"Error: {str(e)}")
                        raise

            # Record the successful migration
            await self.record_migration(script_name, script_hash)
            log.info(f"Successfully executed migration: {script_name}")

        except Exception as e:
            log.error(f"Error executing migration {script_name}: {str(e)}")
            raise

    async def run_migrations(self):
        """Run all pending migrations."""
        try:
            # Create migration tracking table
            await self.create_migration_table()

            # Get all SQL files and already executed migrations
            sql_files = self.get_sql_files()
            executed_migrations = await self.get_executed_migrations()

            log.info(f"Found {len(sql_files)} SQL files in schema directory")
            log.info(f"Found {len(executed_migrations)} already executed migrations")

            # Execute pending migrations
            for sql_file in sql_files:
                script_name = sql_file.name

                if script_name in executed_migrations:
                    log.info(f"Skipping already executed migration: {script_name}")
                    continue

                log.info(f"Executing migration: {script_name}")
                await self.execute_sql_file(sql_file)

            log.info("Database migrations completed successfully")

        except Exception as e:
            log.error(f"Migration failed: {str(e)}")
            raise


async def run_database_migrations(connection: asyncpg.Connection):
    """Run database migrations using the provided connection."""
    migrator = DatabaseMigrator(connection)
    await migrator.run_migrations()
