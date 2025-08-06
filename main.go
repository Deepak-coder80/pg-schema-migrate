package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// DatabaseConfig holds connection details
type DatabaseConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	SSLMode  string
}

// MigrationOptions holds migration configuration
type MigrationOptions struct {
	Mode         string // "direct" or "export"
	OutputDir    string
	CreateBackup bool
	BackupDir    string
	IncludeRoles bool
	IncludeData  bool // For rollback scripts
	DryRun       bool
}

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *Logger) Info(msg string) {
	l.Printf("[INFO] %s", msg)
}

func (l *Logger) Error(msg string) {
	l.Printf("[ERROR] %s", msg)
}

func (l *Logger) Success(msg string) {
	l.Printf("[SUCCESS] %s", msg)
}

func (l *Logger) Warning(msg string) {
	l.Printf("[WARNING] %s", msg)
}

func (l *Logger) Debug(msg string) {
	l.Printf("[DEBUG] %s", msg)
}

var logger = NewLogger()

func main() {
	var rootCmd = &cobra.Command{
		Use:   "pg-schema-migrate",
		Short: "PostgreSQL schema migration tool",
		Long:  "A CLI tool to migrate PostgreSQL database schemas (structure only) between different hosts",
		Run:   runSchemaMigration,
	}

	// Source database flags
	rootCmd.Flags().StringP("source-host", "s", "localhost", "Source database host")
	rootCmd.Flags().StringP("source-port", "", "5432", "Source database port")
	rootCmd.Flags().StringP("source-user", "u", "postgres", "Source database username")
	rootCmd.Flags().StringP("source-db", "d", "", "Source database name (required)")
	rootCmd.Flags().StringP("source-ssl", "", "require", "Source SSL mode (disable, require, verify-ca, verify-full)")

	// Destination database flags
	rootCmd.Flags().StringP("dest-host", "", "localhost", "Destination database host")
	rootCmd.Flags().StringP("dest-port", "", "5432", "Destination database port")
	rootCmd.Flags().StringP("dest-user", "", "postgres", "Destination database username")
	rootCmd.Flags().StringP("dest-db", "", "", "Destination database name (leave empty to prompt)")
	rootCmd.Flags().StringP("dest-ssl", "", "require", "Destination SSL mode (disable, require, verify-ca, verify-full)")

	// Migration mode flags
	rootCmd.Flags().StringP("mode", "m", "direct", "Migration mode: 'direct' or 'export'")
	rootCmd.Flags().StringP("output-dir", "o", "./schema_migration", "Output directory for export mode")
	rootCmd.Flags().BoolP("dry-run", "", false, "Show what would be done without executing")
	rootCmd.Flags().BoolP("include-roles", "", false, "Include database roles and permissions")
	rootCmd.Flags().BoolP("no-backup", "", false, "Skip creating rollback backup")

	rootCmd.MarkFlagRequired("source-db")

	if err := rootCmd.Execute(); err != nil {
		logger.Error(fmt.Sprintf("Command execution failed: %v", err))
		os.Exit(1)
	}
}

func runSchemaMigration(cmd *cobra.Command, args []string) {
	logger.Info("Starting PostgreSQL schema migration...")

	// Parse migration options
	options, err := parseMigrationOptions(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to parse options: %v", err))
		os.Exit(1)
	}

	// Get source configuration
	sourceConfig, err := getSourceConfig(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get source config: %v", err))
		os.Exit(1)
	}

	// Get destination configuration (only for direct mode)
	var destConfig *DatabaseConfig
	if options.Mode == "direct" {
		destConfig, err = getDestConfig(cmd, sourceConfig.Database)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get destination config: %v", err))
			os.Exit(1)
		}

		// Validate connections
		if err := validateConnections(sourceConfig, destConfig); err != nil {
			logger.Error(fmt.Sprintf("Connection validation failed: %v", err))
			os.Exit(1)
		}
	} else {
		// For export mode, only validate source
		if err := validateSourceConnection(sourceConfig); err != nil {
			logger.Error(fmt.Sprintf("Source connection validation failed: %v", err))
			os.Exit(1)
		}
	}

	// Perform schema migration
	if err := performSchemaMigration(sourceConfig, destConfig, options); err != nil {
		logger.Error(fmt.Sprintf("Schema migration failed: %v", err))
		os.Exit(1)
	}

	logger.Success("Schema migration completed successfully!")
}

