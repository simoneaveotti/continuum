# Export Encryption

Continuum can protect exported archives with a passphrase via `ctx export --encrypt`.

The default format is `aes-gcm-v2`, which uses Argon2id-derived keys and embeds
the parameters needed for decryption.

What is encrypted:

- task archives exported with `ctx export <task> --encrypt`
- project archives exported with `ctx export --project=<name> --encrypt`
- full-session archives exported with `ctx export --session --encrypt`

To decrypt and restore an encrypted archive, use `ctx import` and provide the
same passphrase:

```bash
ctx import ./continuum-history.zip --decrypt
```

Passphrases are read from stdin; interactive entry is safer than piping
literals from shell commands.

What this does not guarantee:

- it is practical file protection, not enterprise key management
- it does not protect a machine that is already compromised
- it does not replace path isolation, local permissions, or keeping sensitive
  Continuum storage unsynced when required
