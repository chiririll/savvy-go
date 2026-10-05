# Data and Backups

## Data directory

Everything Go Savvy keeps is inside one directory: `/data` in Docker, `/var/lib/savvy-go` for the Debian package, `./data` when run from source, or whatever `DATA_DIR` points to. Backing up or moving that directory moves the whole instance.

```
server.sqlite          users, sessions, spaces, members, links, server settings
spaces/<id>.sqlite     one database per space with its finances
keys/server.ed25519    the server's signing key
backups/               backups made from the UI
uploads/               uploaded files (imports)
```

### Signing key

The server signs backups and every transfer between spaces with `keys/server.ed25519`, generated on first start. The private key is never part of a space backup or any API response; only a server backup includes it.

> [!IMPORTANT]
> Keep the key file with the data. If it goes missing while the data still holds its signatures, Savvy exits at startup instead of quietly creating a new key, which would leave every transfer unverifiable. Put the file back to start again.

Server admins see the public key in **Administration → Security**, can rotate it there (older signatures stay valid) and can trust the public key of another server. Trusting the old server's key is what lets transfers of spaces moved from that server sync again.

## Backups in the app

- **Settings → Backups**: space admins create, download and restore backups of the current space, or import a backup as a new space. A space backup is a zip with the space database and a manifest signed by the server. The number of kept backups per space is set by the server admin.
- **Administration → System backups**: server admins back up and restore the whole server: all databases and the signing key.

Restoring never trusts the uploaded file:

- The archive is unpacked with size and name checks, and the manifest signature and file hashes are verified. Backups made elsewhere or edited by hand are shown as *not signed by this server*; they can still be restored.
- A fresh database is built from the current migrations and only known tables and columns are copied from the backup, so triggers, views or extra tables in the file are dropped.
- A space backup can only be restored over the space it was taken from; import it as a new space otherwise. A backup larger than the space quota is rejected.
- Transfers with linked spaces that changed since the backup come back as pending transactions to review; see [Spaces](spaces.md#linked-spaces-and-transfers).

Backups of the Laravel version can be restored as a whole server or imported as a new space.

When a space is deleted, a final backup is kept for server admins in **Administration → Spaces**.

## Manual backups

> [!WARNING]
> The databases run in **WAL mode**, so recent writes may still sit in the `*.sqlite-wal` files and **not in the `.sqlite` files yet**. Copying files of a running instance can silently lose the latest data. Use the in-app backups, or stop Savvy and copy the whole data directory.

Docker (stop the container first so the WAL is flushed):

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

To restore, stop Savvy, replace the contents of the data directory with the archive and start it again.

## Upgrading from the Laravel version

The Laravel version keeps everything in one `database.sqlite`, and a single database file is only accepted from it; Go Savvy itself never writes one. On first start, while the new server has no users yet, Savvy splits it into `server.sqlite` and a first space and renames the old file to `database.sqlite.migrated`. Old roles are mapped like this:

| Old role     | Server role | Role in the first space |
|--------------|-------------|-------------------------|
| `admin`      | admin       | admin                   |
| `read-write` | user        | editor                  |
| `read-only`  | user        | viewer                  |
