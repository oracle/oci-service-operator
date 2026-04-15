# Contributing to the OCI Service Operator for Kubernetes

We welcome your contributions! There are multiple ways to contribute.

## Opening issues

For bugs or enhancement requests, please file a GitHub issue unless it's
security related. When filing a bug remember that the better written the bug is,
the more likely it is to be fixed. If you think you've found a security
vulnerability, do not raise a GitHub issue and follow the instructions in our
[security policy](./SECURITY.md).

## Contributing code

We welcome your code contributions. Before submitting code via a pull request,
you will need to haved signed the [Oracle Contributor Agreement][OCA] (OCA) and
your commits need to include the following line using the name and e-mail
address you used to sign the OCA:

```text
Signed-off-by: Your Name <you@example.org>
```

This can be automatically added to pull requests by committing with `--sign-off`
or `-s`, e.g.

```text
git commit --signoff
```

Only pull requests from committers that can be verified as having signed the OCA
can be accepted.

## Local validation

The standard repo gate order is:

1. `make fmt`
1. `make vet`
1. `make test`
1. `make build`

`make test` uses envtest assets rooted under `$(TMPDIR)/oci-service-operator-envtest`
by default. In environments where outbound fetches are unreliable:

1. Run `make envtest` once while network access is available to preseed the
   pinned envtest bundle and setup cache.
1. Reuse the preseeded bundle with `ENVTEST_INSTALLED_ONLY=true make test`.
1. If you already have a compatible asset bundle elsewhere, run
   `KUBEBUILDER_ASSETS=/path/to/bin ENVTEST_USE_ENV=true make test`.

When local Go cache paths are not writable, prefix the commands above with
explicit temp-rooted cache directories such as
`GOCACHE=/tmp/osok-gocache GOTMPDIR=/tmp/osok-gotmp TMPDIR=/tmp`.

## Pull request process

1. Ensure there is an issue created to track and discuss the fix or enhancement
   you intend to submit.
1. Fork this repository
1. Create a branch in your fork to implement the changes. We recommend using
   the issue number as part of your branch name, e.g. `1234-fixes`
1. Ensure that any documentation is updated with the changes that are required
   by your change.
1. Ensure that any samples are updated if the base image has been changed.
1. Submit the pull request. *Do not leave the pull request blank*. Explain exactly
   what your changes are meant to do and provide simple steps on how to validate
   your changes. Ensure that you reference the issue you created as well.
1. We will assign the pull request to 2-3 people for review before it is merged.

## Code of conduct

Follow the [Golden Rule](https://en.wikipedia.org/wiki/Golden_Rule). If you'd
like more specific guidelines, see the [Contributor Covenant Code of Conduct][COC].

[OCA]: https://oca.opensource.oracle.com
[COC]: https://www.contributor-covenant.org/version/1/4/code-of-conduct/
