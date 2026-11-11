use zeroize::Zeroizing;

use crate::op::{self, Op, Wiped};
use crate::{Config, Ctx, Request, Result, Secret, TAG_MARKER};

/// Returns the response lines for git, or `None` if no item matches.
pub fn get(cfg: &Config, op: &dyn Op, req: &Request) -> Result<Option<Secret>> {
    if req.host.is_none() {
        return Err("host is missing in the input".into());
    }
    let Some(id) = op::find_item_id(cfg, op, req)? else {
        return Ok(None);
    };
    let fields = op::get_fields(cfg, op, &id).ctx("op item get failed")?;
    let username = op::field(&fields, &cfg.username_field);
    let password = op::field(&fields, &cfg.password_field);
    if username.is_empty() || password.is_empty() {
        return Err("username or password is empty, is the item named correctly?".into());
    }
    // a newline or NUL would end the value early and inject extra keys into git's input
    if [username, password]
        .iter()
        .any(|v| v.contains(['\n', '\0']))
    {
        return Err("username or password contains a newline or NUL character".into());
    }
    let mut out = Zeroizing::new(String::with_capacity(username.len() + password.len() + 21));
    for part in ["username=", username, "\npassword=", password, "\n\n"] {
        out.push_str(part);
    }
    Ok(Some(out))
}

pub fn store(cfg: &Config, op: &dyn Op, req: &Request) -> Result<()> {
    let host = req.host.as_deref().unwrap_or("");
    if host.is_empty() {
        return Err("host is missing in the input".into());
    }
    // git itself never stores partial credentials, neither do we
    let (Some(username), Some(password)) = (&req.username, &req.password) else {
        return Ok(());
    };
    if username.is_empty() || password.is_empty() {
        return Ok(());
    }

    let Some(id) = op::find_item_id(cfg, op, req)? else {
        return create(cfg, op, req, host, username, password);
    };

    let fields = op::get_fields(cfg, op, &id).ctx("op item get failed")?;
    if op::field(&fields, &cfg.username_field) == username.as_str()
        && op::field(&fields, &cfg.password_field) == password.as_str()
    {
        return Ok(());
    }
    // Edits pass the credentials as arguments, so they are briefly visible in
    // the process list. Piping a JSON item to `op item edit` would avoid that,
    // but 1Password documents that a JSON template edit overwrites any passkey
    // on the item, and item JSON offers no reliable way to detect one. Keep
    // arguments until 1Password supports passkeys in templates or lets us
    // detect them. Title, URL and tags are left as the user set them.
    let user_arg = Zeroizing::new(format!("{}={}", cfg.username_field, username.as_str()));
    let pass_arg = Zeroizing::new(format!("{}={}", cfg.password_field, password.as_str()));
    op::edit(cfg, op, &[&id, &user_arg, &pass_arg]).ctx("op item edit failed")
}

/// Credentials go in a JSON template on stdin so they never appear in the
/// process list.
fn create(
    cfg: &Config,
    op: &dyn Op,
    req: &Request,
    host: &str,
    username: &str,
    password: &str,
) -> Result<()> {
    let title = format!("{}{host}", cfg.prefix);
    let mut item = Wiped(op::template(op, &cfg.category).ctx("op item template get failed")?);
    if let Some(obj) = item.0.as_object_mut() {
        obj.insert("title".into(), title.as_str().into());
    }
    op::set_field(&mut item.0, &cfg.username_field, username, "STRING")?;
    op::set_field(&mut item.0, &cfg.password_field, password, "CONCEALED")?;
    let stdin = item.to_json()?;

    // no --category: the template carries it, and op rejects having both
    let tags = format!("--tags={TAG_MARKER}");
    let title = format!("--title={title}");
    let url = format!("--url={}", req.url());
    op::create(cfg, op, &[&tags, &title, &url, "-"], Some(&stdin)).ctx("op item create failed")
}

pub fn erase(cfg: &Config, op: &dyn Op, req: &Request) -> Result<()> {
    match op::find_item_id(cfg, op, req)? {
        Some(id) => op::delete(cfg, op, &id).ctx("op item delete failed"),
        None => Ok(()),
    }
}

#[cfg(test)]
#[allow(clippy::too_many_lines, clippy::type_complexity)] // case tables
mod tests {
    use serde_json::{Value, json};

