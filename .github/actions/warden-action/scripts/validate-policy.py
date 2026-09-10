#!/usr/bin/env python3
"""
Warden Policy Security Validator for CI/CD

This script validates Warden policies for common security issues in CI environments,
with special focus on secrets hygiene and excessive permissions.
"""

import sys
import yaml
import re
import os
from pathlib import Path
from typing import Dict, List, Tuple, Any

class PolicyValidator:
    def __init__(self):
        self.warnings = []
        self.errors = []
        
        # Known secret patterns (common environment variable names)
        self.secret_patterns = [
            r'.*_API_KEY$',
            r'.*_SECRET$',
            r'.*_TOKEN$',
            r'.*_PASSWORD$',
            r'.*_PRIVATE_KEY$',
            r'AWS_.*',
            r'GITHUB_TOKEN',
            r'SLACK_.*',
            r'DATABASE_URL',
            r'.*_ENDPOINT$',
        ]
        
        # Risky network endpoints
        self.risky_domains = [
            'amazonaws.com',
            'azure.com',
            'googleusercontent.com',
            'dropbox.com',
            'onedrive.com',
        ]
        
        # Common CI environment variables that are usually safe
        self.safe_ci_vars = {
            'CI', 'GITHUB_ACTIONS', 'RUNNER_OS', 'RUNNER_ARCH',
            'GITHUB_REPOSITORY', 'GITHUB_SHA', 'GITHUB_REF', 'GITHUB_EVENT_NAME',
            'GITHUB_TOKEN',  # GitHub token is expected in CI
            'NODE_ENV', 'PATH', 'HOME', 'USER', 'PWD', 'TERM', 'LANG', 'LC_ALL'
        }
        
        # CI-specific secrets that are commonly needed (still warn but don't error)
        self.expected_ci_secrets = {
            'GITHUB_TOKEN', 'ANTHROPIC_API_KEY', 'OPENAI_API_KEY'
        }

    def validate_policy(self, policy_path: str) -> bool:
        """Validate a policy file and return True if it passes validation."""
        try:
            with open(policy_path, 'r') as f:
                policy = yaml.safe_load(f)
        except Exception as e:
            self.errors.append(f"Failed to parse policy file: {e}")
            return False
        
        self._validate_structure(policy)
        self._validate_filesystem(policy.get('filesystem', {}))
        self._validate_network(policy.get('network', {}))
        self._validate_environment(policy.get('env', {}))
        self._validate_limits(policy.get('limits', {}))
        
        return len(self.errors) == 0

    def _validate_structure(self, policy: Dict[str, Any]):
        """Validate basic policy structure."""
        required_sections = ['filesystem', 'network', 'env']
        for section in required_sections:
            if section not in policy:
                self.errors.append(f"Missing required section: {section}")

    def _validate_filesystem(self, filesystem: Dict[str, Any]):
        """Validate filesystem permissions."""
        read_paths = filesystem.get('read', [])
        write_paths = filesystem.get('write', [])
        
        # Check for overly broad read permissions
        risky_read_paths = ['/', '/home', '/root', '/etc']
        for path in read_paths:
            if path in risky_read_paths:
                self.warnings.append(f"Overly broad read access: {path}")
        
        # Check for risky write permissions
        risky_write_paths = ['/', '/home', '/root', '/etc', '/usr', '/bin']
        for path in write_paths:
            # Allow /tmp in CI environments
            if path == '/tmp' or path == './tmp':
                continue
            if any(path.startswith(risky) for risky in risky_write_paths):
                self.errors.append(f"Dangerous write access: {path}")
        
        # Recommend temporary directories for writes
        if write_paths and not any('/tmp' in path or './tmp' in path for path in write_paths):
            self.warnings.append("Consider adding /tmp or ./tmp for temporary files")

    def _validate_network(self, network: Dict[str, Any]):
        """Validate network permissions."""
        allowed_hosts = network.get('allow', [])
        
        # Check for overly broad network access
        if '*' in allowed_hosts or '0.0.0.0' in allowed_hosts:
            self.errors.append("Wildcard network access (*) is not allowed in CI")
        
        # Check for risky domains
        for host in allowed_hosts:
            for risky_domain in self.risky_domains:
                if risky_domain in host:
                    self.warnings.append(f"Potentially risky domain: {host}")
        
        # Recommend specific endpoints over broad domains
        broad_domains = ['github.com', 'googleapis.com', 'microsoft.com']
        for host in allowed_hosts:
            if host in broad_domains:
                self.warnings.append(f"Consider using specific API endpoint instead of broad domain: {host}")

    def _validate_environment(self, env: Dict[str, Any]):
        """Validate environment variable permissions."""
        allowed_vars = env.get('allow', [])
        
        # Check for potential secret exposure
        for var in allowed_vars:
            for pattern in self.secret_patterns:
                if re.match(pattern, var, re.IGNORECASE):
                    if var not in self.safe_ci_vars and var not in self.expected_ci_secrets:
                        self.warnings.append(f"Potential secret in environment: {var}")
                    elif var in self.expected_ci_secrets:
                        # Just note expected CI secrets without warning
                        pass
        
        # Check for missing essential variables
        essential_vars = ['PATH', 'HOME']
        for var in essential_vars:
            if var not in allowed_vars:
                self.warnings.append(f"Consider adding essential variable: {var}")
        
        # Check for overly permissive patterns
        if '*' in str(allowed_vars) or any('*' in var for var in allowed_vars):
            self.errors.append("Wildcard environment variables (*) are not allowed")

    def _validate_limits(self, limits: Dict[str, Any]):
        """Validate resource limits."""
        memory_mb = limits.get('memory_mb', 0)
        timeout_s = limits.get('timeout_s', 0)
        
        # Check for reasonable limits
        if memory_mb > 4096:
            self.warnings.append(f"High memory limit: {memory_mb}MB (consider if necessary)")
        
        if timeout_s > 3600:  # 1 hour
            self.warnings.append(f"Long timeout: {timeout_s}s (consider if necessary)")
        
        if memory_mb == 0:
            self.warnings.append("No memory limit set (consider adding one)")
        
        if timeout_s == 0:
            self.warnings.append("No timeout set (consider adding one for CI)")

    def print_results(self):
        """Print validation results."""
        if self.errors:
            print("❌ ERRORS:")
            for error in self.errors:
                print(f"  • {error}")
            print()
        
        if self.warnings:
            print("⚠️  WARNINGS:")
            for warning in self.warnings:
                print(f"  • {warning}")
            print()
        
        if not self.errors and not self.warnings:
            print("✅ Policy validation passed with no issues")
        elif not self.errors:
            print("✅ Policy validation passed with warnings")
        else:
            print("❌ Policy validation failed")

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 validate-policy.py <policy.yaml>")
        sys.exit(1)
    
    policy_path = sys.argv[1]
    
    if not os.path.exists(policy_path):
        print(f"Error: Policy file not found: {policy_path}")
        sys.exit(1)
    
    print(f"🔍 Validating Warden policy: {policy_path}")
    print()
    
    validator = PolicyValidator()
    success = validator.validate_policy(policy_path)
    validator.print_results()
    
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()