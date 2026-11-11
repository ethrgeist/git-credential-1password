use std::io::Write;
use std::process::{Command, Stdio};

const PREFIX: &str = "git-credential-1password: ";
const INPUT: &str = "protocol=https\nhost=example.com\n\n";

/// Returns (exit code, stdout, stderr).
fn run(args: &[&str], stdin: &str) -> (i32, String, String) {
    let mut child = Command::new(env!("CARGO_BIN_EXE_git-credential-1password"))
        .args(args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    // the binary may exit before reading stdin (e.g. usage errors)
    let _ = child.stdin.take().unwrap().write_all(stdin.as_bytes());
    let out = child.wait_with_output().unwrap();
    let text = |b: Vec<u8>| String::from_utf8(b).unwrap();
    (
        out.status.code().unwrap(),
        text(out.stdout),
        text(out.stderr),
    )
}

#[test]
fn version_flag() {
    for flag in ["--version", "-version", "--version=true"] {
        let (code, stdout, stderr) = run(&[flag], "");
        assert_eq!(code, 0, "{flag}");
        assert_eq!(
            stderr,
            format!("git-credential-1password {}\n", env!("CARGO_PKG_VERSION")),
            "{flag}"
        );
        assert_eq!(stdout, "");
    }
}

#[test]
fn help_exits_zero() {
    for flag in ["-h", "-help", "--help"] {
        let (code, _, stderr) = run(&[flag], "");
        assert_eq!(code, 0, "{flag}");
        assert!(stderr.contains("usage:"), "{flag}: {stderr}");
    }
}

#[test]
fn wrong_number_of_actions_is_usage_error() {
    for args in [&[][..], &["get", "extra"], &["get", "--vault", "x"]] {
        let (code, _, stderr) = run(args, "");
        assert_eq!(code, 2, "{args:?}");
        assert!(stderr.contains("usage:"), "{args:?}: {stderr}");
    }
}

#[test]
fn bad_flags_are_usage_errors() {
    for args in [
        &["--nope", "get"][..],
        &["--erase=maybe", "erase"],
        &["--vault"],
    ] {
        let (code, _, stderr) = run(args, "");
        assert_eq!(code, 2, "{args:?}");
        assert!(stderr.starts_with(PREFIX), "{args:?}: {stderr}");
        assert!(stderr.contains("usage:"), "{args:?}: {stderr}");
    }
}

#[test]
fn unknown_action() {
    let (code, _, stderr) = run(&["unknown"], "");
    assert_eq!(code, 1);
    assert_eq!(
        stderr,
        format!("{PREFIX}It doesn't look like anything to me. (Unknown argument: unknown)\n")
    );
}

#[test]
fn read_only_store_and_erase_are_noops() {
    for args in [
        &["-read-only", "store"][..],
        &["--read-only", "erase"],
        &["-vault", "x", "-read-only", "--erase", "erase"],
    ] {
        let (code, stdout, stderr) = run(args, INPUT);
        assert_eq!(
            (code, stdout.as_str(), stderr.as_str()),
            (0, "", ""),
            "{args:?}"
        );
    }
}

#[test]
fn erase_without_flag_is_skipped() {
    for args in [
        &["erase"][..],
        &["-erase=false", "erase"],
        &["--erase=0", "erase"],
    ] {
        let (code, _, stderr) = run(args, INPUT);
        assert_eq!(code, 0, "{args:?}");
        assert_eq!(
            stderr,
            format!("{PREFIX}erase skipped, use the --erase flag to allow erasing credentials\n")
        );
    }
}

#[test]
fn empty_required_settings() {
    for (args, msg) in [
        (
            &["--username-field=", "get"][..],
            "username and password field must be set",
        ),
        (
            &["-password-field", "", "get"],
            "username and password field must be set",
        ),
        (&["--category=", "get"], "category must be set"),
    ] {
        let (code, _, stderr) = run(args, INPUT);
        assert_eq!(code, 1, "{args:?}");
        assert!(stderr.contains("usage:"), "{args:?}");
        assert!(
            stderr.ends_with(&format!("{PREFIX}{msg}\n")),
            "{args:?}: {stderr}"
        );
    }
}

#[test]
fn input_errors() {
    let missing_op = "--op-path=definitely-not-op-4c1f";
    let (code, _, stderr) = run(&[missing_op, "get"], "malformed\n");
    assert_eq!(
        (code, stderr.as_str()),
        (
            1,
            "git-credential-1password: reading input: invalid input: malformed\n"
        )
    );

    let (code, _, stderr) = run(&[missing_op, "get"], "protocol=https\n\n");
    assert_eq!(
        (code, stderr.as_str()),
        (
            1,
            "git-credential-1password: host is missing in the input\n"
        )
    );
}

#[test]
fn op_failure_is_reported() {
    let (code, stdout, stderr) = run(&["-op-path", "definitely-not-op-4c1f", "get"], INPUT);
    assert_eq!(code, 1);
    assert_eq!(stdout, "");
    assert!(
        stderr.starts_with(&format!(
            "{PREFIX}op item list failed: definitely-not-op-4c1f: "
        )),
        "{stderr}"
    );
}
