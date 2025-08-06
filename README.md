# pg-schema-migrate

A powerful CLI tool for migrating PostgreSQL database schemas (structure only) between different hosts. This tool safely transfers table structures, indexes, constraints, functions, views, and other schema objects without touching the actual data.

## Features

- **Schema-only migration** - Transfers structure without data
- **Two migration modes**: Direct migration or export for manual application
- **Automatic backups** - Creates rollback scripts and backups
- **Dry run mode** - Preview changes before execution
- **SSL support** - Secure connections with various SSL modes
- **Role migration** - Optionally include database roles and permissions
- **Interactive prompts** - User-friendly configuration experience

## Installation

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/Deepak-coder80/pg-schema-migrate/releases).

#### Linux/macOS
```bash
# Download and install
wget https://github.com/Deepak-coder80/pg-schema-migrate/releases/latest/download/pg-schema-migrate-linux
chmod +x pg-schema-migrate-linux
sudo mv pg-schema-migrate-linux /usr/local/bin/pg-schema-migrate
```

#### Windows
```powershell
# Download pg-schema-migrate-windows.exe and add to PATH
```

### Build from Source

```bash
git clone https://github.com/Deepak-coder80/pg-schema-migrate.git
cd pg-schema-migrate
go build -o pg-schema-migrate
```

## Prerequisites

- **PostgreSQL client tools** (`pg_dump`, `psql`) must be installed and in PATH
- **Network access** to both source and destination PostgreSQL servers
- **Appropriate database permissions** on both source and destination

## Quick Start

### Basic Direct Migration

Migrate schema from one database to another:

```bash
pg-schema-migrate \
  --source-host prod-db.example.com \
  --source-user myuser \
  --source-db myapp_prod \
  --dest-host staging-db.example.com \
  --dest-user myuser \
  --dest-db myapp_staging
```

### Export Mode (Manual Application)

Export schema to files for manual review and application:

```bash
pg-schema-migrate \
  --mode export \
  --source-host prod-db.example.com \
  --source-user myuser \
  --source-db myapp_prod \
  --output-dir ./schema_export
```

## Usage

```bash
pg-schema-migrate [flags]
```

### Required Flags

| Flag | Description |
|------|-------------|
| `--source-db`, `-d` | Source database name |

### Source Database Options

| Flag | Default | Description |
|------|---------|-------------|
| `--source-host`, `-s` | `localhost` | Source database host |
| `--source-port` | `5432` | Source database port |
| `--source-user`, `-u` | `postgres` | Source database username |
| `--source-ssl` | `require` | SSL mode (disable, require, verify-ca, verify-full) |

### Destination Database Options

| Flag | Default | Description |
|------|---------|-------------|
| `--dest-host` | `localhost` | Destination database host |
| `--dest-port` | `5432` | Destination database port |
| `--dest-user` | `postgres` | Destination database username |
| `--dest-db` | (prompt) | Destination database name |
| `--dest-ssl` | `require` | SSL mode |

### Migration Options

| Flag | Default | Description |
|------|---------|-------------|
| `--mode`, `-m` | `direct` | Migration mode: `direct` or `export` |
| `--output-dir`, `-o` | `./schema_migration` | Output directory for files |
| `--dry-run` | `false` | Show what would be done without executing |
| `--include-roles` | `false` | Include database roles and permissions |
| `--no-backup` | `false` | Skip creating rollback backup |

## Examples

### 1. Production to Staging Migration

```bash
pg-schema-migrate \
  --source-host prod.example.com \
  --source-user app_user \
  --source-db production_app \
  --dest-host staging.example.com \
  --dest-user app_user \
  --dest-db staging_app \
  --include-roles
```

### 2. Local Development Setup

```bash
pg-schema-migrate \
  --source-host localhost \
  --source-db myapp_dev \
  --dest-db myapp_test \
  --dry-run
```

### 3. Cross-Cloud Migration

```bash
pg-schema-migrate \
  --source-host aws-rds.amazonaws.com \
  --source-port 5432 \
  --source-user admin \
  --source-db webapp \
  --dest-host gcp-sql.googleapis.com \
  --dest-port 5432 \
  --dest-user admin \
  --dest-db webapp_new \
  --output-dir ./migration_$(date +%Y%m%d)
```

### 4. Export for Manual Review

