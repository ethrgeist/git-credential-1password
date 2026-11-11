//! [`Op`] only runs `op` with raw arguments and every typed call is built on
//! top of it, so tests can check the exact arguments with a scripted fake.

use std::ffi::OsString;
use std::io::Write;
use std::process::{Command, Stdio};

use serde::Deserialize;
use serde_json::Value;
use zeroize::{Zeroize, Zeroizing};

use crate::{Config, Ctx, Error, Request, Result, Secret, TAG_MARKER};

pub trait Op {
    /// Returns stdout. A failure's error includes `op`'s stderr.
    fn run(&self, args: &[&str], stdin: Option<&[u8]>) -> Result<Zeroizing<Vec<u8>>>;
}

pub struct ExecOp {
    program: OsString,
}

impl ExecOp {
    /// Without a path, `op` is looked up in `PATH` (as `op.exe` on Windows).
    #[must_use]
    pub fn new(op_path: Option<&str>) -> Self {
        Self {
            program: op_path.unwrap_or("op").into(),
        }
    }
}

impl Op for ExecOp {
    fn run(&self, args: &[&str], stdin: Option<&[u8]>) -> Result<Zeroizing<Vec<u8>>> {
        let mut child = Command::new(&self.program)
            .args(args)
            .stdin(if stdin.is_some() {
                Stdio::piped()
            } else {
                Stdio::null()
            })
            .stdout(Stdio::piped())
            // kept apart from stdout so op's warnings can't corrupt JSON output
            .stderr(Stdio::piped())
            .spawn()
            .ctx(&self.program.to_string_lossy())?;

        // write stdin from a second thread while we drain stdout/stderr, so a
        // child that answers before reading all input can't deadlock us
        let output = std::thread::scope(|s| {
            if let (Some(data), Some(mut pipe)) = (stdin, child.stdin.take()) {
                // errors (e.g. op exiting early) show up in the exit status instead
                s.spawn(move || pipe.write_all(data));
            }
            child.wait_with_output()
        })
        .ctx("op")?;

        let stdout = Zeroizing::new(output.stdout);
        if output.status.success() {
            return Ok(stdout);
        }
        let stderr = String::from_utf8_lossy(&output.stderr);
        Err(match stderr.trim() {
            "" => Error(output.status.to_string()),
            msg => Error(format!("{}: {msg}", output.status)),
        })
    }
}

#[derive(Debug, Deserialize)]
pub struct ListItem {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub urls: Vec<ItemUrl>,
}

#[derive(Debug, Deserialize)]
pub struct ItemUrl {
    #[serde(default)]
    pub href: String,
}

#[derive(Deserialize)]
pub struct Field {
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub value: Option<Secret>,
}

#[must_use]
pub fn field<'a>(fields: &'a [Field], label: &str) -> &'a str {
    fields
        .iter()
        .find(|f| f.label == label)
        .and_then(|f| f.value.as_deref())
        .map_or("", String::as_str)
}

fn item_args<'a>(cfg: &'a Config, sub: &'a str, args: &[&'a str]) -> Vec<&'a str> {
    let mut all = vec!["item", sub];
    all.extend(cfg.op_scope.iter().map(String::as_str));
    all.extend_from_slice(args);
    all
}

fn json<'de, T: Deserialize<'de>>(bytes: &'de [u8]) -> Result<T> {
    serde_json::from_slice(bytes).ctx("invalid JSON from op")
}

pub fn list(cfg: &Config, op: &dyn Op) -> Result<Vec<ListItem>> {
    let args = [
        "--categories",
        &cfg.category,
        "--format",
        "json",
        "--tags",
        TAG_MARKER,
    ];
    json(&op.run(&item_args(cfg, "list", &args), None)?)
}

pub fn get_fields(cfg: &Config, op: &dyn Op, id: &str) -> Result<Vec<Field>> {
    let fields = format!("{},{}", cfg.username_field, cfg.password_field);
    let args = ["--format", "json", "--reveal", "--fields", &fields, id];
    json(&op.run(&item_args(cfg, "get", &args), None)?)
}

/// Templates are local to `op`, so no account or vault flags are passed.
pub fn template(op: &dyn Op, category: &str) -> Result<Value> {
    json(&op.run(&["item", "template", "get", category], None)?)
}

pub fn create(cfg: &Config, op: &dyn Op, args: &[&str], stdin: Option<&[u8]>) -> Result<()> {
    op.run(&item_args(cfg, "create", args), stdin).map(drop)
}

pub fn edit(cfg: &Config, op: &dyn Op, args: &[&str]) -> Result<()> {
    op.run(&item_args(cfg, "edit", args), None).map(drop)
}

pub fn delete(cfg: &Config, op: &dyn Op, id: &str) -> Result<()> {
    op.run(&item_args(cfg, "delete", &[id]), None).map(drop)
}

