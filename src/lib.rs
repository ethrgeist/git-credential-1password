//! Git credential helper (<https://git-scm.com/docs/gitcredentials>) backed by
//! the 1Password CLI (`op`).

pub mod commands;
pub mod op;

use std::fmt;
use std::io::BufRead;

use zeroize::Zeroizing;

/// Not configurable on purpose: the helper must never touch items the user
/// didn't mark for it.
pub const TAG_MARKER: &str = "git-credential-1password";

pub type Secret = Zeroizing<String>;

#[derive(Debug, Clone)]
pub struct Config {
    pub category: String,
    pub prefix: String,
    pub username_field: String,
    pub password_field: String,
    pub item_id: Option<String>,
    /// `--account`/`--vault` flags, passed to every `op item` call.
    pub op_scope: Vec<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            category: "Login".into(),
            prefix: String::new(),
            username_field: "username".into(),
            password_field: "password".into(),
            item_id: None,
            op_scope: Vec::new(),
        }
    }
}

/// Shown to the user, so messages must never contain credential values.
#[derive(Debug)]
pub struct Error(pub String);

pub type Result<T, E = Error> = std::result::Result<T, E>;

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for Error {}

impl From<&str> for Error {
    fn from(msg: &str) -> Self {
        Self(msg.into())
    }
}

pub trait Ctx<T> {
    fn ctx(self, msg: &str) -> Result<T>;
}

impl<T, E: fmt::Display> Ctx<T> for std::result::Result<T, E> {
    fn ctx(self, msg: &str) -> Result<T> {
        self.map_err(|e| Error(format!("{msg}: {e}")))
    }
}

#[derive(Debug, Default)]
pub struct Request {
    pub protocol: String,
    /// `None` when git sent no `host` line; `get` treats that differently from an empty host.
    pub host: Option<String>,
    pub username: Option<Secret>,
    pub password: Option<Secret>,
}

impl Request {
    /// Values are kept literally, so passwords with leading or trailing spaces survive.
    pub fn parse(mut r: impl BufRead) -> Result<Self> {
        let mut req = Self::default();
        // one preallocated, wiped buffer: lines up to 8 KiB never reallocate,
        // so no unwiped copies of a password are left behind on the heap
        let mut buf = Zeroizing::new(Vec::with_capacity(8 * 1024));
        loop {
            buf.clear();
            r.read_until(b'\n', &mut buf).ctx("read failed")?;
            let line =
                std::str::from_utf8(&buf).map_err(|_| Error::from("invalid input: not UTF-8"))?;
            let line = line.trim_end_matches(['\r', '\n']);
            if line.is_empty() {
                return Ok(req);
            }
            let Some((key, value)) = line.split_once('=') else {
                return Err(Error(format!("invalid input: {line}")));
            };
            match key.trim() {
                "protocol" => req.protocol = value.into(),
                "host" => req.host = Some(value.into()),
                "username" => req.username = Some(Zeroizing::new(value.into())),
                "password" => req.password = Some(Zeroizing::new(value.into())),
                _ => {}
            }
        }
    }

    #[must_use]
    pub fn url(&self) -> String {
        format!("{}://{}", self.protocol, self.host.as_deref().unwrap_or(""))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse(input: &str) -> Result<Request> {
        Request::parse(input.as_bytes())
    }

    fn fields(r: &Request) -> (&str, Option<&str>, Option<&str>, Option<&str>) {
        (
            &r.protocol,
            r.host.as_deref(),
            r.username.as_ref().map(|s| s.as_str()),
            r.password.as_ref().map(|s| s.as_str()),
        )
    }

    #[test]
    fn parses_valid_input() {
        let cases = [
            (
                "blank line terminator",
                "protocol=https\nhost=github.com\n\n",
                ("https", Some("github.com"), None, None),
            ),
            (
                "EOF terminated",
                "protocol=https\nhost=github.com",
                ("https", Some("github.com"), None, None),
            ),
            ("empty input", "", ("", None, None, None)),
            ("single blank line", "\n", ("", None, None, None)),
            (
                "keys trimmed",
                " protocol =https\n host =github.com\n\n",
                ("https", Some("github.com"), None, None),
            ),
            (
                "values literal",
                "username= alice\npassword= a b \n\n",
                ("", None, Some(" alice"), Some(" a b ")),
            ),
            (
                "values literal EOF",
                "password=\tsecret ",
                ("", None, None, Some("\tsecret ")),
            ),
            (
                "value with equals",
                "password=my=password\n\n",
                ("", None, None, Some("my=password")),
            ),
            (
                "CRLF",
                "protocol=https\r\nhost=github.com\r\n\r\n",
                ("https", Some("github.com"), None, None),
            ),
            (
                "full block",
                "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n",
                ("https", Some("github.com"), Some("alice"), Some("secret")),
            ),
            (
                "single pair EOF",
                "host=example.com",
                ("", Some("example.com"), None, None),
            ),
            ("empty value", "host=\n\n", ("", Some(""), None, None)),
            (
                "last duplicate wins",
                "host=a\nhost=b\n\n",
                ("", Some("b"), None, None),
            ),
            (
                "unknown keys ignored",
                "capability[]=authtype\nwwwauth[]=Basic\nhost=a\n\n",
                ("", Some("a"), None, None),
            ),
            (
                "stops at blank line",
                "host=a\n\nhost=b\n",
                ("", Some("a"), None, None),
            ),
        ];
        for (name, input, want) in cases {
            let req = parse(input).unwrap_or_else(|e| panic!("{name}: {e}"));
            assert_eq!(fields(&req), want, "{name}");
        }
    }

    #[test]
    fn rejects_invalid_input() {
        for input in ["invalid-line\n", "invalid-line", "host=a\nbroken\n\n"] {
            let err = parse(input).unwrap_err();
            assert!(err.0.starts_with("invalid input"), "{input:?}: {err}");
        }
        let err = Request::parse(&b"host=\xff\n\n"[..]).unwrap_err();
        assert_eq!(err.0, "invalid input: not UTF-8");
    }

    #[test]
    fn url_joins_protocol_and_host() {
        assert_eq!(
            parse("protocol=https\nhost=github.com").unwrap().url(),
            "https://github.com"
        );
        assert_eq!(parse("protocol=https").unwrap().url(), "https://");
    }

    #[test]
    fn ctx_prefixes_once() {
        let err: Result<()> = Err::<(), _>("boom").ctx("op item list failed");
        assert_eq!(err.unwrap_err().0, "op item list failed: boom");
    }
}
