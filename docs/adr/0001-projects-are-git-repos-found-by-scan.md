# Projects are git repositories found by Scan

_Partly superseded by [0002](0002-projects-can-be-added-by-hand.md): a directory added with `cdd add` is a Project too._

Until v0.3.0 cdd searched a configured Root with a fixed shape: every directory under Root was a Kind, every directory under a Kind was a Project, and any directory counted. That shape forced one layout onto the disk and listed folders that were never meant to be jumped to, while repositories nested deeper, or kept outside Root, could not be reached at all.

A Project is now any directory holding a `.git` directory or file, at any depth, and there is no Root or Kind to configure. Only `cdd scan [dir...]` walks the disk (the home directory by default); it seeds History with absolute paths, and the Picker lists History alone, dropping any Visit whose `.git` has gone. The walk stops descending at the first repository it finds, except in a directory named to Scan, so a dotfiles repository at the home directory does not hide everything under it.

## Considered options

- **Walk the home directory each time the Picker opens.** Always fresh, but opening cost grows with the whole home directory; opening the Picker must stay instant.
- **Remember the directories Scan was given.** A plain `cdd scan` could re-walk them, but that is a configured Root under another name.
- **Record any `cd` into a repository through the Wrapper.** Useful, but it changes what counts as a Visit; it can be added later without undoing this decision.

## Consequences

- History lines written before v0.3.0 hold Root-relative paths; they are skipped as malformed and age out, and a fresh Scan rebuilds the order.
- The Grouped Layout drew Kind headers, so it is gone; `picker.layout` stays for layouts still to come.
- A config.toml still setting `root`, or `picker.layout = "grouped"`, is a hard error naming the change, so nobody is left wondering why their Root is ignored.