pub fn find_item_id(cfg: &Config, op: &dyn Op, req: &Request) -> Result<Option<String>> {
    if let Some(id) = &cfg.item_id {
        return Ok(Some(id.clone()));
    }
    let url = req.url();
    let items = list(cfg, op).ctx("op item list failed")?;
    Ok(items
        .into_iter()
        .find(|i| i.urls.iter().any(|u| u.href == url))
        .map(|i| i.id))
}

pub fn set_field(item: &mut Value, label: &str, value: &str, kind: &str) -> Result<()> {
    let fields = item
        .as_object_mut()
        .ok_or("item template is not a JSON object")?
        .entry("fields")
        .or_insert_with(|| Value::Array(Vec::new()))
        .as_array_mut()
        .ok_or("item template fields are not a JSON array")?;
    let is_match = |f: &Value| {
        [&f["label"], &f["id"]]
            .iter()
            .any(|v| v.as_str() == Some(label))
    };
    match fields.iter_mut().find(|f| is_match(f)) {
        Some(f) => f["value"] = value.into(),
        None => fields.push(serde_json::json!({ "label": label, "type": kind, "value": value })),
    }
    Ok(())
}

pub struct Wiped(pub Value);

impl Drop for Wiped {
    fn drop(&mut self) {
        fn wipe(v: &mut Value) {
            match v {
                Value::String(s) => s.zeroize(),
                Value::Array(a) => a.iter_mut().for_each(wipe),
                Value::Object(o) => o.values_mut().for_each(wipe),
                _ => {}
            }
        }
        wipe(&mut self.0);
    }
}

impl Wiped {
    pub fn to_json(&self) -> Result<Zeroizing<Vec<u8>>> {
        let mut buf = Zeroizing::new(Vec::with_capacity(4096));
        serde_json::to_writer(&mut *buf, &self.0).ctx("serializing item template")?;
        Ok(buf)
    }
}

#[cfg(test)]
pub(crate) mod mock {
    use std::cell::RefCell;
    use std::collections::VecDeque;

    use super::{Op, Result, Zeroizing};
    use crate::Error;

    pub struct Step {
        pub args: Vec<String>,
        pub stdin: Option<serde_json::Value>,
        pub out: Result<String, String>,
    }

    pub fn ok(args: &[&str], out: &str) -> Step {
        Step {
            args: args.iter().map(|&s| s.into()).collect(),
            stdin: None,
            out: Ok(out.into()),
        }
    }

    pub fn err(args: &[&str], msg: &str) -> Step {
        Step {
            out: Err(msg.into()),
            ..ok(args, "")
        }
    }

    pub struct Script {
        steps: RefCell<VecDeque<Step>>,
        pub calls: RefCell<Vec<Vec<String>>>,
    }

    impl Script {
        pub fn new(steps: Vec<Step>) -> Self {
            Self {
                steps: RefCell::new(steps.into()),
                calls: RefCell::default(),
            }
        }
    }

    impl Op for Script {
        fn run(&self, args: &[&str], stdin: Option<&[u8]>) -> Result<Zeroizing<Vec<u8>>> {
            self.calls
                .borrow_mut()
                .push(args.iter().map(|&s| s.into()).collect());
            let step = self.steps.borrow_mut().pop_front();
            let step = step.unwrap_or_else(|| panic!("unexpected op call: {args:?}"));
            assert_eq!(args, step.args, "op args");
            let stdin = stdin
                .map(|b| serde_json::from_slice::<serde_json::Value>(b).expect("stdin is JSON"));
            assert_eq!(stdin, step.stdin, "op stdin for {args:?}");
            step.out
                .map(|s| Zeroizing::new(s.into_bytes()))
                .map_err(Error)
        }
    }

