# How to Contribute

Kubermatic projects are [Apache 2.0 licensed](LICENSE) and accept contributions via
GitHub pull requests. This document outlines some of the conventions on
development workflow, commit message formatting, contact points and other
resources to make it easier to get your contribution accepted.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

Any copyright notices in this repo should specify the authors as "the  Kubermatic Kubernetes Platform contributors".

To sign your work, just add a line like this at the end of your commit message:

```
Signed-off-by: Joe Example <joe@example.com>
```

This can easily be done with the `--signoff` option to `git commit`.

Note that we're requiring all commits in a PR to be signed-off. If you already created a PR, you can sign-off all existing commits by rebasing with the `--signoff` flag.

```
git rebase --signoff origin/master
```

By doing this you state that you can certify the following (from https://developercertificate.org/):
```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Email and Chat

Kubermatic Kubernetes Platform currently uses the general Kubermatic email list and Slack channel:
- Email: [kubermatic-dev](https://groups.google.com/forum/#!forum/kubermatic-dev)
- Slack: #[Slack](http://slack.kubermatic.io/) on Slack

Please avoid emailing maintainers found in the MAINTAINERS file directly. They
are very busy and read the mailing lists.

## Reporting a security vulnerability

Due to their public nature, GitHub and mailing lists are not appropriate places for reporting vulnerabilities. If you suspect you have found a security vulnerability, please do not file a GitHub issue, but instead email security@kubermatic.com with the full details, including steps to reproduce the issue.

## Getting Started

- Fork the repository on GitHub
- Read the [README](README.md) for build and test instructions
- Play with the project, submit bugs, submit patches!

### Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where you want to base your work (usually master).
- Make commits of logical units.
- Make sure your commit messages are in the proper format (see below).
- Push your changes to a topic branch in your fork of the repository.
- Make sure the tests pass, and add any new tests as appropriate.
- Submit a pull request to the original repository.

Thanks for your contributions!
