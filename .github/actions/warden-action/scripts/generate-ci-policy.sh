#!/bin/bash
# Warden CI Policy Generator
# Generates secure policies optimized for CI/CD environments

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/../templates"

# Default values
OUTPUT_FILE="ci-policy.yaml"
POLICY_TYPE="general"
TRACE_FILE=""
COMMAND=""

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Generate Warden policies optimized for CI/CD environments"
    echo ""
    echo "Options:"
    echo "  -t, --type TYPE       Policy type: ai-review, build, test, deploy"
    echo "  -o, --output FILE     Output policy file (default: ci-policy.yaml)"
    echo "  -c, --command CMD     Command to include in policy"
    echo "  --trace FILE          Generate from existing trace file"
    echo "  --validate            Validate the generated policy"
    echo "  -h, --help            Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --type ai-review --output ai-policy.yaml"
    echo "  $0 --type build --command 'npm run build'"
    echo "  $0 --trace audit.jsonl --validate"
}

generate_policy_header() {
    cat << EOF
# Generated Warden Policy for CI/CD
# Type: $POLICY_TYPE
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# 
# This policy follows CI/CD security best practices:
# - Minimal filesystem access
# - Restricted network endpoints
# - Explicit environment variables only
# - Resource limits for CI efficiency

EOF
}

generate_base_policy() {
    local cmd_array=""
    if [ -n "$COMMAND" ]; then
        cmd_array="command: [\"bash\", \"-c\", \"$COMMAND\"]"
    else
        cmd_array="# command: [\"your-command-here\"]"
    fi
    
    cat << EOF
$cmd_array

filesystem:
  read:
    - "."                    # Repository root
    - "/tmp"                 # Temporary files
    - "/usr/bin"             # System binaries
    - "/bin"                 # Core binaries
    - "/usr/lib"             # System libraries
    - "/lib"                 # Core libraries
    - "/lib64"               # 64-bit libraries
    - "/etc/ssl"             # SSL certificates
    - "/etc/ca-certificates" # CA certificates
    
  write:
    - "./tmp"                # Project temporary files
    - "/tmp"                 # System temporary files

network:
  allow: []                  # No network access by default

env:
  allow:
    # Essential system variables
    - "PATH"
    - "HOME"
    - "USER" 
    - "PWD"
    - "TERM"
    
    # CI environment
    - "CI"
    - "GITHUB_ACTIONS"
    - "RUNNER_OS"

limits:
  memory_mb: 512
  timeout_s: 300             # 5 minutes default
EOF
}

enhance_for_ai_review() {
    cat << 'EOF'

# AI Review Policy Enhancements
filesystem:
  write:
    - "./tmp"
    - "./reports"            # AI analysis reports
    - "./analysis-output"    # Analysis artifacts

network:
  allow:
    - "api.anthropic.com"    # Claude API
    - "api.openai.com"       # OpenAI API
    - "api.github.com"       # GitHub API (for PR comments)

env:
  allow:
    - "PATH"
    - "HOME"
    - "USER"
    - "PWD"
    - "TERM"
    - "CI"
    - "GITHUB_ACTIONS"
    - "RUNNER_OS"
    # AI API access (secrets must be explicitly listed)
    - "ANTHROPIC_API_KEY"
    - "OPENAI_API_KEY"
    # GitHub integration
    - "GITHUB_TOKEN"
    - "GITHUB_REPOSITORY"
    - "GITHUB_SHA"
    - "GITHUB_REF"

limits:
  memory_mb: 1024
  timeout_s: 600             # 10 minutes for AI processing
EOF
}