    use super::*;
    use crate::op::mock::{Script, Step, err, ok};

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
    const GITHUB: &str = r#"[{"id":"item-1","urls":[{"href":"https://github.com"}]}]"#;
    const GET: &[&str] = &[
        "item",
        "get",
        "--format",
        "json",
        "--reveal",
        "--fields",
        "username,password",
        "item-1",
    ];
    const ALICE: &str =
        r#"[{"label":"username","value":"alice"},{"label":"password","value":"secret"}]"#;
    const TEMPLATE: &[&str] = &["item", "template", "get", "Login"];
    const LOGIN_TEMPLATE: &str = r#"{"title":"","category":"LOGIN","fields":[
        {"id":"username","type":"STRING","purpose":"USERNAME","label":"username","value":""},
        {"id":"password","type":"CONCEALED","purpose":"PASSWORD","label":"password","password_details":{"strength":"TERRIBLE"},"value":""},
        {"id":"notesPlain","type":"STRING","purpose":"NOTES","label":"notesPlain","value":""}]}"#;
    const GET_INPUT: &str = "protocol=https\nhost=github.com\n\n";
    const STORE_INPUT: &str =
        "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n";

    fn req(input: &str) -> Request {
        Request::parse(input.as_bytes()).unwrap()
    }

    fn with(f: impl FnOnce(&mut Config)) -> Config {
        let mut cfg = Config::default();
        f(&mut cfg);
        cfg
    }

    fn explicit_id() -> Config {
        with(|c| c.item_id = Some("explicit-id".into()))
    }

    fn login_item(title: &str, username: &str, password: &str) -> Value {
        let mut item: Value = serde_json::from_str(LOGIN_TEMPLATE).unwrap();
        item["title"] = title.into();
        item["fields"][0]["value"] = username.into();
        item["fields"][1]["value"] = password.into();
        item
    }

    fn create_args(title: &str) -> Vec<String> {
        [
            "item",
            "create",
            &format!("--tags={TAG_MARKER}"),
            &format!("--title={title}"),
            "--url=https://github.com",
            "-",
        ]
        .map(String::from)
        .to_vec()
    }

    fn create(title: &str, item: Value) -> Step {
        Step {
            args: create_args(title),
            stdin: Some(item),
            out: Ok(String::new()),
        }
    }

    fn check<T: PartialEq + std::fmt::Debug>(name: &str, result: Result<T>, want: Result<T, &str>) {
        match (result, want) {
            (Ok(got), Ok(want)) => assert_eq!(got, want, "{name}"),
            (Err(e), Err(want)) => {
                assert!(e.0.contains(want), "{name}: error {e:?} lacks {want:?}");
            }
            (got, want) => panic!("{name}: got {got:?}, want {want:?}"),
        }
    }

    #[test]
    fn get_cases() {
        let password_is = |p: &str| {
            json!([{"label":"username","value":"alice"},{"label":"password","value":p}]).to_string()
        };
        let cases: Vec<(&str, Config, &str, Vec<Step>, Result<Option<&str>, &str>)> = vec![
            (
                "happy path",
                Config::default(),
                GET_INPUT,
                vec![ok(LIST, GITHUB), ok(GET, ALICE)],
                Ok(Some("username=alice\npassword=secret\n\n")),
            ),
            (
                "explicit id",
                explicit_id(),
                GET_INPUT,
                vec![ok(
                    &[
                        "item",
                        "get",
                        "--format",
                        "json",
                        "--reveal",
                        "--fields",
                        "username,password",
                        "explicit-id",
                    ],
                    ALICE,
                )],
                Ok(Some("username=alice\npassword=secret\n\n")),
            ),
            (
                "missing host",
                Config::default(),
                "protocol=https\n\n",
                vec![],
                Err("host is missing"),
            ),
            (
                "empty host is allowed",
                Config::default(),
                "protocol=https\nhost=\n\n",
                vec![ok(LIST, GITHUB)],
                Ok(None),
            ),
            (
                "not found",
                Config::default(),
                GET_INPUT,
                vec![ok(LIST, "[]")],
                Ok(None),
            ),
            (
                "list error",
                Config::default(),
                GET_INPUT,
                vec![err(LIST, "op not found")],
                Err("op item list failed: op not found"),
            ),
            (
                "get error",
                Config::default(),
                GET_INPUT,
                vec![ok(LIST, GITHUB), err(GET, "connection timeout")],
                Err("op item get failed: connection timeout"),
            ),
            (
                "empty username",
                Config::default(),
                GET_INPUT,
                vec![
                    ok(LIST, GITHUB),
                    ok(
                        GET,
                        r#"[{"label":"username","value":""},{"label":"password","value":"secret"}]"#,
                    ),
                ],
                Err("username or password is empty"),
            ),
            (
                "empty password",
                Config::default(),
                GET_INPUT,
                vec![ok(LIST, GITHUB), ok(GET, &password_is(""))],
                Err("username or password is empty"),
            ),
            (
                "newline in value",
                Config::default(),
                GET_INPUT,
                vec![
                    ok(LIST, GITHUB),
                    ok(GET, &password_is("secret\nhost=evil.example")),
                ],
                Err("newline or NUL"),
            ),
            (
                "NUL in value",
                Config::default(),
                GET_INPUT,
                vec![ok(LIST, GITHUB), ok(GET, &password_is("secret\0"))],
                Err("newline or NUL"),
            ),
            (
                "custom field names",
                with(|c| {
                    c.username_field = "email".into();
                    c.password_field = "token".into();
                }),
                GET_INPUT,
                vec![
                    ok(LIST, GITHUB),
                    ok(
                        &[
                            "item",
                            "get",
                            "--format",
                            "json",
                            "--reveal",
                            "--fields",
                            "email,token",
                            "item-1",
                        ],
                        r#"[{"label":"email","value":"alice@example.com"},{"label":"token","value":"ghp_abc123"}]"#,
                    ),
                ],
                Ok(Some("username=alice@example.com\npassword=ghp_abc123\n\n")),
            ),
        ];
        for (name, cfg, input, steps, want) in cases {
            let got = get(&cfg, &Script::new(steps), &req(input));
            check(
                name,
                got.map(|o| o.map(|s| s.to_string())),
                want.map(|o| o.map(String::from)),
            );
        }
    }

    #[test]
    fn store_cases() {
        let edit = |id: &str, user: &str, pass: &str| ok(&["item", "edit", id, user, pass], "");
        let api_template = r#"{"title":"","category":"API_CREDENTIAL","fields":[
            {"id":"username","type":"STRING","label":"username","value":""},
            {"id":"credential","type":"CONCEALED","label":"credential","value":""}]}"#;
        let cases: Vec<(&str, Config, &str, Vec<Step>, Result<(), &str>)> = vec![
            (
                "create new item via stdin",
                Config::default(),
                STORE_INPUT,
                vec![
                    ok(LIST, "[]"),
                    ok(TEMPLATE, LOGIN_TEMPLATE),
                    create("github.com", login_item("github.com", "alice", "secret")),
                ],
                Ok(()),
            ),
            (
                "create with prefix",
                with(|c| c.prefix = "git-".into()),
                STORE_INPUT,
                vec![
                    ok(LIST, "[]"),
                    ok(TEMPLATE, LOGIN_TEMPLATE),
                    create(
                        "git-github.com",
                        login_item("git-github.com", "alice", "secret"),
                    ),
                ],
                Ok(()),
            ),
            (
                "create with custom password field appends concealed field",
                with(|c| c.password_field = "token".into()),
                STORE_INPUT,
                vec![ok(LIST, "[]"), ok(TEMPLATE, LOGIN_TEMPLATE), {
                    let mut item = login_item("github.com", "alice", "");
                    item["fields"]
                        .as_array_mut()
                        .unwrap()
                        .push(json!({"label":"token","type":"CONCEALED","value":"secret"}));
                    create("github.com", item)
                }],
                Ok(()),
            ),
            (
                "create API Credential",
                with(|c| c.category = "API Credential".into()),
                STORE_INPUT,
                vec![
                    ok(
                        &[
                            "item",
                            "list",
                            "--categories",
                            "API Credential",
                            "--format",
                            "json",
                            "--tags",
                            TAG_MARKER,
                        ],
                        "[]",
                    ),
                    ok(&["item", "template", "get", "API Credential"], api_template),
                    Step {
                        args: [
                            "item",
                            "create",
                            &format!("--tags={TAG_MARKER}"),
                            "--title=github.com",
                            "--url=https://github.com",
                            "-",
                        ]
                        .map(String::from)
                        .to_vec(),
                        stdin: Some(
                            json!({"title":"github.com","category":"API_CREDENTIAL","fields":[
                            {"id":"username","type":"STRING","label":"username","value":"alice"},
                            {"id":"credential","type":"CONCEALED","label":"credential","value":""},
                            {"label":"password","type":"CONCEALED","value":"secret"}]}),
                        ),
                        out: Ok(String::new()),
                    },
                ],
                Ok(()),
            ),
            (
                "create error",
                Config::default(),
                STORE_INPUT,
                vec![ok(LIST, "[]"), ok(TEMPLATE, LOGIN_TEMPLATE), {
                    Step {
                        out: Err("access denied".into()),
                        ..create("github.com", login_item("github.com", "alice", "secret"))
                    }
                }],
                Err("op item create failed: access denied"),
            ),
            (
                "template error",
                Config::default(),
                STORE_INPUT,
                vec![ok(LIST, "[]"), err(TEMPLATE, "unknown category")],
                Err("op item template get failed: unknown category"),
            ),
            (
                "unchanged is a no-op",
                Config::default(),
                STORE_INPUT,
                vec![ok(LIST, GITHUB), ok(GET, ALICE)],
                Ok(()),
            ),
            (
                "password changed",
                Config::default(),
                "protocol=https\nhost=github.com\nusername=alice\npassword=new-secret\n\n",
                vec![
                    ok(LIST, GITHUB),
                    ok(GET, ALICE),
                    edit("item-1", "username=alice", "password=new-secret"),
                ],
                Ok(()),
            ),
            (
                "username changed",
                Config::default(),
                "protocol=https\nhost=github.com\nusername=bob\npassword=secret\n\n",
                vec![
                    ok(LIST, GITHUB),
                    ok(GET, ALICE),
                    edit("item-1", "username=bob", "password=secret"),
                ],
                Ok(()),
            ),
            (
                "edit error",
                Config::default(),
                "protocol=https\nhost=github.com\nusername=alice\npassword=new\n\n",
                vec![
                    ok(LIST, GITHUB),
                    ok(GET, ALICE),
                    err(
                        &["item", "edit", "item-1", "username=alice", "password=new"],
                        "network error",
                    ),
                ],
                Err("op item edit failed: network error"),
            ),
            (
                "get error",
                Config::default(),
                STORE_INPUT,
                vec![ok(LIST, GITHUB), err(GET, "item corrupted")],
                Err("op item get failed: item corrupted"),
            ),
            (
                "list error",
                Config::default(),
                STORE_INPUT,
                vec![err(LIST, "op not found")],
                Err("op item list failed"),
            ),
            (
                "missing host",
                Config::default(),
                "protocol=https\nusername=alice\npassword=secret\n\n",
                vec![],
                Err("host is missing"),
            ),
            (
                "empty host",
                Config::default(),
                "protocol=https\nhost=\nusername=alice\npassword=secret\n\n",
                vec![],
                Err("host is missing"),
            ),
            (
                "only username",
                Config::default(),
                "protocol=https\nhost=github.com\nusername=alice\n\n",
                vec![],
                Ok(()),
            ),
            (
                "only password",
                Config::default(),
                "protocol=https\nhost=github.com\npassword=secret\n\n",
                vec![],
                Ok(()),
            ),
            (
                "empty password",
                Config::default(),
                "protocol=https\nhost=github.com\nusername=alice\npassword=\n\n",
                vec![],
                Ok(()),
            ),
            (
                "explicit id edits, never creates",
                explicit_id(),
                "protocol=https\nhost=github.com\nusername=alice\npassword=new\n\n",
                vec![
                    ok(
                        &[
                            "item",
                            "get",
                            "--format",
                            "json",
                            "--reveal",
                            "--fields",
                            "username,password",
                            "explicit-id",
                        ],
                        ALICE,
                    ),
                    edit("explicit-id", "username=alice", "password=new"),
                ],
                Ok(()),
            ),
        ];
        for (name, cfg, input, steps, want) in cases {
            let creates = steps
                .iter()
                .any(|s| s.args.get(1).is_some_and(|a| a == "create"));
            let op = Script::new(steps);
            check(name, store(&cfg, &op, &req(input)), want);
            if creates {
                let leaked = op
                    .calls
                    .borrow()
                    .iter()
                    .flatten()
                    .any(|a| a.contains("secret"));
                assert!(!leaked, "{name}: password passed as an argument");
            }
        }
    }

    #[test]
    fn erase_cases() {
        let delete = |id: &str| ok(&["item", "delete", id], "");
        let cases: Vec<(&str, Config, Vec<Step>, Result<(), &str>)> = vec![
            (
                "found and deleted",
                Config::default(),
                vec![ok(LIST, GITHUB), delete("item-1")],
                Ok(()),
            ),
            ("not found", Config::default(), vec![ok(LIST, "[]")], Ok(())),
            (
                "delete error",
                Config::default(),
                vec![
                    ok(LIST, GITHUB),
                    err(&["item", "delete", "item-1"], "permission denied"),
                ],
                Err("op item delete failed: permission denied"),
            ),
            (
                "list error",
                Config::default(),
                vec![err(LIST, "op not found")],
                Err("op item list failed"),
            ),
            (
                "explicit id",
                explicit_id(),
                vec![delete("explicit-id")],
                Ok(()),
            ),
        ];
        for (name, cfg, steps, want) in cases {
            check(
                name,
                erase(&cfg, &Script::new(steps), &req(GET_INPUT)),
                want,
            );
        }
    }
}
