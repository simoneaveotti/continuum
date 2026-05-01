# Export Encryption

Continuum can protect exported archives with a passphrase via `ctx export --encrypt`.

- the default format is `aes-gcm-v2`, which uses Argon2id-derived keys and embeds the parameters needed for decryption
- this is practical file protection, not enterprise key management
- passphrases are read from stdin; interactive entry is safer than piping literals from shell commands
