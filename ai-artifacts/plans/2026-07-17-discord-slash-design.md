# Discord slash migration

Approved by user: full slash migration, no prefix transition.

- `/saham kode`: local popular-symbol autocomplete (manual IDX symbols remain supported), public quote embed, owner-only Refresh button updating the original message.
- `/scan ara|arb|all`: retain existing destination channels; deferred ephemeral execution summary, including partial failures.
- `/verif`: reuse token modal and ephemeral validation flow. Keep existing verification buttons working.
- Register commands on connection using application command upserts. Optional DISCORD_GUILD_ID gives immediate guild-scoped deployment; otherwise global commands. Do not delete unrelated application commands.
- Remove message handler and Message Content intent; retain scheduled scanner and CLI.
- Move bot token to DISCORD_TOKEN. Operator must rotate previously embedded token.
- Defer market requests before external calls. Autocomplete does not call external services.

Rejected: prefix compatibility (requires message content access); network-dependent autocomplete (unnecessary latency and new upstream dependency).

Verification: Go tests for command definitions, autocomplete, symbol validation, interaction routing/defer/update and registration via mocked HTTP; go test ./..., go vet ./.... Live guild smoke test requires operator credentials. Changing command registration scope requires manual cleanup of the previous scope.