func parseMigrationOptions(cmd *cobra.Command) (*MigrationOptions, error) {
	mode, _ := cmd.Flags().GetString("mode")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	includeRoles, _ := cmd.Flags().GetBool("include-roles")
	noBackup, _ := cmd.Flags().GetBool("no-backup")

	if mode != "direct" && mode != "export" {
		return nil, fmt.Errorf("mode must be 'direct' or 'export'")
	}

	return &MigrationOptions{
		Mode:         mode,
		OutputDir:    outputDir,
		CreateBackup: !noBackup,
		BackupDir:    filepath.Join(outputDir, "backup"),
		IncludeRoles: includeRoles,
		IncludeData:  true, // For rollback scripts
		DryRun:       dryRun,
	}, nil
}

func getSourceConfig(cmd *cobra.Command) (*DatabaseConfig, error) {
	sourceHost, _ := cmd.Flags().GetString("source-host")
	sourcePort, _ := cmd.Flags().GetString("source-port")
	sourceUser, _ := cmd.Flags().GetString("source-user")
	sourceDB, _ := cmd.Flags().GetString("source-db")
	sourceSSL, _ := cmd.Flags().GetString("source-ssl")

	if err := validateSSLMode(sourceSSL); err != nil {
		return nil, fmt.Errorf("invalid source SSL mode: %v", err)
	}

	fmt.Printf("Enter password for source database (%s@%s): ", sourceUser, sourceHost)
	sourcePassword, err := readPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to read source password: %v", err)
	}

	return &DatabaseConfig{
		Host:     sourceHost,
		Port:     sourcePort,
		Username: sourceUser,
		Password: sourcePassword,
		Database: sourceDB,
		SSLMode:  sourceSSL,
	}, nil
}

func getDestConfig(cmd *cobra.Command, sourceDBName string) (*DatabaseConfig, error) {
	destHost, _ := cmd.Flags().GetString("dest-host")
	destPort, _ := cmd.Flags().GetString("dest-port")
	destUser, _ := cmd.Flags().GetString("dest-user")
	destDB, _ := cmd.Flags().GetString("dest-db")
	destSSL, _ := cmd.Flags().GetString("dest-ssl")

	if err := validateSSLMode(destSSL); err != nil {
		return nil, fmt.Errorf("invalid destination SSL mode: %v", err)
	}

	fmt.Printf("Enter password for destination database (%s@%s): ", destUser, destHost)
	destPassword, err := readPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to read destination password: %v", err)
	}

	// Ask for destination database name if not provided
	if destDB == "" {
		fmt.Printf("\nDestination database options:\n")
		fmt.Printf("1. Use same name as source (%s)\n", sourceDBName)
		fmt.Printf("2. Use different name\n")
		fmt.Print("Choose option (1 or 2): ")

		reader := bufio.NewReader(os.Stdin)
		choice, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read choice: %v", err)
		}

		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			destDB = sourceDBName
		case "2":
			fmt.Print("Enter destination database name: ")
			destDB, err = reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read database name: %v", err)
			}
			destDB = strings.TrimSpace(destDB)
		default:
			return nil, fmt.Errorf("invalid choice: %s", choice)
		}
	}

	return &DatabaseConfig{
		Host:     destHost,
		Port:     destPort,
		Username: destUser,
		Password: destPassword,
		Database: destDB,
		SSLMode:  destSSL,
	}, nil
}

func validateSSLMode(sslMode string) error {
	validModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range validModes {
		if sslMode == mode {
			return nil
		}
	}
	return fmt.Errorf("must be one of: %s", strings.Join(validModes, ", "))
}

