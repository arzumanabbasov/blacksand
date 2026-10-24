# Releases

Source: https://github.com/arzumanabbasov/blacksand

Downloads: https://github.com/arzumanabbasov/blacksand/releases

Container image: ghcr.io/arzumanabbasov/blacksand

## Before tagging

Run CI against the intended revision. Review the staged file list and scan for secrets. Installation databases, signing keys, provider credentials, transcripts, browser logs, binaries and generated exports do not belong in a source commit.

Update CHANGELOG.md with concrete user-visible changes and known limitations. Verify first setup, login, persistence, a controlled task, findings and restart on the platforms being advertised. Live E2B checks require your own account; unit tests do not certify paid-provider behavior.

## Publish a version

From a reviewed, green main branch:

~~~sh
git tag -a v0.1.0 -m "Blacksand v0.1.0"
git push origin v0.1.0
~~~

Use the actual next version rather than reusing the example. The release workflow exports the frontend, builds Linux amd64/arm64, macOS amd64/arm64 and Windows amd64 packages, generates SHA256SUMS and creates a GitHub Release. A separate job publishes Linux amd64/arm64 images to GHCR using the repository's GitHub token.

No personal registry password is required. Workflows run on GitHub-hosted runners; running CI does not use a maintainer's local Docker installation. Main-branch pushes run checks and deploy the static introduction; they do not tag or publish executable releases.

## Local packages

Use build.sh with BLACKSAND_BUILD_VERSION, BLACKSAND_TARGETS, BLACKSAND_RELEASE and BLACKSAND_PACKAGE as appropriate. Inspect its options before a cross-platform build. Validate archive contents and checksums, including adapters, documentation and notices.

Automatic upstream fetching is disabled. Users should back up the database and artifacts, stop active work deliberately and install a reviewed build. Re-test existing configurations before replacing a live installation.
