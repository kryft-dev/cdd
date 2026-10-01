# cdd

A terminal UI for jumping to recently used projects. It remembers where you have been working and gets you back there in a keystroke.

## Language

**Project**:
A directory holding a git repository (a `.git` directory or file), identified by its absolute path. It may sit at any depth, and a directory that is not a repository is never a Project.
_Avoid_: Repo, workspace, folder, directory

**Jump**:
Changing the shell's working directory to a chosen Project.
_Avoid_: cd, navigate, switch, open, go to

**Visit**:
A single recorded Jump to a Project, or an entry seeded for a Project by a Scan. Changing directory by other means is not tracked.
_Avoid_: Access, hit, entry, usage

**Stale Visit**:
A Visit whose Project no longer has a `.git` at its path, whatever the cause. It contributes nothing to the Picker and is never pruned; it ages out of History.
_Avoid_: Orphan, dead entry, dangling, missing project

**History**:
The ordered record of Visits from which recent Projects are derived.
_Avoid_: Cache, log, database, store

**Scan**:
A one-off sweep of the directories given to it (the home directory by default) that seeds History with a Visit per Project found at any depth below them, dated from evidence found inside the Project itself. Nothing about which directories were swept is remembered.
_Avoid_: Import, index, crawl, rebuild

**Picker**:
The interactive screen that lists Projects and lets the user choose one to Jump to.
_Avoid_: Menu, list, finder, selector

**Query**:
The text typed into the Picker to filter it. Split on spaces into words: the last word matches a Project's name and every earlier word matches, in order, within its parent directory; a query with no space matches the whole shown path. Each word matches fuzzily and tolerates a typo or two by its length.
_Avoid_: Search, filter string, pattern

**Layout**:
One of the arrangements in which the Picker draws Projects. The List Layout, the only one today, draws them in one flat run.
_Avoid_: View, mode, style, theme, skin

**Wrapper**:
The shell function installed into the user's shell that turns a Project chosen in the Picker into a Jump.
_Avoid_: Hook, integration, plugin, shim, alias