    impl Drop for Script {
        fn drop(&mut self) {
            if !std::thread::panicking() {
                let left: Vec<_> = self
                    .steps
                    .get_mut()
                    .iter()
                    .map(|s| s.args.clone())
                    .collect();
                assert!(left.is_empty(), "expected op calls not made: {left:?}");
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::mock::{Script, err, ok};
    use super::*;
    use serde_json::json;

    fn fields(json: &str) -> Vec<Field> {
        serde_json::from_str(json).unwrap()
    }

    #[test]
    fn field_lookup() {
        let two = r#"[{"label":"username","value":"alice"},{"label":"password","value":"secret"}]"#;
        let cases = [
            (two, "username", "alice"),
            (r#"[{"label":"username","value":"alice"}]"#, "password", ""),
            ("[]", "username", ""),
            (
                r#"[{"label":"username","value":"first"},{"label":"username","value":"second"}]"#,
                "username",
                "first",
            ),
            (r#"[{"value":"val"}]"#, "", "val"),
            (r#"[{"label":"username","value":null}]"#, "username", ""),
            (r#"[{"label":"username"}]"#, "username", ""),
        ];
        for (input, label, want) in cases {
            assert_eq!(field(&fields(input), label), want, "{input} / {label}");
        }
    }

    fn request(protocol: &str, host: &str) -> Request {
        Request {
            protocol: protocol.into(),
            host: Some(host.into()),
            ..Request::default()
        }
    }

    const LIST: &[&str] = &[
        "item",
        "list",
        "--categories",
        "Login",
        "--format",
        "json",
        "--tags",
        TAG_MARKER,
    ];

    #[test]
    fn find_item_id_by_url() {
        let cases = [
            (
                r#"[{"id":"abc","urls":[{"href":"https://github.com"}]}]"#,
                Some("abc"),
            ),
            (
                r#"[{"id":"abc","urls":[{"href":"https://gitlab.com"}]}]"#,
                None,
            ),
            ("[]", None),
            (
                r#"[{"id":"abc","urls":[{"href":"https://x"},{"href":"https://github.com"}]}]"#,
                Some("abc"),
            ),
            (
                r#"[{"id":"first","urls":[{"href":"https://github.com"}]},{"id":"second","urls":[{"href":"https://github.com"}]}]"#,
                Some("first"),
            ),
            (r#"[{"id":"abc"}]"#, None),
            (
                r#"[{"id":"abc","urls":[{"href":"https://github.com/"}]}]"#,
                None,
            ), // exact match only
        ];
        for (list_json, want) in cases {
            let op = Script::new(vec![ok(LIST, list_json)]);
            let got =
                find_item_id(&Config::default(), &op, &request("https", "github.com")).unwrap();
            assert_eq!(got.as_deref(), want, "{list_json}");
        }
    }

    #[test]
    fn find_item_id_explicit_id_skips_list() {
        let cfg = Config {
            item_id: Some("explicit-id".into()),
            ..Config::default()
        };
        let got =
            find_item_id(&cfg, &Script::new(vec![]), &request("https", "github.com")).unwrap();
        assert_eq!(got.as_deref(), Some("explicit-id"));
    }

    #[test]
    fn find_item_id_error_prefixed_once() {
        let op = Script::new(vec![err(LIST, "exit code: 1: [ERROR] item not found")]);
        let e = find_item_id(&Config::default(), &op, &request("https", "github.com")).unwrap_err();
        assert_eq!(
            e.0,
            "op item list failed: exit code: 1: [ERROR] item not found"
        );
    }

    #[test]
    fn invalid_json_is_an_error() {
        let op = Script::new(vec![ok(LIST, "not json")]);
        let e = list(&Config::default(), &op).unwrap_err();
        assert!(e.0.starts_with("invalid JSON from op"), "{e}");
    }

    #[test]
    fn scope_flags_follow_subcommand() {
        let cfg = Config {
            op_scope: ["--account", "acc", "--vault", "My Vault"]
                .map(String::from)
                .to_vec(),
            ..Config::default()
        };
        let op = Script::new(vec![
            ok(
                &[
                    "item",
                    "get",
                    "--account",
                    "acc",
                    "--vault",
                    "My Vault",
                    "--format",
                    "json",
                    "--reveal",
                    "--fields",
                    "username,password",
                    "id1",
                ],
                "[]",
            ),
            ok(
                &[
                    "item",
                    "delete",
                    "--account",
                    "acc",
                    "--vault",
                    "My Vault",
                    "id1",
                ],
                "",
            ),
            ok(&["item", "template", "get", "Login"], "{}"),
        ]);
        get_fields(&cfg, &op, "id1").unwrap();
        delete(&cfg, &op, "id1").unwrap();
        template(&op, "Login").unwrap();
    }

    #[test]
    fn set_field_fills_builtin_or_appends() {
        let mut item = json!({"fields": [
            {"id": "username", "label": "username", "type": "STRING", "purpose": "USERNAME", "value": ""},
            {"id": "validFrom", "label": "valid from", "type": "DATE", "value": ""},
        ]});
        set_field(&mut item, "username", "alice", "STRING").unwrap();
        set_field(&mut item, "validFrom", "2026-01-01", "STRING").unwrap(); // by id
        set_field(&mut item, "token", "t0k", "CONCEALED").unwrap();
        assert_eq!(
            item,
            json!({"fields": [
                {"id": "username", "label": "username", "type": "STRING", "purpose": "USERNAME", "value": "alice"},
                {"id": "validFrom", "label": "valid from", "type": "DATE", "value": "2026-01-01"},
                {"label": "token", "type": "CONCEALED", "value": "t0k"},
            ]})
        );

        let mut empty = json!({});
        set_field(&mut empty, "password", "p", "CONCEALED").unwrap();
        assert_eq!(
            empty,
            json!({"fields": [{"label": "password", "type": "CONCEALED", "value": "p"}]})
        );

        assert!(set_field(&mut json!([]), "x", "y", "STRING").is_err());
    }

    #[test]
    fn wiped_serializes() {
        let w = Wiped(json!({"a": ["s", {"b": "t"}], "n": 1}));
        assert_eq!(&**w.to_json().unwrap(), br#"{"a":["s",{"b":"t"}],"n":1}"#);
    }
}