func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // New line after password input
	return string(bytePassword), nil
}

func validateSourceConnection(source *DatabaseConfig) error {
	logger.Info("Validating source database connection...")

	sourceConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		source.Host, source.Port, source.Username, source.Password, source.Database, source.SSLMode)

	sourceDB, err := sql.Open("postgres", sourceConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	if err := sourceDB.Ping(); err != nil {
		return fmt.Errorf("source database ping failed: %v", err)
	}
	logger.Info("Source database connection successful")
	return nil
}

func validateConnections(source, dest *DatabaseConfig) error {
	// Validate source
	if err := validateSourceConnection(source); err != nil {
		return err
	}

	// Validate destination server
	logger.Info("Validating destination database connection...")
	destConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		dest.Host, dest.Port, dest.Username, dest.Password, dest.SSLMode)

	destDB, err := sql.Open("postgres", destConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to destination server: %v", err)
	}
	defer destDB.Close()

	if err := destDB.Ping(); err != nil {
		return fmt.Errorf("destination server ping failed: %v", err)
	}
	logger.Info("Destination server connection successful")

	return nil
}

func performSchemaMigration(source, dest *DatabaseConfig, options *MigrationOptions) error {
	timestamp := time.Now().Format("20060102_150405")

	// Create output directories
	if err := createDirectories(options); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Step 1: Export source schema
	schemaFile := filepath.Join(options.OutputDir, fmt.Sprintf("schema_%s_%s.sql", source.Database, timestamp))
	if err := exportSchema(source, schemaFile, options); err != nil {
		return fmt.Errorf("failed to export schema: %v", err)
	}

	// if options.Mode == "export" {
	// 	logger.Success(fmt.Sprintf("Schema exported to: %s", schemaFile))
	// 	return generateMigrationInstructions(options.OutputDir, schemaFile)
	// }

	// Direct migration mode continues...

	// Step 2: Create backup of destination (if exists and backup enabled)
	var backupFile string
	if options.CreateBackup {
		backupFile = filepath.Join(options.BackupDir, fmt.Sprintf("backup_%s_%s.sql", dest.Database, timestamp))
		if err := createDestinationBackup(dest, backupFile, options); err != nil {
			logger.Warning(fmt.Sprintf("Backup creation failed (continuing): %v", err))
		}
	}

	if options.DryRun {
		logger.Info("DRY RUN MODE - showing what would be done:")
		logger.Info(fmt.Sprintf("1. Drop and recreate database: %s", dest.Database))
		logger.Info(fmt.Sprintf("2. Apply schema from: %s", schemaFile))
		if options.CreateBackup && backupFile != "" {
			logger.Info(fmt.Sprintf("3. Backup created at: %s", backupFile))
		}
		return generateRollbackScript(dest, backupFile, options)
	}

	// Step 3: Drop and recreate destination database
	if err := recreateDestinationDatabase(dest); err != nil {
		return fmt.Errorf("failed to recreate destination database: %v", err)
	}

	// Step 4: Apply schema to destination
	if err := applySchema(dest, schemaFile); err != nil {
		return fmt.Errorf("failed to apply schema: %v", err)
	}

	// Step 5: Generate rollback script
	if err := generateRollbackScript(dest, backupFile, options); err != nil {
		logger.Warning(fmt.Sprintf("Failed to generate rollback script: %v", err))
	}

	return nil
}

