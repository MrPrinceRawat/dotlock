# dotlock

Encrypt your `.env` files. Commit them safely. Share with your team.

<!-- ![demo](./demo.gif) -->

**dotlock** encrypts `.env` files into a single `.dotlock` vault file using a shared passphrase. Fully offline!

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/mrprincerawat/dotlock/main/install.sh | sh
```

### Other methods

```bash
# Homebrew
brew install mrprincerawat/tap/dotlock

# Go
go install github.com/mrprincerawat/dotlock@latest
```

Or download binaries from [Releases](https://github.com/mrprincerawat/dotlock/releases).

## Quick Start

```bash
# 1. Initialize (encrypts all .env files)
dotlock init

# 2. Commit the vault
git add .dotlock .env.example .dotlock.readme
git commit -m "add encrypted env files"

# 3. On another machine, unlock
dotlock unlock
```

## Commands

| Command                      | Description                                              |
| ---------------------------- | -------------------------------------------------------- |
| `dotlock init`               | Detect `.env` files, encrypt them, set up git protection |
| `dotlock lock [env]`         | Encrypt `.env` files into the vault                      |
| `dotlock unlock [env]`       | Decrypt environments from the vault                      |
| `dotlock diff [env1] [env2]` | Compare environments                                     |
| `dotlock ls`                 | List environments in the vault                           |
| `dotlock doctor`             | Diagnose setup health                                    |
| `dotlock scan`               | Scan codebase for hardcoded secrets                      |

## How It Works

1. **Encryption**: Passphrase → Argon2id → AES-256-GCM
2. **Storage**: All environments stored in a single `.dotlock` JSON file
3. **Key caching**: Derived key cached in `~/.dotlock/keys/` after first use
4. **Git protection**: `.gitignore` + `.git/info/exclude` + pre-commit hook
5. **Auto-lock**: Pre-commit hook automatically re-locks on commit

## CI/CD

Set the `DOTLOCK_PASSPHRASE` environment variable:

```yaml
# GitHub Actions
env:
  DOTLOCK_PASSPHRASE: ${{ secrets.DOTLOCK_PASSPHRASE }}

steps:
  - run: dotlock unlock
```

## Security

- **Argon2id** key derivation (time=1, memory=64MB, threads=4)
- **AES-256-GCM** authenticated encryption
- Cached keys stored with `0600` permissions
- Pre-commit hook blocks `.env` files and scans for secrets

## License

MIT
