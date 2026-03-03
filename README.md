# git-credential-1password

A Git [credential helper](https://git-scm.com/docs/gitcredentials) that stores and retrieves credentials using [1Password](https://1password.com/) via the [1Password CLI](https://developer.1password.com/docs/cli/).
No external dependencies other than the `op` CLI - no runtime, no config files.

## Requirements

Before using this helper, make sure:

1. The [1Password CLI](https://support.1password.com/command-line-getting-started/) (`op`) is installed and configured. Verify with `op whoami`.
2. Items that the helper should find **must** (when **not** using `--id`):
   - Have the **category** `Login` (default, configurable via `--category`).
   - Be **tagged** `git-credential-1password` (hardcoded, not configurable - this is a safety measure so the helper never touches unrelated items).
   - Have a **URL** field that exactly matches `protocol://host` (e.g. `https://github.com`).
3. The item **name does not matter** for lookup - only the URL is used. The name is cosmetic (set to `[prefix]host` when the helper creates an item).
4. When using `--id`, the helper skips the tag/URL lookup entirely and fetches the item by its unique 1Password ID. The item does not need to be tagged or have a matching URL.

Items created by the helper automatically get the correct category, tag, and URL - you only need to worry about the above when managing items manually.

## Installation

Clone and build:

```bash
go build -o git-credential-1password
```

Copy the binary to a directory in your `PATH`.

Verify Git can find it:

```bash
git credential-1password --version
```

If you have problems, make sure the binary is [in your PATH](https://superuser.com/a/284351/62691) and [is executable](https://askubuntu.com/a/229592/18504).

## Usage

Set as the global credential helper:

```bash
git config --global credential.helper "1password"
```

Or scope it to a single host:

```bash
git config --global credential.https://gitlab.example.net.helper "1password"
```

Or pin a specific 1Password item by its unique ID (rename-proof, works for multiple hostnames):

```bash
git config --global credential.https://git.example.xyz.helper "1password --account=my --vault=Family --id=m5jcyagohuo7usjc76fkpiwuum"
```

The helper supports the standard Git credential operations: `get`, `store`, and `erase`.

When you push to a host that requires authentication, 1Password will prompt you to unlock your vault and then supply the stored credentials.

## Flags

| Flag               | Default        | Description                                                         |
| ------------------ | -------------- | ------------------------------------------------------------------- |
| `--account`        | _(op default)_ | 1Password account to use                                            |
| `--vault`          | _(op default)_ | 1Password vault to use                                              |
| `--category`       | `Login`        | 1Password item category (e.g. `Login`, `API Credential`)            |
| `--prefix`         | _(none)_       | Prefix for item names, e.g. `Git:·` → `Git: github.com`             |
| `--username-field` | `username`     | Field name to read/write the username                               |
| `--password-field` | `password`     | Field name to read/write the password or token                      |
| `--erase`          | `false`        | **⚠️ Danger** - enable erase (deletes the matching 1Password item!) |
| `--read-only`      | `false`        | Disable store and erase - get only                                  |
| `--op-path`        | _(auto)_       | Path to the `op` binary (if not in PATH)                            |
| `--id`             | _(none)_       | 1Password item unique ID (bypasses URL-based lookup)                |
| `--version`        | -              | Print version and exit                                              |

All flags work with both `-` and `--` prefix.

Example with multiple flags:

```bash
git config --global credential.helper "1password --account=myaccount --vault='Dev Vault' --category='API Credential' --prefix='Git: '"
```

**Notes:**

- _Account:_ Sometimes using the account email doesn't work - try the account ID instead.
- _Tokens:_ Providers like [GitHub require a personal access token](https://docs.github.com/en/get-started/git-basics/about-remote-repositories#cloning-with-https-urls) instead of a password. Use `--password-field` to point at the field holding the token.
- _Windows:_ The helper automatically uses `op.exe` on Windows. If you need `--op-path`, use forward slashes: `C:/path/to/op.exe`.
- _Item ID:_ Use `--id` to pin a specific 1Password item by its unique ID. This bypasses URL-based lookup entirely, so renaming items or using multiple hostnames for the same credential won't break anything. Find the ID with `op item list` or in the 1Password app (item → "Copy Item UUID").

## How Items Are Matched

### Get

1. Lists items filtered by **category** + **tag** `git-credential-1password` (scoped to account/vault if set).
2. Finds the item whose URL field exactly matches `protocol://host`.
3. Returns `username` and `password` fields.
4. If no match is found, exits with code 1 (no output) - Git will try the next credential helper in the chain.

### Store

1. Searches for an existing item (same rules as Get).
2. If found **and** the username or password changed → updates the item (only the credential fields; title, URL, and tags are left untouched).
3. If not found → creates a new item with the configured category, tag, URL, title, and credentials.

### Erase

1. Requires the `--erase` flag (disabled by default).
2. If a matching item is found → deletes it.
3. Both `store` and `erase` are silently skipped when `--read-only` is set.

## FAQ

**Why Go?**
Portable, compiles to a single binary, no runtime required. The code is small enough to audit in minutes.

**Why no binary releases?**
To avoid trust issues - you build it yourself and can verify every line. Signing is also costly for a project this small.

**Alternatives?**
For OAuth flows, see [git-credential-oauth](https://github.com/hickford/git-credential-oauth).

**My items are "API Credential", not "Login" - why doesn't it work?**
By default the helper only searches `Login` items. Pass `--category='API Credential'` to match a different category.

**Other Forms of distribution?**
A `flake.nix` is included (`nix build`). A Gentoo ebuild is available via [benknoble's overlay](https://github.com/benknoble/benknoble-gentoo-overlay). Both are community-contributed and not officially supported.

## Contributing

Feel free to open issues or pull requests.

## Inspiration

This project was inspired by [git-credential-oauth](https://github.com/hickford/git-credential-oauth).