enhance_for_build() {
    cat << 'EOF'

# Build Policy Enhancements  
filesystem:
  write:
    - "./tmp"
    - "./dist"               # Build output
    - "./build"              # Build artifacts
    - "./out"                # Output directory
    - "./node_modules"       # Package cache
    - "./.next"              # Next.js cache
    - "./target"             # Rust/Java builds
    - "./coverage"           # Test coverage

network:
  allow:
    - "registry.npmjs.org"   # NPM packages
    - "registry.yarnpkg.com" # Yarn packages
    - "pypi.org"             # Python packages
    - "proxy.golang.org"     # Go modules
    - "github.com"           # Git dependencies
    - "codeload.github.com"  # GitHub archives

env:
  allow:
    - "PATH"
    - "HOME"
    - "USER"
    - "PWD"
    - "TERM"
    - "CI"
    - "GITHUB_ACTIONS"
    - "RUNNER_OS"
    # Build environment
    - "NODE_ENV"
    - "NODE_OPTIONS"
    - "NPM_CONFIG_CACHE"
    - "YARN_CACHE_FOLDER"
    - "PYTHONPATH"
    - "GOPATH"
    - "GOCACHE"

limits:
  memory_mb: 2048
  timeout_s: 1800            # 30 minutes for builds
EOF
}

enhance_for_test() {
    cat << 'EOF'

# Test Policy Enhancements
filesystem:
  write:
    - "./tmp"
    - "./coverage"           # Coverage reports
    - "./test-results"       # Test outputs
    - "./.nyc_output"        # NYC coverage
    - "./junit.xml"          # JUnit results

network:
  allow:
    - "registry.npmjs.org"   # For test dependencies
    # Add test service endpoints as needed

env:
  allow:
    - "PATH"
    - "HOME"
    - "USER"
    - "PWD"
    - "TERM"
    - "CI"
    - "GITHUB_ACTIONS"
    - "RUNNER_OS"
    # Test environment
    - "NODE_ENV"
    - "TEST_ENV"

limits:
  memory_mb: 1024
  timeout_s: 900             # 15 minutes for tests
EOF
}

validate_policy() {
    local policy_file="$1"
    echo "🔍 Validating generated policy..."
    
    if [ -f "$SCRIPT_DIR/validate-policy.py" ]; then
        python3 "$SCRIPT_DIR/validate-policy.py" "$policy_file"
    else
        echo "⚠️  Policy validator not found, skipping validation"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            POLICY_TYPE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -c|--command)
            COMMAND="$2"
            shift 2
            ;;
        --trace)
            TRACE_FILE="$2"
            shift 2
            ;;
        --validate)
            VALIDATE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

echo "🔧 Generating Warden CI policy..."
echo "Type: $POLICY_TYPE"
echo "Output: $OUTPUT_FILE"

# Generate policy
{
    generate_policy_header
    
    if [ -n "$TRACE_FILE" ] && [ -f "$TRACE_FILE" ]; then
        echo "📊 Generating from trace file: $TRACE_FILE"
        # Use warden init if available, otherwise fall back to template
        if command -v warden &> /dev/null; then
            echo "# Generated from trace: $TRACE_FILE"
            warden init --log "$TRACE_FILE" --output /dev/stdout 2>/dev/null | tail -n +2
        else
            echo "⚠️  Warden not found, using template instead"
            generate_base_policy
        fi
    else
        generate_base_policy
        
        case "$POLICY_TYPE" in
            "ai-review")
                enhance_for_ai_review
                ;;
            "build")
                enhance_for_build
                ;;
            "test")
                enhance_for_test
                ;;
            "deploy")
                echo "# Deploy policies require manual customization"
                echo "# Add specific deployment endpoints and credentials"
                ;;
        esac
    fi
} > "$OUTPUT_FILE"

echo "✅ Policy generated: $OUTPUT_FILE"

# Validate if requested
if [ "$VALIDATE" = true ]; then
    validate_policy "$OUTPUT_FILE"
fi

echo ""
echo "📝 Next steps:"
echo "1. Review the generated policy for your specific needs"
echo "2. Add any missing environment variables to env.allow"
echo "3. Adjust network.allow for required endpoints"
echo "4. Test the policy with: warden run --policy $OUTPUT_FILE -- <command>"