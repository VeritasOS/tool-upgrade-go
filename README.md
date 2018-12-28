# tool-upgrade-go

A library for golang that helps upgrade tools to newer versions with ease.

## Usage

### Repository structure

```
# over-simplified example of directory and file structure required
# mycompany.com web server directory or artifact repository

# binaries must be suffixed with _${GOOS}_${GOARCH}
% find path/to/dist
path/to/dist/coolTool_darwin_amd64
path/to/dist/coolTool_linux_amd64
path/to/dist/coolTool_windows_amd64

% cat path/to/coolTool/.version-latest
1.2.2

# copy new binaries into place
% mkdir -p path/to/coolTool/1.2.3
% cp path/to/dist/coolTool_* path/to/coolTool/1.2.3

# update version channel
% echo "1.2.3" > path/to/coolTool/.version-latest
```

### Check for updates and upgrade if needed

`tool-upgrade-go` will upgrade to the version available in the version channel.
It leverages `GOOS` and `GOARCH` of the current runtime to determine which
platform architecture to download for the upgrade. In practice, this is often
done using a CLI subcommand.

```
import "github.com/VeritasOS/tool-upgrade-go"

func main() {
  needUpgrade, err := upgrade.CheckAndNotifyIfOutOfDate(
    "coolTool", // tool name
    "1.2.2",    // current (installed) version
    "https://mycompany.com/path/to/coolTool", // artifact repo base
    ".version-", // version channel prefix
    "latest",    // version channel
  )

  if needUpgrade {
    err := upgrade.Upgrade(
      "coolTool",  // tool name
      "1.2.2",     // current (installed) version
      "https://mycompany.com/path/to/coolTool", // artifact repo base
      ".version-", // version channel prefix
      "latest",    // version channel name
      false,       // force upgrade when available version is the same or older
    )
  }
}
```