func createDirectories(options *MigrationOptions) error {
	dirs := []string{options.OutputDir}
	if options.CreateBackup {
		dirs = append(dirs, options.BackupDir)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func exportSchema(config *DatabaseConfig, outputFile string, options *MigrationOptions) error {
	logger.Info(fmt.Sprintf("Exporting schema from database '%s'...", config.Database))

	// Set environment variables
	os.Setenv("PGPASSWORD", config.Password)
	defer os.Unsetenv("PGPASSWORD")
	os.Setenv("PGSSLMODE", config.SSLMode)
	defer os.Unsetenv("PGSSLMODE")

	// Build pg_dump command for schema only
	args := []string{
		"-h", config.Host,
		"-p", config.Port,
		"-U", config.Username,
		"-d", config.Database,
		"-f", outputFile,
		"--schema-only",   // Schema only, no data
		"--no-owner",      // Don't include ownership commands
		"--no-privileges", // Don't include privilege commands (unless explicitly wanted)
		"--verbose",
		"--no-password",
	}

	// Include roles and privileges if requested
	if options.IncludeRoles {
		// Remove --no-privileges flag
		args = removeFromSlice(args, "--no-privileges")
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %v", err)
	}

	logger.Info("Schema export completed")
	return nil
}

func createDestinationBackup(config *DatabaseConfig, backupFile string, options *MigrationOptions) error {
	// Check if destination database exists
	exists, err := databaseExists(config)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("Destination database doesn't exist, skipping backup")
		return nil
	}

	logger.Info(fmt.Sprintf("Creating backup of destination database '%s'...", config.Database))

	// Set environment variables
	os.Setenv("PGPASSWORD", config.Password)
	defer os.Unsetenv("PGPASSWORD")
	os.Setenv("PGSSLMODE", config.SSLMode)
	defer os.Unsetenv("PGSSLMODE")

	args := []string{
		"-h", config.Host,
		"-p", config.Port,
		"-U", config.Username,
		"-d", config.Database,
		"-f", backupFile,
		"--verbose",
		"--no-password",
	}

	// Include data in backup for complete rollback capability
	if options.IncludeData {
		// Full backup including data
	} else {
		args = append(args, "--schema-only")
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("backup pg_dump failed: %v", err)
	}

	logger.Info("Backup created successfully")
	return nil
}

func recreateDestinationDatabase(config *DatabaseConfig) error {
	// Drop database if exists
	if err := dropDatabaseIfExists(config); err != nil {
		return err
	}

	// Create database
	if err := createDatabase(config); err != nil {
		return err
	}

	return nil
}
func databaseExists(config *DatabaseConfig) (bool, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	// Use quoted identifier to preserve case
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	err = db.QueryRow(query, config.Database).Scan(&exists)
	return exists, err
}

func dropDatabaseIfExists(config *DatabaseConfig) error {
	exists, err := databaseExists(config)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("Destination database doesn't exist, skipping drop")
		return nil
	}

	logger.Info(fmt.Sprintf("Dropping existing database '%s'", config.Database))

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	// Terminate connections to the database
	terminateQuery := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`

	_, err = db.Exec(terminateQuery, config.Database)
	if err != nil {
		logger.Warning(fmt.Sprintf("Could not terminate all connections: %v", err))
	}

	// Drop the database - use quoted identifier to preserve case
	dropQuery := fmt.Sprintf(`DROP DATABASE "%s"`, config.Database)
	_, err = db.Exec(dropQuery)
	if err != nil {
		return err
	}

	logger.Info("Database dropped successfully")
	return nil
}

func createDatabase(config *DatabaseConfig) error {
	logger.Info(fmt.Sprintf("Creating destination database '%s'...", config.Database))

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create database - use quoted identifier to preserve case
	createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, config.Database)
	_, err = db.Exec(createQuery)
	if err != nil {
		return err
	}

	logger.Info("Database created successfully")
	return nil
}
func applySchema(config *DatabaseConfig, schemaFile string) error {
	logger.Info(fmt.Sprintf("Applying schema to destination database '%s'...", config.Database))

	// Set environment variables
	os.Setenv("PGPASSWORD", config.Password)
	defer os.Unsetenv("PGPASSWORD")
	os.Setenv("PGSSLMODE", config.SSLMode)
	defer os.Unsetenv("PGSSLMODE")

	cmd := exec.Command("psql",
		"-h", config.Host,
		"-p", config.Port,
		"-U", config.Username,
		"-d", config.Database,
		"-f", schemaFile,
		"--no-password")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql schema application failed: %v", err)
	}

	logger.Info("Schema applied successfully")
	return nil
}

func generateRollbackScript(config *DatabaseConfig, backupFile string, options *MigrationOptions) error {
	if !options.CreateBackup || backupFile == "" {
		return nil
	}

	rollbackScript := filepath.Join(options.OutputDir, "rollback.sh")
	logger.Info(fmt.Sprintf("Generating rollback script: %s", rollbackScript))

	script := fmt.Sprintf(`#!/bin/bash
# Rollback script generated by pg-schema-migrate
# Created: %s
# Database: %s@%s:%s

echo "WARNING: This will restore the database to its previous state!"
echo "This will DROP the current database and restore from backup."
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" = "yes" ]; then
    echo "Starting rollback..."

    # Set password (you'll need to enter it)
    export PGPASSWORD=""
    export PGSSLMODE="%s"

    # Drop current database
    echo "Dropping current database..."
    psql -h %s -p %s -U %s -d postgres -c "DROP DATABASE IF EXISTS %s;"

    # Create database
    echo "Creating database..."
    psql -h %s -p %s -U %s -d postgres -c "CREATE DATABASE %s;"

    # Restore from backup
    echo "Restoring from backup..."
    psql -h %s -p %s -U %s -d %s -f %s

    echo "Rollback completed!"
else
    echo "Rollback cancelled."
fi
`,
		time.Now().Format("2006-01-02 15:04:05"),
		config.Username, config.Host, config.Port,
		config.SSLMode,
		config.Host, config.Port, config.Username, config.Database,
		config.Host, config.Port, config.Username, config.Database,
		config.Host, config.Port, config.Username, config.Database, backupFile)

	if err := ioutil.WriteFile(rollbackScript, []byte(script), 0755); err != nil {
		return err
	}

	logger.Success(fmt.Sprintf("Rollback script created: %s", rollbackScript))
	return nil
}

// func generateMigrationInstructions(outputDir, schemaFile string) error {
// 	instructionsFile := filepath.Join(outputDir, "README.md")

// 	instructions := fmt.Sprintf(`# Schema Migration Instructions

// Generated: %s

// ## Files Created

// - **%s** - Complete schema export
// - **README.md** - This instruction file

// ## Manual Migration Steps

// ### 1. Review Schema File

// cat %s

// ### 2. Apply to Destination Database

// #### Option A: Direct Application

// # Set environment variables
// export PGPASSWORD="your_dest_password"
// export PGSSLMODE="require"  # or appropriate SSL mode

// # Apply schema
// psql -h dest_host -p 5432 -U postgres -d dest_database -f %s
// ```

// #### Option B: With Backup

// # Create backup first
// pg_dump -h dest_host -p 5432 -U postgres -d dest_database -f backup_$(date +%%Y%%m%%d_%%H%%M%%S).sql

// # Apply schema
// psql -h dest_host -p 5432 -U postgres -d dest_database -f %s
// ```

// ### 3. Verification

// # Check applied tables
// psql -h dest_host -p 5432 -U postgres -d dest_database -c "\dt"

// # Check functions and views
// psql -h dest_host -p 5432 -U postgres -d dest_database -c "\df"
// psql -h dest_host -p 5432 -U postgres -d dest_database -c "\dv"
// ```

// ## Important Notes

// - **This is schema-only migration** - no data is included
// - **Review the schema file** before applying to production
// - **Always create backups** before applying to important databases
// - **Test in staging** environment first

// ## Support

// If you encounter issues, check the PostgreSQL logs and ensure:
// - Destination database exists
// - User has appropriate permissions
// - Network connectivity is available
// - SSL configuration is correct
// `,
// 		time.Now().Format("2006-01-02 15:04:05"),
// 		filepath.Base(schemaFile),
// 		schemaFile,
// 		schemaFile,
// 		schemaFile)

// 	if err := ioutil.WriteFile(instructionsFile, []byte(instructions), 0644); err != nil {
// 		return err
// 	}

// 	logger.Success(fmt.Sprintf("Migration instructions created: %s", instructionsFile))
// 	return nil
// }

// Helper function to remove string from slice
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
