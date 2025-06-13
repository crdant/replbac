package cmd

import (
	"os"
	"strings"
	"time"
)

// GenerateManPage generates the complete man page content in groff format
func GenerateManPage() (string, error) {
	var content strings.Builder

	// Man page header
	content.WriteString(".TH REPLBAC 1 \"")
	content.WriteString(time.Now().Format("January 2006"))
	content.WriteString("\" \"replbac ")
	content.WriteString(Version)
	content.WriteString("\" \"User Commands\"\n")

	// NAME section
	content.WriteString(".SH NAME\n")
	content.WriteString("replbac \\- Replicated RBAC Synchronization Tool\n")

	// SYNOPSIS section
	content.WriteString(".SH SYNOPSIS\n")
	content.WriteString(".B replbac\n")
	content.WriteString("[\\fIOPTIONS\\fR] \\fICOMMAND\\fR [\\fIARGS\\fR...]\n")

	// DESCRIPTION section
	content.WriteString(".SH DESCRIPTION\n")
	content.WriteString("\\fBreplbac\\fR is a command-line tool for synchronizing Role-Based Access Control (RBAC)\n")
	content.WriteString("configurations between local YAML files and the Replicated Vendor Portal API.\n")
	content.WriteString("It allows you to manage team permissions as code, providing version control\n")
	content.WriteString("and automated deployment of role definitions.\n")
	content.WriteString(".PP\n")
	content.WriteString("Key features include bidirectional synchronization, dry-run mode for previewing\n")
	content.WriteString("changes, flexible configuration options, and comprehensive error handling.\n")

	// COMMANDS section
	content.WriteString(".SH COMMANDS\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBsync\\fR [\\fIdirectory\\fR]\n")
	content.WriteString("Synchronize local role files to Replicated API. Reads role definitions from\n")
	content.WriteString("local YAML files and synchronizes them with the Replicated platform.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBpull\\fR [\\fIdirectory\\fR]\n")
	content.WriteString("Pull role definitions from Replicated API to local files. Downloads existing\n")
	content.WriteString("role definitions and creates local YAML files.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBversion\\fR\n")
	content.WriteString("Print version information including build details.\n")

	// OPTIONS section
	content.WriteString(".SH OPTIONS\n")
	content.WriteString(".SS Global Options\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--api-token\\fR \\fITOKEN\\fR\n")
	content.WriteString("Replicated API token for authentication.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--config\\fR \\fIFILE\\fR\n")
	content.WriteString("Path to configuration file.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--confirm\\fR\n")
	content.WriteString("Automatically confirm destructive operations.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--log-level\\fR \\fILEVEL\\fR\n")
	content.WriteString("Set log level: debug, info, warn, error.\n")
	content.WriteString(".SS Sync Command Options\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--dry-run\\fR\n")
	content.WriteString("Preview changes without applying them.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--diff\\fR\n")
	content.WriteString("Preview changes with detailed diffs (implies --dry-run).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--delete\\fR\n")
	content.WriteString("Delete remote roles not present in local files.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--force\\fR\n")
	content.WriteString("Skip confirmation prompts (requires --delete).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--verbose\\fR\n")
	content.WriteString("Enable info-level logging to stderr.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--debug\\fR\n")
	content.WriteString("Enable debug-level logging to stderr.\n")
	content.WriteString(".SS Pull Command Options\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--force\\fR\n")
	content.WriteString("Overwrite existing files.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--dry-run\\fR\n")
	content.WriteString("Preview changes without applying them.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fB--diff\\fR\n")
	content.WriteString("Preview changes with detailed diffs.\n")

	// ENVIRONMENT section
	content.WriteString(".SH ENVIRONMENT\n")
	content.WriteString("Configuration can be provided via environment variables as an alternative to CLI flags:\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBREPLICATED_API_TOKEN\\fR\n")
	content.WriteString("Replicated API token (for replicated CLI compatibility).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBREPLBAC_API_TOKEN\\fR\n")
	content.WriteString("Replicated API token (alternative to REPLICATED_API_TOKEN).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBREPLBAC_CONFIG\\fR\n")
	content.WriteString("Path to configuration file.\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBREPLBAC_CONFIRM\\fR\n")
	content.WriteString("Automatically confirm operations (true/false).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fBREPLBAC_LOG_LEVEL\\fR\n")
	content.WriteString("Log level (debug, info, warn, error).\n")
	content.WriteString(".PP\n")
	content.WriteString("Environment variables have lower precedence than CLI flags but higher than config files.\n")

	// FILES section
	content.WriteString(".SH FILES\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fI~/.config/replbac/config.yaml\\fR\n")
	content.WriteString("User configuration file (Linux).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fI~/Library/Preferences/com.replicated.replbac/config.yaml\\fR\n")
	content.WriteString("User configuration file (macOS).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fI~/.replbac/config.yaml\\fR\n")
	content.WriteString("User configuration file (Windows and fallback).\n")
	content.WriteString(".TP\n")
	content.WriteString("\\fI*.yaml, *.yml\\fR\n")
	content.WriteString("Role definition files in YAML format.\n")

	// EXAMPLES section
	content.WriteString(".SH EXAMPLES\n")
	content.WriteString(".PP\n")
	content.WriteString("Synchronize roles from current directory:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac sync\\fR\n")
	content.WriteString(".RE\n")
	content.WriteString(".PP\n")
	content.WriteString("Preview sync changes without applying:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac sync --dry-run\\fR\n")
	content.WriteString(".RE\n")
	content.WriteString(".PP\n")
	content.WriteString("Sync with detailed diff output:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac sync --diff\\fR\n")
	content.WriteString(".RE\n")
	content.WriteString(".PP\n")
	content.WriteString("Sync and delete remote roles not in local files:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac sync --delete\\fR\n")
	content.WriteString(".RE\n")
	content.WriteString(".PP\n")
	content.WriteString("Pull roles from API to local files:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac pull ./roles\\fR\n")
	content.WriteString(".RE\n")
	content.WriteString(".PP\n")
	content.WriteString("Preview pull operation:\n")
	content.WriteString(".RS\n")
	content.WriteString("\\fBreplbac pull --dry-run\\fR\n")
	content.WriteString(".RE\n")

	// SEE ALSO section
	content.WriteString(".SH SEE ALSO\n")
	content.WriteString("\\fByaml\\fR(1), \\fBcurl\\fR(1)\n")
	content.WriteString(".PP\n")
	content.WriteString("Replicated Vendor Portal documentation:\n")
	content.WriteString("https://docs.replicated.com/vendor/\n")

	// AUTHOR section
	content.WriteString(".SH AUTHOR\n")
	content.WriteString("Charles Dant <crdant@replicated.com>\n")

	return content.String(), nil
}

// WriteManPageToFile writes the man page content to a file
func WriteManPageToFile(filename string) error {
	content, err := GenerateManPage()
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(content), 0644)
}
