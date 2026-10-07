# Data and Backups

## Data directory

Everything lives in one directory: `/data` in Docker, `/var/lib/savvy-go` for the Debian package, `./savvy-data` from source. Back up or move it to move the whole instance.

```
server.sqlite          users, sessions, spaces, members, server settings
spaces/<id>.sqlite     one database per space
keys/server.ed25519    server signing key
backups/               backups made from the UI
uploads/               uploaded files (imports)
```

### Signing key

The server signs backups and cross-space transfers with `keys/server.ed25519`, created on first start. Only a server backup includes it.

> [!IMPORTANT]
> Keep the key with the data. If it is missing while the data has its signatures, Savvy refuses to start instead of creating a new key. Put the file back to continue.

Server admins can see the public key, rotate it (old signatures stay valid) and trust another server's key in **Administration → Security**. Trusting an old server's key lets transfers of spaces moved from it sync again.

## Backups in the app

- **Settings → Backups** (space admins): create, download and restore backups of the space, or import one as a new space. The number of kept backups is set by the server admin.
- **Administration → System backups** (server admins): back up and restore the whole server, including the signing key.

When restoring:

- Backups not made by this server show as *not signed by this server*, but can still be restored.
- Only known tables and columns are copied into a fresh database; anything extra in the file is dropped.
- A space backup restores only over its own space; otherwise import it as a new space. Backups over the space quota are rejected.
- Transfers with linked spaces that changed since the backup come back as pending, see [Spaces](spaces.md#linked-spaces-and-transfers).

A database of the Laravel version is not kept as a backup, see [Upgrading from the Laravel version](#upgrading-from-the-laravel-version). A deleted space leaves a final backup in **Administration → Spaces**.

## Manual backups

> [!WARNING]
> Databases use WAL mode: recent writes may still be in `*.sqlite-wal` files. Copying files of a running instance can lose data. Use in-app backups, or stop Savvy first.

Docker:

```bash
docker compose down
docker run --rm -v savvy-go-data:/data -v "$PWD":/out alpine tar -C /data -czf /out/savvy-$(date +%Y%m%d).tar.gz .
docker compose up -d
```

Debian package:

```bash
sudo systemctl stop savvy-go
sudo tar -C /var/lib/savvy-go -czf savvy-go-$(date +%Y%m%d).tar.gz .
sudo systemctl start savvy-go
```

To restore: stop Savvy, replace the data directory contents with the archive, start it.

## Upgrading from the Laravel version

A database of the Laravel version (a bare `.sqlite` file) is never kept as a backup. Upload it in the backup dialog: Savvy sees that it is a Laravel database, converts it to the current format and restores it at once, as what you choose:

- **The whole server** (**Administration → System backups**, server admins): replaces everything, users and spaces included. The dialog asks for the old `APP_KEY` (from the `.env` of the Laravel install), which decrypts the two-factor secrets. Without it, two-factor sign-in is switched off for the users who had it, so they can still sign in with their password and set it up again.
- **The current space** (**Settings → Backups**, space admins): replaces the data of that space with the old finances. Users are not imported.
- **A new space**: creates a space with the old finances and makes you its admin. Users are not imported.

Roles map as:

| Old role     | Server role | Space role |
|--------------|-------------|------------|
| `admin`      | admin       | admin      |
| `read-write` | user        | editor     |
| `read-only`  | user        | viewer     |
