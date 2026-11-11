use std::ffi::OsString;
use std::io::{self, Write};
use std::process::ExitCode;

use git_credential_1password::op::ExecOp;
use git_credential_1password::{Config, Ctx, Error, Request, Result, commands};

const PREFIX: &str = "git-credential-1password: ";
const VERSION: &str = env!("CARGO_PKG_VERSION");

const VALUE_FLAGS: [&str; 8] = [
    "account",
    "vault",
    "category",
    "prefix",
    "username-field",
    "password-field",
    "op-path",
    "id",
];
const BOOL_FLAGS: [&str; 4] = ["erase", "read-only", "version", "help"];

struct Cli {
    cfg: Config,
    op_path: Option<String>,
    allow_erase: bool,
    read_only: bool,
    action: String,
}

enum Parsed {
    Run(Cli),
    Exit(ExitCode),
}

fn usage() {
    eprint!(
        "\
git-credential-1password {VERSION}
usage: git credential-1password [<options>] <action>

Options:
  --account <name>         1Password account (default: op CLI's default)
  --vault <name>           1Password vault (default: op CLI's default)
  --category <name>        1Password item category to search (default: Login)
  --prefix <text>          1Password item name prefix
  --username-field <name>  What field to use for the username (default: username)
  --password-field <name>  What field to use for the password (default: password)
  --erase                  Allow erasing credentials
  --read-only              Only allow reading credentials (disables store and erase)
  --op-path <path>         Path to the op binary
  --id <id>                1Password item unique ID (bypasses URL-based lookup)
  --version                Print version
Flags may also start with a single '-'.

Actions:
  get            Generate credential [called by Git]
  store          Store credential [called by Git]
  erase          Erase credential [called by Git]

See also https://github.com/ethrgeist/git-credential-1password
"
    );
}

/// Long flags may start with a single `-` (`-vault x`), which lexopt would read
/// as a bundle of short flags. Rewrite known names only, so a value such as
/// `--prefix -x` is left alone.
fn normalize(arg: OsString) -> OsString {
    let Some(rest) = arg.to_str().and_then(|s| s.strip_prefix('-')) else {
        return arg;
    };
    let name = rest.split_once('=').map_or(rest, |(name, _)| name);
    if VALUE_FLAGS.contains(&name) || BOOL_FLAGS.contains(&name) {
        format!("-{}", arg.to_string_lossy()).into()
    } else {
        arg
    }
}

fn bool_value(p: &mut lexopt::Parser, name: &str) -> Result<bool, lexopt::Error> {
    let Some(value) = p.optional_value() else {
        return Ok(true);
    };
    match value.to_str() {
        Some("1" | "t" | "T" | "true" | "TRUE" | "True") => Ok(true),
        Some("0" | "f" | "F" | "false" | "FALSE" | "False") => Ok(false),
        _ => Err(format!("invalid boolean value {value:?} for --{name}").into()),
    }
}

fn parse(args: impl IntoIterator<Item = OsString>) -> Result<Parsed, lexopt::Error> {
    use lexopt::prelude::*;

    let mut p = lexopt::Parser::from_args(args.into_iter().map(normalize));
    let mut cfg = Config::default();
    let (mut account, mut vault, mut op_path, mut id) =
        (String::new(), String::new(), String::new(), String::new());
    let (mut allow_erase, mut read_only, mut version) = (false, false, false);
    let mut positional = Vec::new();

    while let Some(arg) = p.next()? {
        match arg {
            Long("account") => account = p.value()?.string()?,
            Long("vault") => vault = p.value()?.string()?,
            Long("category") => cfg.category = p.value()?.string()?,
            Long("prefix") => cfg.prefix = p.value()?.string()?,
            Long("username-field") => cfg.username_field = p.value()?.string()?,
            Long("password-field") => cfg.password_field = p.value()?.string()?,
            Long("op-path") => op_path = p.value()?.string()?,
            Long("id") => id = p.value()?.string()?,
            Long("erase") => allow_erase = bool_value(&mut p, "erase")?,
            Long("read-only") => read_only = bool_value(&mut p, "read-only")?,
            Long("version") => version = bool_value(&mut p, "version")?,
            Short('h') | Long("help") => {
                usage();
                return Ok(Parsed::Exit(ExitCode::SUCCESS));
            }
            Value(action) => {
                positional.push(action);
                positional.extend(p.raw_args()?);
                break;
            }
            _ => return Err(arg.unexpected()),
        }
    }

    if version {
        eprintln!("git-credential-1password {VERSION}");
        return Ok(Parsed::Exit(ExitCode::SUCCESS));
    }
    let Ok([action]) = <[OsString; 1]>::try_from(positional) else {
        usage();
        return Ok(Parsed::Exit(ExitCode::from(2)));
    };

    for (flag, value) in [("--account", account), ("--vault", vault)] {
        if !value.is_empty() {
            cfg.op_scope.extend([flag.to_owned(), value]);
        }
    }
    cfg.item_id = Some(id).filter(|id| !id.is_empty());
    Ok(Parsed::Run(Cli {
        cfg,
        op_path: Some(op_path).filter(|p| !p.is_empty()),
        allow_erase,
        read_only,
        action: action.to_string_lossy().into_owned(),
    }))
}

fn run(cli: &Cli) -> Result<ExitCode> {
    let cfg = &cli.cfg;
    if cfg.username_field.is_empty() || cfg.password_field.is_empty() {
        usage();
        return Err("username and password field must be set".into());
    }
    if cfg.category.is_empty() {
        usage();
        return Err("category must be set".into());
    }

    let op = ExecOp::new(cli.op_path.as_deref());
    let request = || Request::parse(io::stdin().lock()).ctx("reading input");
    match cli.action.as_str() {
        "get" => {
            let Some(response) = commands::get(cfg, &op, &request()?)? else {
                // no output: git moves on to the next credential helper
                return Ok(ExitCode::FAILURE);
            };
            let mut stdout = io::stdout().lock();
            stdout
                .write_all(response.as_bytes())
                .and_then(|()| stdout.flush())
                .ctx("writing output")?;
        }
        "store" | "erase" if cli.read_only => {}
        "store" => commands::store(cfg, &op, &request()?)?,
        // git calls erase on its own after a failed login, so this is not a usage error
        "erase" if !cli.allow_erase => {
            eprintln!("{PREFIX}erase skipped, use the --erase flag to allow erasing credentials");
        }
        "erase" => commands::erase(cfg, &op, &request()?)?,
        other => {
            return Err(Error(format!(
                "It doesn't look like anything to me. (Unknown argument: {other})"
            )));
        }
    }
    Ok(ExitCode::SUCCESS)
}

// Returning ExitCode instead of calling process::exit lets destructors run,
// so secret buffers are wiped on every path.
fn main() -> ExitCode {
    let cli = match parse(std::env::args_os().skip(1)) {
        Ok(Parsed::Run(cli)) => cli,
        Ok(Parsed::Exit(code)) => return code,
        Err(e) => {
            eprintln!("{PREFIX}{e}");
            usage();
            return ExitCode::from(2);
        }
    };
    match run(&cli) {
        Ok(code) => code,
        Err(e) => {
            eprintln!("{PREFIX}{e}");
            ExitCode::FAILURE
        }
    }
}
