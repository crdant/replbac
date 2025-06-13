# Simplified Role Management with Replicated

A few days ago, a question came up on a call about managing Replicated Vendor Portal roles in an infrastructure-as-code style. We do have a [Terraform provider](https://github.com/replicatedhq/terraform-provider-replicated) that's focused on testing scenarios, but it doesn't focus on maintaining your Vendor Portal configuration.

I thought about extending it, but this felt like something that might be handled well with a purpose-built tool. Or maybe I just love building new tools since Claude made it so easy.

## Why Manage Roles as Code?

Role management in any system starts simple and gets complex fast. You begin with a handful of roles for your team members. Or you give them all admin access because you're moving quickly. Then you bring on customer success folks who need read access to support tickets but shouldn't delete applications. Next comes the QA team that needs to create test installations but shouldn't touch production releases.

Before you know it, you've got a bunch of roles and not much of a sense of how to manage them. Or you give up and give people imperfect access just to keep things going.

Managing roles as code solves the problems I've encountered repeatedly. Audit trails matter---who change the "Sales" role (and _why_) is much easier to tell from a PR than an audit log.

## Enter replbac

I built `replbac` to handle exactly this scenario. It's a CLI tool that synchronizes role definitions between local YAML files and the Replicated Vendor Portal API. You define your roles in version-controlled files, then push those definitions to your vendor portal.

Here's what typical role definitions looks like:

```yaml
# roles/viewer.yaml
name: viewer
resources:
  allowed:
    - "kots/app/*/read"
    - "team/support-issues/read"
  denied:
    - "kots/app/*/write"
    - "kots/app/*/delete"
    - "kots/app/*/admin"
```

```yaml
# roles/admin.yaml
name: admin
resources:
  allowed:
    - "**/*"
  denied:
    - "kots/app/*/delete"
```

The format stays simple and readable. Each role gets its own file, which makes diffs clean and merge conflicts rare.

## Practical Workflows

The tool supports bidirectional synchronization, which matters more than you might initially think. Sometimes you need to bootstrap your local configuration from existing portal settings:

```bash
# Pull existing roles from your vendor portal
replbac pull roles/

# Make changes to the YAML files
# Test your changes
replbac sync --dry-run

# Apply the changes
replbac sync roles/
```

Dry-run modes have  saved me several times, so I made sure I added one here.
Role permissions can have subtle implications, and seeing exactly what changes
before applying them prevents surprises. For ongoing management, the sync
command becomes your primary tool, with options to delete remote roles that
don't exist locally or force changes without confirmation prompts for CI/CD
workflows.

That delete flag deserves special attention. By default, the tool won't remove
roles that exist in your vendor portal but not in your local files. This
prevents accidental deletions when you're working with partial role
definitions or testing in different directories. When you want to treat your
local files as the source of truth, you explicitly enable deletions.

## CI/CD Integration

The real power emerges when you integrate this into your deployment pipeline.
A GitHub Actions workflow can automatically sync role changes whenever someone
pushes changes to your roles directory:

```yaml
name: Sync RBAC Roles

on:
  push:
    branches: [ main ]
    paths:
      - 'roles/**'

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install replbac
        run: |
          curl -L https://github.com/crdant/replbac/releases/latest/download/replbac-linux-amd64 -o replbac
          chmod +x replbac
          sudo mv replbac /usr/local/bin/

      - name: Sync roles
        run: replbac sync roles --delete --force
        env:
          REPLICATED_API_TOKEN: ${{ secrets.REPLICATED_API_TOKEN }}
```

This workflow triggers whenever someone pushes changes to the roles directory,
applying role updates automatically with the same review process as any other
code change.

## Configuration and Security

The tool handles API credentials through multiple methods, prioritizing
security and flexibility. You can use environment variables (recommended for
scripts), command-line flags for one-off operations, or standard Replicated
environment variables. For team environments, I recommend storing the API
token in your CI/CD secrets and using environment variables in your workflows.
Never commit API tokens to version control, even in private repositories.

## When This Approach Works Best

This infrastructure-as-code approach to role management is always awesome, but
it really shines when you have multiple teams interacting with Replicated a
need a clearly defined set of roles.

The approach requires more initial setup than clicking through web interfaces,
but it pays dividends as your organization grows and your access patterns
become more sophisticated. Start by pulling your existing roles to see how
they translate to YAML format, then gradually incorporate the tool into your
development workflow.

## Getting Started

I've published `replbac` on GitHub with binaries for major platforms and
comprehensive documentation at
[github.com/crdant/replbac](https://github.com/crdant/replbac). The tool
includes extensive help text and error messages designed to guide you through
common scenarios.

Like most infrastructure-as-code practices, the benefits compound over time as
your team and access requirements grow more complex. The tool reflects my
belief that operations tasks should integrate seamlessly with development
workflows. Rather than context-switching to web interfaces for role
management, you can handle these changes alongside code reviews, feature
branches, and deployment pipelines. That's the kind of workflow integration
that makes teams more productive and systems more reliable.

## Disclaimer

This is not an official Replicated product. I wil do my best to support you on
an as-needed basis, but I make no promises and you can't submit a support
ticket. That said, let us know if it makes sense for us to productize
something like this.
