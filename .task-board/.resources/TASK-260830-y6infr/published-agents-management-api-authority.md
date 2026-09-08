# Published agents-management API authority

- Repository: `github.com/relux-works/skill-agents-management`
- Remote branch: `codex/local-runtime-public-api-rev5`
- Exact commit: `046baef11790e93a7967230eec760c4563432270`
- Exact tree: `454e2aae8f975f7991edd34cb8cd96171afde790`
- Reviewed patch SHA-256: `471ebb6ce1b3805bcc8213248ebf0f8ac3c67c7ee215eb279c2f4490c76d96cb`
- Review evidence: `agents-management-public-api-rev5-review.md`

The commit is published on a non-default remote branch so agents-infra can
resolve an immutable Go pseudo-version without a local `replace`, copied
interface, or unreviewed API. The final stable semver tag remains gated on the
cumulative consumer validation and canonical Story integration.

Development resolution command:

```bash
go get github.com/relux-works/skill-agents-management@046baef11790e93a7967230eec760c4563432270
```

Do not contact or inspect a live local model/runtime/service/socket while
implementing or validating the adapter. Use static fixtures and fake children.
