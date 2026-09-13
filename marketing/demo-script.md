# Demo GIF Recording Script

Use [asciinema](https://asciinema.org/) or [vhs](https://github.com/charmbracelet/vhs) to record.

## Option 1: asciinema + agg (recommended)

```bash
# Install
brew install asciinema agg  # macOS
# or
pip install asciinema && sudo apt install gifski  # Linux

# Record
asciinema rec demo.cast

# Inside the recording, run these commands:
```

### Demo Script (copy-paste into terminal while recording)

```bash
# 1. Show the problem - MCP server has full access
echo "Without Warden:"
node -e "console.log('Can read:', require('fs').readdirSync('.').join(', '))"

# 2. Install Warden
echo ""
echo "Install Warden:"
npm install -g warden-sandbox-cli

# 3. Create a simple policy
echo ""
echo "Create policy:"
cat > demo-policy.yaml << 'EOF'
command: ["node", "server.js"]
filesystem:
  read: ["./public"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["NODE_ENV"]
EOF

cat demo-policy.yaml

# 4. Show the sandbox summary
echo ""
echo "Run with sandbox:"
warden run --policy demo-policy.yaml -- node -e "
const fs = require('fs');
console.log('Can read ./public:', fs.readdirSync('./public').length > 0);
try { fs.readdirSync('.'); } catch(e) { console.log('Cannot read root: denied'); }
try { require('dns').lookup('evil.com', () => {}); } catch(e) {}
console.log('Sandbox active ✓');
"

# 5. Show audit log
echo ""
echo "Audit log:"
warden logs -n 3
```

## Option 2: vhs (charmbracelet/vhs) - animated tape file

```bash
# Install
brew install vhs

# Create demo.tape
cat > demo.tape << 'EOF'
Output demo.gif

3s

echo "npm install -g warden-sandbox-cli"
Type "npm install -g warden-sandbox-cli"
Enter
2s

Output "\n"

echo "Create policy.yaml"
Type "cat > policy.yaml << 'EOF'"
Enter
Type "command: [\"node\", \"server.js\"]"
Enter
Type "filesystem:"
Enter
Type "  read: [\"./data\"]"
Enter
Type "  write: [\"./output\"]"
Enter
Type "network:"
Enter
Type "  allow: [\"api.github.com\"]"
Enter
Type "EOF"
Enter
1s

Output "\n"

echo "Run sandboxed"
Type "warden run --policy policy.yaml -- node server.js"
Enter
3s

Output "\n"

echo "View audit log"
Type "warden logs -n 5"
Enter
2s
EOF

# Record
vhs demo.tape
```

## Recording Tips

1. **Terminal size:** 80x24 is standard for GIFs
2. **Font:** Use a monospace font (Fira Code, JetBrains Mono)
3. **Theme:** Dark theme looks best (Dracula, One Dark)
4. **Speed:** Keep typing slow enough to read
5. **Length:** 15-30 seconds max for social media

## Post-Production

```bash
# Optimize GIF with gifski
gifski -o demo-optimized.gif demo.gif --quality 80

# Or with gifsicle
gifsicle -O3 --colors 128 demo.gif -o demo-optimized.gif
```

## Upload Locations

- **GitHub README:** Upload to repo, reference with relative path
- **Twitter:** Direct upload (max 15MB)
- **Reddit:** Upload to imgur or use native image upload
- **Dev.to:** Direct upload in editor
