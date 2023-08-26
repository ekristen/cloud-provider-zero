# Cloud Provider Zero

Helper tool to get some of the benefits of a cloud provider with kubernetes without the overhead of the cloud provider

Cloud Providers are cool, but their operators are not always friendly and tend to make a lot of assumption about your
cluster, this doesn't often work well when you aren't using one of the managed kubernetes flavors.

This tool is designed to help provide some of the benefits of the cloud provider operators without the overhead.

## Features

- AWS: Set `Node.Spec.ProviderID` from Label Information

### AWS: Node.Spec.ProviderID

**TL;DR** - Allow Karpenter to work in AWS with non-EKS clusters.

The mutating webhook server will set the `ProviderID` of the node, which is useful for several reasons, to include
but not limited to [karpenter](https://karpenter.sh) based on label information that can be set on the node during
registration time.

If you are using a non-EKS deployment but still want to have the benefits of tooling and providers written for EKS and
other cloud distributions this solution can be helpful.

- `cpz.ekristen.dev/instance-id=i-000000000000`
- `cpz.ekristen.dev/provider=aws`
- `topology.kubernetes.io/zone=us-east-2a`

These labels are used to build a `providerID` is it is currently empty. This would be the case if you aren't running
an actual cloud provider operator in your cluster.

## Building

The following will build binaries in snapshot order.

```console
goreleaser --clean --snapshot
```

## Configure

1. Rename Repository
2. Generate Cosign Keys
3. Update `.goreleaser.yml`, search/replace go-project-template with new project name, adjust GitHub owner
4. Update `main.go`,
5. Update `go.mod`, rename go project (using IDE is best so renames happen across all files)

### Signing

1. Create a password
   - Recommend exporting in environment as `COSIGN_PASSWORD` using something like [direnv](http://direnv.net)
2. Generate cosign keys `cosign generate-key-pair`
3. Create GitHub Action Secrets
   - `COSIGN_KEY` -> populate with cosign.key value
   - `COSIGN_PASSWORD` -> populate with password from step 1

### Releases

In order for Release Drafter and GoReleaser to work properly you have to create a PAT to run Release Drafter
so it's actions against the repository can trigger other workflows. Unfortunately there is no way to trigger 
a workflow from a workflow if both are run by the automatically generated GitHub Actions secret.

1. Create PAT that has write contents permissions to the repository
2. Create GitHub Action Secret
   - `RELEASE_DRAFTER_SECRET` -> populated with PAT from step 1
3. Done

## Documentation

The project is built to have the documentation right alongside the code in the `docs/` directory leveraging Mkdocs Material.

In the root of the project exists mkdocs.yml which drives the configuration for the documentation.

This README.md is currently copied to `docs/index.md` and the documentation is automatically published to the GitHub
pages location for this repository using a GitHub Action workflow. It does not use the `gh-pages` branch.

### Running Locally

```console
make docs-serve
```

OR (if you have docker)

```console
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```
