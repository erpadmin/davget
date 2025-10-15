# davget

**davget** is a standalone binary for listing and recursively downloading files (akin to `wget`) from WebDAV servers.

This tool was created for use with [Pydio Cells](https://pydio.com/) public links (after appending `/dav/` to the URL). It’s ideal for users who prefer to recursively download files from the CLI rather than the web portal.

---

## Usage

### List files (non-recursive)

```sh
davget -l https://example.com/public/ce3dh435e9ec/dav/
```

### Download a specific file

```sh
davget https://example.com/public/ce3dh435e9ec/dav/file1.txt
```

### Recursively download all files

```sh
davget -r https://example.com/public/ce3dh435e9ec/dav/
```

---
