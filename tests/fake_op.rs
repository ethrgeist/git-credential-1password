//! This test binary doubles as the fake `op` (run as `fake-op <mode>`), which
//! works on every OS, unlike a shell script.

use std::io::{self, Read, Write};
use std::process::ExitCode;

use git_credential_1password::op::{ExecOp, Op};

fn fake_op(mode: &str) -> ExitCode {
    match mode {
        "warn" => {
            eprintln!("[WARN] a new version of op is available");
            println!(r#"[{{"id":"abc123","urls":[{{"href":"https://example.com"}}]}}]"#);
            ExitCode::SUCCESS
        }
        "fail" => {
            eprintln!("[ERROR] item not found");
            ExitCode::FAILURE
        }
        "fail-silent" => ExitCode::from(3),
        // echoes while still reading, so a runner that writes all of stdin
        // before reading stdout deadlocks on large payloads
        "echo-stdin" => match io::copy(&mut io::stdin().lock(), &mut io::stdout().lock()) {
            Ok(_) => ExitCode::SUCCESS,
            Err(_) => ExitCode::FAILURE,
        },
        "stdin-len" => {
            let mut buf = Vec::new();
            io::stdin().read_to_end(&mut buf).unwrap();
            print!("{}", buf.len());
            ExitCode::SUCCESS
        }
        _ => ExitCode::from(99),
    }
}

fn fake() -> ExecOp {
    let exe = std::env::current_exe().unwrap();
    ExecOp::new(Some(exe.to_str().unwrap()))
}

fn stderr_ignored_on_success() {
    let out = fake().run(&["fake-op", "warn"], None).unwrap();
    let text = String::from_utf8(out.to_vec()).unwrap();
    assert_eq!(
        text.trim(),
        r#"[{"id":"abc123","urls":[{"href":"https://example.com"}]}]"#
    );
}

fn stderr_included_on_failure() {
    let err = fake().run(&["fake-op", "fail"], None).unwrap_err();
    assert!(err.0.ends_with(": [ERROR] item not found"), "{err}");
    let err = fake().run(&["fake-op", "fail-silent"], None).unwrap_err();
    assert!(err.0.contains('3'), "exit status in {err}");
}

fn stdin_is_piped() {
    let out = fake()
        .run(&["fake-op", "echo-stdin"], Some(br#"{"fields":[]}"#))
        .unwrap();
    assert_eq!(&**out, br#"{"fields":[]}"#);
}

fn large_stdin_does_not_deadlock() {
    let payload: Vec<u8> = b"abcdefghijklmnopqrstuvwxyz"
        .iter()
        .copied()
        .cycle()
        .take(4 << 20)
        .collect();
    let out = fake()
        .run(&["fake-op", "echo-stdin"], Some(&payload))
        .unwrap();
    assert!(
        *out == payload,
        "echoed {} of {} bytes",
        out.len(),
        payload.len()
    );
}

fn stdin_is_empty_without_input() {
    // op must never wait on (or read) the stdin git gave us
    let out = fake().run(&["fake-op", "stdin-len"], None).unwrap();
    assert_eq!(&**out, b"0");
}

fn missing_binary_names_it() {
    let err = ExecOp::new(Some("definitely-not-op-4c1f"))
        .run(&["item", "list"], None)
        .unwrap_err();
    assert!(err.0.starts_with("definitely-not-op-4c1f: "), "{err}");
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args.first().map(String::as_str) == Some("fake-op") {
        return fake_op(args.get(1).map_or("", String::as_str));
    }

    let tests: [(&str, fn()); 6] = [
        ("stderr_ignored_on_success", stderr_ignored_on_success),
        ("stderr_included_on_failure", stderr_included_on_failure),
        ("stdin_is_piped", stdin_is_piped),
        (
            "large_stdin_does_not_deadlock",
            large_stdin_does_not_deadlock,
        ),
        ("stdin_is_empty_without_input", stdin_is_empty_without_input),
        ("missing_binary_names_it", missing_binary_names_it),
    ];
    let mut failed = 0;
    for (name, test) in tests {
        let ok = std::panic::catch_unwind(test).is_ok();
        println!("test {name} ... {}", if ok { "ok" } else { "FAILED" });
        failed += usize::from(!ok);
    }
    io::stdout().flush().unwrap();
    if failed == 0 {
        ExitCode::SUCCESS
    } else {
        ExitCode::FAILURE
    }
}
