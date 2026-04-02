#!/bin/bash
set -e

# Default-deny outbound policy with whitelisted domains.
# This prevents Claude Code (or anything in the container) from
# reaching arbitrary hosts when running with --dangerously-skip-permissions.

# Flush existing rules
sudo iptables -F OUTPUT 2>/dev/null || true

# Allow loopback
sudo iptables -A OUTPUT -o lo -j ACCEPT

# Allow established/related connections
sudo iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow DNS (needed to resolve whitelisted domains)
sudo iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
sudo iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# --- Whitelisted domains ---

# Anthropic API (Claude Code)
sudo iptables -A OUTPUT -d api.anthropic.com -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d claude.ai -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d statsig.anthropic.com -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d sentry.io -p tcp --dport 443 -j ACCEPT

# npm registry
sudo iptables -A OUTPUT -d registry.npmjs.org -p tcp --dport 443 -j ACCEPT

# GitHub
sudo iptables -A OUTPUT -d github.com -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d github.com -p tcp --dport 22 -j ACCEPT
sudo iptables -A OUTPUT -d api.github.com -p tcp --dport 443 -j ACCEPT

# Go modules
sudo iptables -A OUTPUT -d proxy.golang.org -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d sum.golang.org -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -d storage.googleapis.com -p tcp --dport 443 -j ACCEPT

# --- Default deny ---
sudo iptables -A OUTPUT -j REJECT

echo "Firewall initialized: default-deny with whitelisted domains."
sudo iptables -L OUTPUT -n --line-numbers
