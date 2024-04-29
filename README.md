# git-credential-1password

This is a simple Git credential helper that uses the [1Password](https://1password.com/) password manager to retrieve credentials.

When i encountered git servers, which had no SSH and therefore no SSH keys, during my professional work, i wanted to use 1Password to store my credentials instead of [storing passwords in plaintext](https://stackoverflow.com/questions/35942754/how-can-i-save-username-and-password-in-git/35942890#35942890) in the git configuration.
During my professional work, I came across git servers that did not support SSH and, as a result, did not support SSH keys. In order to avoid [storing passwords in plaintext](https://stackoverflow.com/questions/35942754/how-can-i-save-username-and-password-in-git/35942890#35942890) in the git configuration, I decided to use a password manager to save my credentials.

Seriously, do not do that! any run-away-script could grap these and exfil them in various ways, since the `.gitconfig` is usually in a well-defined place.

## 🔐 Features

This credential helper expects a 1Password item with the following fields:

- `username`: The username to use for authentication.
- `password`: The password to use for authentication, could also be a personal access token.

Item name must the same as the hostname of the repository you are authenticating against, e.g. `github.com` or `gitlab.example.net`. If the credentials are unknown, a new item will be created.

The [arguments](https://git-scm.com/docs/gitcredentials) `get`, `store`, and `erase` are supported.

**⚠️ Danger: `erase` will remove the 1Password item matching the hostname!**

### 🚧 Why Go?

It's portable and very lightweight, so it's easy to build and run on different systems. Also it's a compiled language, so you don't have to worry about the user having the correct runtime installed.

### 📦 Why no binary releases?

I don't want to distribute binaries that could be used to steal your 1Password data and i don't want you to have to trust me.

The program logic is very simple and commented, so you can easily audit the code.

Also it's effort to ensure that builds run on different systems, signing binaries and so on.

### 🔄 Alternatives?

If your target system uses Oauth, you might want to try [git-credential-oauth](https://github.com/hickford/git-credential-oauth), althought it is a bit more complex to setup.

## 🏗️ Installation

Clone this repository and build the binary, simplest way could be:

```bash
go build -o git-credential-1password
```

Then copy the binary to a directory in your PATH.

You must have installed and configured the [1Password CLI](https://support.1password.com/command-line-getting-started/) for this to work.

You can test the 1Password CLI by running:

```bash
op whoami
```

This should prompt you to unlock your vault and then print account information.

This helper has no external dependencies other than the 1Password CLI.

Verify that `git` can find the helper by running:

```bash
git credential-1password
```

If you have problems, make sure that the binary is [located in the path](https://superuser.com/a/284351/62691) and [is executable](https://askubuntu.com/a/229592/18504).

## Usage

To use this credential helper, you need to configure Git to use it. You can do this by running:

```bash
git config --global credential.helper "1password"
```

Depending on your setup, it might be a better strategie to just set it as helper for a single host:

```bash
git config --global credential.https://gitlab.example.net.helper "1password"
```

Then, when you push to a repository that requires authentication, 1Password will prompt you to unlock your vault and will then use the credentials stored in the item with the same name as the hostname.

*Note: Depending on your OS, you might geht prompted in different ways for your credentials.*

## 🌳 Collaboration

Feel free to open issues or pull requests.

## 💌 Inspiration

This project was inspired by [git-credential-oauth](https://github.com/hickford/git-credential-oauth)
