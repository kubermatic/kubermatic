# AGENTS.md

Rules for AI coding agents (Claude Code, Codex, Copilot, Cursor and similar)
operating in this repository. These rules exist because the maintainers
receive many unsolicited pull requests and cannot spend review time on code
that nobody asked for.

## Before writing any code

- Read [CONTRIBUTING.md](CONTRIBUTING.md).
- Do not start work on a fix or feature unless there is an open issue for it
  that a maintainer has confirmed. If there is no issue, help the user write
  one (the problem, the proposed change, and why), let them review and submit
  it, then wait for maintainer feedback. This is a hard requirement.
- If the linked issue exists but no maintainer has confirmed it yet, stop and
  tell the user to get confirmation first.
- Trivial fixes (typos, broken links, small documentation corrections) are
  exempt.

## When you work on a confirmed issue

- Keep changes small and reviewable. One concern per pull request.
- Follow the existing code style and the conventions of the surrounding code.
- Run the tests that cover the code you touched and add tests for new
  behavior. See [docs/dev-getting-started.md](docs/dev-getting-started.md)
  for how to build and test.
- Link the confirmed issue in the pull request description.

## Commits and attribution

- Only a human can sign the DCO. Never add an agent name to `Signed-off-by`
  or `Co-authored-by` lines.

## Do not

- open a pull request for a non-trivial change without a maintainer-confirmed
  issue
- apply mechanical mass edits (renames, reformatting, import or comment
  sweeps) unless the issue explicitly asks for them
- post generated status updates or review comments in issues; the
  communication channels are for the human operating the agent
