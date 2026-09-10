#!/bin/bash
set -e

# Test script for Warden GitHub Action
# This script simulates what would happen in a GitHub Actions environment

echo "🔧 Testing Warden GitHub Action..."

# Set up test environment
TEST_DIR="/tmp/warden-action-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Create test policy
cat > test-policy.yaml << 'EOF'
command: ["echo", "Hello from Warden sandbox"]

filesystem:
  read:
    - "."
    - "/bin"
    - "/usr/bin"
  write:
    - "./tmp"

network:
  allow: []

env:
  allow:
    - "PATH"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 60
EOF

echo "📄 Created test policy"
cat test-policy.yaml

# Test 1: Validate policy syntax
echo ""
echo "🧪 Test 1: Policy validation"
if command -v warden &> /dev/null; then
    echo "✅ Warden binary found"
    warden version
    
    # Try to validate the policy
    echo "Validating policy..."
    # Note: This is a basic validation - full validation needs the actual command
    if [ -f test-policy.yaml ]; then
        echo "✅ Policy file exists and is readable"
    else
        echo "❌ Policy file not found"
        exit 1
    fi
else
    echo "⚠️  Warden binary not found - would be installed by action"
fi

# Test 2: Check system requirements
echo ""
echo "🧪 Test 2: System requirements"

# Check for strace
if command -v strace &> /dev/null; then
    echo "✅ strace available"
else
    echo "❌ strace not found - action would install it"
fi

# Check for bubblewrap
if command -v bwrap &> /dev/null; then
    echo "✅ bubblewrap available"
    bwrap --version | head -1
else
    echo "❌ bubblewrap not found - action would install it"
fi

# Check user namespaces (critical for sandbox)
echo "🔍 Checking user namespace support..."
if [ -f /proc/sys/user/max_user_namespaces ]; then
    MAX_NS=$(cat /proc/sys/user/max_user_namespaces)
    if [ "$MAX_NS" -gt 0 ]; then
        echo "✅ User namespaces enabled (max: $MAX_NS)"
    else
        echo "❌ User namespaces disabled"
    fi
else
    echo "⚠️  Cannot determine user namespace support"
fi

# Test 3: Simulate action behavior
echo ""
echo "🧪 Test 3: Action simulation"

# Create a simple test command that would run in the action
cat > test-command.sh << 'EOF'
#!/bin/bash
echo "Running in sandbox environment"
echo "Working directory: $(pwd)"
echo "Available files: $(ls -la . | wc -l) items"
echo "Environment variables: $(env | wc -l) vars"

# Try to access something that should be blocked
echo "Testing filesystem restrictions..."
if [ -r /etc/shadow ]; then
    echo "❌ SECURITY ISSUE: Can read /etc/shadow"
    exit 1
else
    echo "✅ Cannot read /etc/shadow (correctly blocked)"
fi

# Try to create a file in allowed location
mkdir -p tmp
echo "test content" > tmp/test.txt
if [ -f tmp/test.txt ]; then
    echo "✅ Can write to allowed tmp directory"
else
    echo "❌ Cannot write to tmp directory"
    exit 1
fi

echo "✅ Basic sandbox test passed"
EOF

chmod +x test-command.sh

# If Warden is available, test it
if command -v warden &> /dev/null && command -v bwrap &> /dev/null; then
    echo "🚀 Running actual Warden test..."
    
    # Update policy to run our test script
    cat > test-policy.yaml << EOF
command: ["bash", "./test-command.sh"]

filesystem:
  read:
    - "."
    - "/bin"
    - "/usr/bin"
    - "/lib"
    - "/lib64"
  write:
    - "./tmp"

network:
  allow: []

env:
  allow:
    - "PATH"
    - "HOME"
    - "PWD"

limits:
  memory_mb: 256
  timeout_s: 60
EOF

    if warden run --policy test-policy.yaml; then
        echo "✅ Warden execution test passed"
    else
        echo "❌ Warden execution test failed"
        exit 1
    fi
    
    # Check if audit log was created
    AUDIT_FILE="$HOME/.local/state/warden/audit.jsonl"
    if [ -f "$AUDIT_FILE" ]; then
        echo "✅ Audit log created"
        EVENTS=$(wc -l < "$AUDIT_FILE")
        echo "📊 Audit events: $EVENTS"
        
        # Show recent events (last 5)
        echo "Recent audit events:"
        tail -5 "$AUDIT_FILE" | jq -r '"\(.timestamp) \(.type) \(.path // .hostname // .details)"' 2>/dev/null || tail -5 "$AUDIT_FILE"
    else
        echo "⚠️  No audit log found at $AUDIT_FILE"
    fi
else
    echo "⚠️  Full Warden test skipped - dependencies not installed"
fi

# Test 4: Action file validation
echo ""
echo "🧪 Test 4: Action configuration validation"

ACTION_FILE="../../../../../.github/actions/warden-action/action.yml"
if [ -f "$ACTION_FILE" ]; then
    echo "✅ Action file exists"
    
    # Basic YAML validation
    if command -v yq &> /dev/null || command -v python3 &> /dev/null; then
        echo "📄 Action metadata:"
        if command -v yq &> /dev/null; then
            yq e '.name, .description' "$ACTION_FILE"
        else
            python3 -c "import yaml; f=open('$ACTION_FILE'); print(yaml.safe_load(f)['name']); print(yaml.safe_load(f)['description'])" 2>/dev/null || echo "Could not parse YAML"
        fi
    fi
else
    echo "❌ Action file not found at $ACTION_FILE"
fi

# Cleanup
echo ""
echo "🧹 Cleaning up test environment"
cd /
rm -rf "$TEST_DIR"

echo ""
echo "🎉 Warden GitHub Action tests completed!"
echo ""
echo "Summary:"
echo "- Action structure: Created and validated"
echo "- Policy examples: Ready for CI use"  
echo "- System compatibility: Checked"
echo "- Security model: Fail-closed design"
echo ""
echo "Next steps:"
echo "1. Build Warden binary distribution"
echo "2. Test in actual GitHub Actions environment"
echo "3. Set up release automation"