```bash
pg-schema-migrate \
  --mode export \
  --source-host prod.example.com \
  --source-db critical_app \
  --output-dir ./schema_review \
  --include-roles
```

## Migration Modes

### Direct Mode (`--mode direct`)

- Connects to both source and destination databases
- Creates automatic backup of destination (if exists)
- Drops and recreates destination database
- Applies schema directly
- Generates rollback script

**Use when**: You want automated, immediate migration between databases you control.

### Export Mode (`--mode export`)

- Connects only to source database
- Exports schema to SQL files
- No changes made to any database
- Generates manual application instructions

**Use when**: You need to review changes, have restricted access, or want manual control.

## File Structure

After running the tool, you'll find these files in the output directory:

```
schema_migration/
├── schema_mydb_20240806_143022.sql    # Exported schema
├── rollback.sh                        # Automatic rollback script
└── backup/
    └── backup_mydb_20240806_143022.sql # Destination backup
```

## Security Considerations

### Password Handling
- Passwords are entered securely via hidden input prompts
- Passwords are not stored in command history
- Environment variables are cleared after use

### SSL Configuration
- Always use SSL in production (`--source-ssl require` or higher)
- Verify certificates in production environments
- Use `verify-full` for maximum security

### Database Permissions
Ensure your database user has these permissions:

**Source Database:**
- `CONNECT` privilege
- `USAGE` on schemas
- `SELECT` on system catalogs (for schema reading)

**Destination Database:**
- `CREATEDB` privilege (for database recreation)
- `CONNECT` privilege
- Full privileges on destination database

## Troubleshooting

### Common Issues

#### "pg_dump: command not found"
```bash
# Install PostgreSQL client tools
# Ubuntu/Debian
sudo apt-get install postgresql-client

# macOS
brew install postgresql

# Windows
# Install from https://www.postgresql.org/download/windows/
```

#### "Connection refused"
- Verify host and port are correct
- Check firewall settings
- Ensure PostgreSQL is accepting connections
- Verify SSL configuration

#### "Permission denied"
- Check database user privileges
- Verify user can connect to both databases
- For destination: ensure user can create/drop databases

#### "Database already exists" (in direct mode)
- This is expected - the tool will drop and recreate
- Use `--dry-run` to see what would happen
- Ensure you have backups if needed

### Debug Mode

For troubleshooting, check the verbose output from `pg_dump` and `psql` commands that the tool executes.

## Rollback Procedure

If migration fails or you need to rollback:

1. **Automatic Rollback Script**:
   ```bash
   cd schema_migration
   ./rollback.sh
   ```

2. **Manual Rollback**:
   ```bash
   # Drop current database
   psql -h dest-host -U user -d postgres -c "DROP DATABASE mydb;"

   # Recreate and restore
   psql -h dest-host -U user -d postgres -c "CREATE DATABASE mydb;"
   psql -h dest-host -U user -d mydb -f backup/backup_mydb_timestamp.sql
   ```

## Best Practices

### Pre-Migration
- **Always test in staging first**
- **Create manual backups** of critical databases
- **Review exported schema** files before applying
- **Coordinate with your team** about downtime
- **Verify network connectivity** and permissions

### During Migration
- **Use `--dry-run`** to preview changes
- **Monitor logs** for any errors or warnings
- **Keep terminal session active** during migration
- **Have rollback plan ready**

### Post-Migration
- **Verify schema integrity**:
  ```bash
  psql -h dest-host -d mydb -c "\dt"  # List tables
  psql -h dest-host -d mydb -c "\df"  # List functions
  psql -h dest-host -d mydb -c "\di"  # List indexes
  ```
- **Test application connectivity**
- **Update connection strings** in applications
- **Archive migration files** for future reference

## Support

### Getting Help

1. **Check logs** - The tool provides detailed logging
2. **Use dry-run mode** - Preview changes safely
3. **Check PostgreSQL logs** - Server-side error details
4. **Verify prerequisites** - pg_dump, psql, network access

### Common Commands for Verification

```bash
# Check PostgreSQL client version
pg_dump --version
psql --version

# Test connections
psql -h source-host -U user -d postgres -c "SELECT version();"
psql -h dest-host -U user -d postgres -c "SELECT version();"

# List databases
psql -h host -U user -d postgres -c "\l"
```


## License

This project is licensed under the MIT License
